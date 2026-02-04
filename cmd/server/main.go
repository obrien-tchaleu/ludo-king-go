// cmd/server/main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/obrien-tchaleu/ludo-king-go/internal/server/game"
	"github.com/obrien-tchaleu/ludo-king-go/internal/shared/constants"
	"github.com/obrien-tchaleu/ludo-king-go/internal/shared/models"
	"github.com/obrien-tchaleu/ludo-king-go/pkg/database"
)

// Config représente la configuration du serveur
type Config struct {
	Server struct {
		Host           string `yaml:"host"`
		Port           string `yaml:"port"`
		MaxConnections int    `yaml:"max_connections"`
	} `yaml:"server"`
	Database struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
	} `yaml:"database"`
	Game struct {
		MaxPlayersPerRoom int `yaml:"max_players_per_room"`
		MinPlayersPerRoom int `yaml:"min_players_per_room"`
		TurnTimeout       int `yaml:"turn_timeout"`
		ReconnectTimeout  int `yaml:"reconnect_timeout"`
	} `yaml:"game"`
	Logging struct {
		Level string `yaml:"level"`
		File  string `yaml:"file"`
	} `yaml:"logging"`
}

// Server représente le serveur de jeu
type Server struct {
	listener    net.Listener
	clients     map[int64]*Client
	rooms       map[string]*GameRoom
	db          *database.DB
	mu          sync.RWMutex
	matchmaking *MatchmakingQueue
	config      *Config
}

// Client représente un client connecté
type Client struct {
	conn     net.Conn
	userID   int64
	username string
	roomID   string
	send     chan *models.NetworkMessage
}

// GameRoom représente une salle avec son moteur
type GameRoom struct {
	room    *models.Room
	engine  *game.Engine
	clients map[int64]*Client
	mu      sync.RWMutex
}

// MatchmakingQueue gère le matchmaking
type MatchmakingQueue struct {
	waiting []*Client
	mu      sync.Mutex
}

func main() {
	// Charger la configuration
	config, err := loadConfig("configs/server.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connexion à la base de données
	db, err := database.NewDB(
		config.Database.Host,
		config.Database.Port,
		config.Database.Username,
		config.Database.Password,
		config.Database.Database,
	)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Printf("✅ Connected to database successfully")

	// Créer le serveur
	server := &Server{
		clients:     make(map[int64]*Client),
		rooms:       make(map[string]*GameRoom),
		db:          db,
		matchmaking: &MatchmakingQueue{waiting: make([]*Client, 0)},
		config:      config,
	}

	// Démarrer le serveur TCP
	listener, err := net.Listen("tcp", ":"+config.Server.Port)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()

	server.listener = listener
	log.Printf("🎲 Ludo King Server started on port %s", config.Server.Port)

	// Démarrer le matchmaking automatique
	go server.processMatchmaking()

	// Accepter les connexions
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go server.handleConnection(conn)
	}
}

// loadConfig charge la configuration depuis un fichier YAML
func loadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return &config, nil
}

// handleConnection gère une nouvelle connexion
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	log.Printf("New connection from %s", conn.RemoteAddr())

	client := &Client{
		conn: conn,
		send: make(chan *models.NetworkMessage, 256),
	}

	// Goroutine pour envoyer les messages
	go s.writeMessages(client)

	// Lire les messages
	decoder := json.NewDecoder(conn)
	for {
		var msg models.NetworkMessage
		if err := decoder.Decode(&msg); err != nil {
			log.Printf("Client disconnected: %v", err)
			s.handleDisconnect(client)
			return
		}

		s.handleMessage(client, &msg)
	}
}

// writeMessages envoie les messages au client
func (s *Server) writeMessages(client *Client) {
	encoder := json.NewEncoder(client.conn)
	for msg := range client.send {
		if err := encoder.Encode(msg); err != nil {
			log.Printf("Failed to send message: %v", err)
			return
		}
	}
}

// handleMessage traite un message reçu
func (s *Server) handleMessage(client *Client, msg *models.NetworkMessage) {
	switch msg.Type {
	case constants.MsgCreateRoom:
		s.handleCreateRoom(client, msg)
	case constants.MsgJoinRoom:
		s.handleJoinRoom(client, msg)
	case constants.MsgLeaveRoom:
		s.handleLeaveRoom(client, msg)
	case constants.MsgRollDice:
		s.handleRollDice(client, msg)
	case constants.MsgMoveToken:
		s.handleMoveToken(client, msg)
	case constants.MsgReady:
		s.handlePlayerReady(client, msg)
	case constants.MsgPing:
		s.sendMessage(client, &models.NetworkMessage{
			Type:      constants.MsgPong,
			Timestamp: time.Now(),
		})
	}
}

// handleCreateRoom crée une nouvelle salle
func (s *Server) handleCreateRoom(client *Client, msg *models.NetworkMessage) {
	payload := msg.Payload.(map[string]interface{})

	// Générer un ID unique
	roomID := generateRoomID()

	// Créer la salle
	room := &models.Room{
		ID:         roomID,
		Name:       payload["name"].(string),
		HostID:     int64(payload["user_id"].(float64)),
		Players:    make([]*models.Player, 0, constants.MaxPlayers),
		MaxPlayers: int(payload["max_players"].(float64)),
		GameMode:   payload["game_mode"].(string),
		State:      constants.StateWaiting,
		CreatedAt:  time.Now(),
		IsPrivate:  payload["is_private"].(bool),
	}

	client.userID = room.HostID
	client.username = payload["username"].(string)
	client.roomID = roomID

	// Créer le joueur hôte
	player := models.NewPlayer(client.userID, client.username, constants.ColorRed)
	room.Players = append(room.Players, player)

	// Créer le moteur de jeu
	gameRoom := &GameRoom{
		room:    room,
		clients: make(map[int64]*Client),
	}
	gameRoom.clients[client.userID] = client

	// Callbacks du moteur
	callbacks := game.EngineCallbacks{
		OnDiceRolled: func(playerID int64, value int, extraTurn bool) {
			s.broadcastToRoom(roomID, &models.NetworkMessage{
				Type: constants.MsgDiceRolled,
				Payload: models.DiceRolledPayload{
					PlayerID:  playerID,
					DiceValue: value,
					ExtraTurn: extraTurn,
				},
				Timestamp: time.Now(),
			})
		},
		OnTokenMoved: func(playerID int64, token *models.Token, from, to int) {
			s.broadcastToRoom(roomID, &models.NetworkMessage{
				Type: constants.MsgTokenMoved,
				Payload: models.TokenMovedPayload{
					PlayerID: playerID,
					TokenID:  token.ID,
					FromPos:  from,
					ToPos:    to,
				},
				Timestamp: time.Now(),
			})
		},
		OnTokenCaptured: func(capturer, victim int64, token *models.Token, pos int) {
			s.broadcastToRoom(roomID, &models.NetworkMessage{
				Type: constants.MsgTokenCaptured,
				Payload: models.TokenCapturedPayload{
					CapturedBy:   capturer,
					CapturedFrom: victim,
					TokenID:      token.ID,
					Position:     pos,
				},
				Timestamp: time.Now(),
			})
		},
		OnTurnChanged: func(playerID int64) {
			s.broadcastToRoom(roomID, &models.NetworkMessage{
				Type:      constants.MsgTurnChanged,
				Payload:   map[string]interface{}{"player_id": playerID},
				Timestamp: time.Now(),
			})
		},
		OnGameOver: func(winner *models.Player, rankings []*models.Player) {
			s.handleGameOver(roomID, winner, rankings)
		},
	}

	gameRoom.engine = game.NewEngine(room, callbacks)

	// Enregistrer la salle
	s.mu.Lock()
	s.rooms[roomID] = gameRoom
	s.clients[client.userID] = client
	s.mu.Unlock()

	// Envoyer la confirmation
	s.sendMessage(client, &models.NetworkMessage{
		Type: constants.MsgRoomCreated,
		Payload: map[string]interface{}{
			"room_id": roomID,
			"room":    room,
		},
		Timestamp: time.Now(),
	})

	log.Printf("Room created: %s by %s", roomID, client.username)
}

// handleJoinRoom permet à un joueur de rejoindre une salle
func (s *Server) handleJoinRoom(client *Client, msg *models.NetworkMessage) {
	payload := msg.Payload.(map[string]interface{})
	roomID := payload["room_id"].(string)

	s.mu.RLock()
	gameRoom, exists := s.rooms[roomID]
	s.mu.RUnlock()

	if !exists {
		s.sendError(client, constants.ErrRoomNotFound, "Room not found")
		return
	}

	gameRoom.mu.Lock()
	defer gameRoom.mu.Unlock()

	if len(gameRoom.room.Players) >= gameRoom.room.MaxPlayers {
		s.sendError(client, constants.ErrGameFull, "Room is full")
		return
	}

	// Choisir une couleur disponible
	colors := []constants.PlayerColor{
		constants.ColorRed, constants.ColorBlue,
		constants.ColorGreen, constants.ColorYellow,
	}
	usedColors := make(map[constants.PlayerColor]bool)
	for _, p := range gameRoom.room.Players {
		usedColors[p.Color] = true
	}

	var playerColor constants.PlayerColor
	for _, c := range colors {
		if !usedColors[c] {
			playerColor = c
			break
		}
	}

	client.userID = int64(payload["user_id"].(float64))
	client.username = payload["username"].(string)
	client.roomID = roomID

	player := models.NewPlayer(client.userID, client.username, playerColor)
	gameRoom.room.Players = append(gameRoom.room.Players, player)
	gameRoom.clients[client.userID] = client

	s.mu.Lock()
	s.clients[client.userID] = client
	s.mu.Unlock()

	// Notifier tous les joueurs
	s.broadcastToRoom(roomID, &models.NetworkMessage{
		Type:      constants.MsgPlayerJoined,
		Payload:   map[string]interface{}{"player": player},
		Timestamp: time.Now(),
	})

	// Envoyer l'état du jeu au nouveau joueur
	s.sendMessage(client, &models.NetworkMessage{
		Type: constants.MsgGameState,
		Payload: models.GameStatePayload{
			Game: gameRoom.engine.GetGameState(),
		},
		Timestamp: time.Now(),
	})

	log.Printf("%s joined room %s", client.username, roomID)
}

// handleRollDice traite un lancer de dé
func (s *Server) handleRollDice(client *Client, msg *models.NetworkMessage) {
	s.mu.RLock()
	gameRoom := s.rooms[client.roomID]
	s.mu.RUnlock()

	if gameRoom == nil {
		return
	}

	gameRoom.engine.RollDice(client.userID)
}

// handleMoveToken traite un déplacement de token
func (s *Server) handleMoveToken(client *Client, msg *models.NetworkMessage) {
	payload := msg.Payload.(map[string]interface{})
	tokenID := int(payload["token_id"].(float64))

	s.mu.RLock()
	gameRoom := s.rooms[client.roomID]
	s.mu.RUnlock()

	if gameRoom == nil {
		return
	}

	err := gameRoom.engine.MoveToken(client.userID, tokenID)
	if err != nil {
		s.sendError(client, constants.ErrInvalidMove, err.Error())
	}
}

// handlePlayerReady marque un joueur comme prêt
func (s *Server) handlePlayerReady(client *Client, msg *models.NetworkMessage) {
	s.mu.RLock()
	gameRoom := s.rooms[client.roomID]
	s.mu.RUnlock()

	if gameRoom == nil {
		return
	}

	gameRoom.mu.Lock()
	defer gameRoom.mu.Unlock()

	for _, player := range gameRoom.room.Players {
		if player.ID == client.userID {
			player.IsReady = true
			break
		}
	}

	// Vérifier si tous sont prêts
	allReady := true
	for _, player := range gameRoom.room.Players {
		if !player.IsReady && !player.IsAI {
			allReady = false
			break
		}
	}

	if allReady && len(gameRoom.room.Players) >= constants.MinPlayers {
		gameRoom.engine.Start()
		s.broadcastToRoom(client.roomID, &models.NetworkMessage{
			Type:      constants.MsgGameStart,
			Timestamp: time.Now(),
		})
	}
}

// broadcastToRoom envoie un message à tous les joueurs d'une salle
func (s *Server) broadcastToRoom(roomID string, msg *models.NetworkMessage) {
	s.mu.RLock()
	gameRoom := s.rooms[roomID]
	s.mu.RUnlock()

	if gameRoom == nil {
		return
	}

	gameRoom.mu.RLock()
	defer gameRoom.mu.RUnlock()

	for _, client := range gameRoom.clients {
		select {
		case client.send <- msg:
		default:
			log.Printf("Failed to send to client %d", client.userID)
		}
	}
}

// sendMessage envoie un message à un client
func (s *Server) sendMessage(client *Client, msg *models.NetworkMessage) {
	select {
	case client.send <- msg:
	default:
		log.Printf("Failed to send message to client")
	}
}

// sendError envoie une erreur au client
func (s *Server) sendError(client *Client, code, message string) {
	s.sendMessage(client, &models.NetworkMessage{
		Type: constants.MsgError,
		Payload: models.ErrorPayload{
			Code:    code,
			Message: message,
		},
		Timestamp: time.Now(),
	})
}

// handleDisconnect gère la déconnexion d'un client
func (s *Server) handleDisconnect(client *Client) {
	s.mu.Lock()
	delete(s.clients, client.userID)
	s.mu.Unlock()

	if client.roomID != "" {
		s.handleLeaveRoom(client, nil)
	}

	close(client.send)
}

// handleLeaveRoom gère la sortie d'une salle
func (s *Server) handleLeaveRoom(client *Client, msg *models.NetworkMessage) {
	// Implementation similaire...
}

// handleGameOver gère la fin de partie
func (s *Server) handleGameOver(roomID string, winner *models.Player, rankings []*models.Player) {
	s.mu.RLock()
	gameRoom := s.rooms[roomID]
	s.mu.RUnlock()

	if gameRoom == nil {
		return
	}

	// Sauvegarder en base de données
	go func() {
		game := gameRoom.engine.GetGameState()
		if err := s.db.SaveGameHistory(game); err != nil {
			log.Printf("Failed to save game: %v", err)
		}

		// Mettre à jour les stats
		for _, player := range game.Room.Players {
			if player.IsAI {
				continue
			}
			won := player.ID == winner.ID
			s.db.UpdatePlayerStats(player.ID, won, 0, 0)
		}
	}()

	// Notifier les joueurs
	s.broadcastToRoom(roomID, &models.NetworkMessage{
		Type: constants.MsgGameOver,
		Payload: models.GameOverPayload{
			Winner:   winner,
			Rankings: rankings,
			Duration: int(time.Since(gameRoom.engine.GetGameState().StartTime).Seconds()),
		},
		Timestamp: time.Now(),
	})
}

// processMatchmaking traite le matchmaking automatique
func (s *Server) processMatchmaking() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.matchmaking.mu.Lock()
		if len(s.matchmaking.waiting) >= constants.MinPlayers {
			// Créer une partie automatiquement
			// Implementation...
		}
		s.matchmaking.mu.Unlock()
	}
}

// generateRoomID génère un ID de salle unique
func generateRoomID() string {
	return fmt.Sprintf("ROOM_%d", time.Now().UnixNano())
}
