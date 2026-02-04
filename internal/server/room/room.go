// internal/server/room/room.go
package room

import (
	"fmt"
	"sync"
	"time"

	"github.com/obrien-tchaleu/ludo-king-go/internal/server/game"
	"github.com/obrien-tchaleu/ludo-king-go/internal/shared/constants"
	"github.com/obrien-tchaleu/ludo-king-go/internal/shared/models"
)

// Room représente une salle de jeu active
type Room struct {
	Model    *models.Room
	Engine   *game.Engine
	players  map[int64]*PlayerConnection
	messages chan *RoomMessage
	mu       sync.RWMutex
	done     chan bool
}

// PlayerConnection représente une connexion de joueur dans la salle
type PlayerConnection struct {
	PlayerID int64
	Username string
	JoinedAt time.Time
	Ready    bool
}

// RoomMessage représente un message dans la salle
type RoomMessage struct {
	Type     string
	PlayerID int64
	Data     interface{}
}

// AddPlayer ajoute un joueur à la salle
func (r *Room) AddPlayer(playerID int64, username string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Vérifier si la salle est pleine
	if len(r.Model.Players) >= r.Model.MaxPlayers {
		return fmt.Errorf("room is full")
	}

	// Vérifier si le joueur est déjà dans la salle
	if _, exists := r.players[playerID]; exists {
		return fmt.Errorf("player already in room")
	}

	// Choisir une couleur disponible
	colors := []constants.PlayerColor{
		constants.ColorRed, constants.ColorBlue,
		constants.ColorGreen, constants.ColorYellow,
	}
	usedColors := make(map[constants.PlayerColor]bool)
	for _, p := range r.Model.Players {
		usedColors[p.Color] = true
	}

	var playerColor constants.PlayerColor
	for _, c := range colors {
		if !usedColors[c] {
			playerColor = c
			break
		}
	}

	// Créer le joueur
	player := models.NewPlayer(playerID, username, playerColor)
	r.Model.Players = append(r.Model.Players, player)

	// Ajouter la connexion
	r.players[playerID] = &PlayerConnection{
		PlayerID: playerID,
		Username: username,
		JoinedAt: time.Now(),
		Ready:    false,
	}

	return nil
}

// RemovePlayer retire un joueur de la salle
func (r *Room) RemovePlayer(playerID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Retirer des connexions
	delete(r.players, playerID)

	// Retirer de la liste des joueurs
	for i, p := range r.Model.Players {
		if p.ID == playerID {
			r.Model.Players = append(r.Model.Players[:i], r.Model.Players[i+1:]...)
			break
		}
	}

	// Si l'hôte quitte, promouvoir un autre joueur
	if r.Model.HostID == playerID && len(r.Model.Players) > 0 {
		r.Model.HostID = r.Model.Players[0].ID
	}
}

// SetPlayerReady marque un joueur comme prêt
func (r *Room) SetPlayerReady(playerID int64, ready bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	conn, exists := r.players[playerID]
	if !exists {
		return fmt.Errorf("player not in room")
	}

	conn.Ready = ready

	// Mettre à jour le modèle du joueur
	for _, p := range r.Model.Players {
		if p.ID == playerID {
			p.IsReady = ready
			break
		}
	}

	return nil
}

// CanStart vérifie si la partie peut démarrer
func (r *Room) CanStart() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.Model.Players) < constants.MinPlayers {
		return false
	}

	// Vérifier que tous les joueurs humains sont prêts
	for _, conn := range r.players {
		if !conn.Ready {
			return false
		}
	}

	return true
}

// Start démarre la partie
func (r *Room) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Model.State != constants.StateWaiting {
		return fmt.Errorf("game already started")
	}

	// Créer le moteur de jeu si pas encore fait
	if r.Engine == nil {
		callbacks := game.EngineCallbacks{
			OnDiceRolled: func(playerID int64, value int, extraTurn bool) {
				r.messages <- &RoomMessage{
					Type:     "dice_rolled",
					PlayerID: playerID,
					Data: map[string]interface{}{
						"dice_value": value,
						"extra_turn": extraTurn,
					},
				}
			},
			OnTokenMoved: func(playerID int64, token *models.Token, from, to int) {
				r.messages <- &RoomMessage{
					Type:     "token_moved",
					PlayerID: playerID,
					Data: map[string]interface{}{
						"token_id": token.ID,
						"from_pos": from,
						"to_pos":   to,
					},
				}
			},
			OnTokenCaptured: func(capturer, victim int64, token *models.Token, pos int) {
				r.messages <- &RoomMessage{
					Type:     "token_captured",
					PlayerID: capturer,
					Data: map[string]interface{}{
						"victim":   victim,
						"token_id": token.ID,
						"position": pos,
					},
				}
			},
			OnTurnChanged: func(playerID int64) {
				r.messages <- &RoomMessage{
					Type:     "turn_changed",
					PlayerID: playerID,
				}
			},
			OnGameOver: func(winner *models.Player, rankings []*models.Player) {
				r.messages <- &RoomMessage{
					Type: "game_over",
					Data: map[string]interface{}{
						"winner":   winner,
						"rankings": rankings,
					},
				}
			},
		}

		r.Engine = game.NewEngine(r.Model, callbacks)
	}

	// Démarrer le moteur
	if err := r.Engine.Start(); err != nil {
		return err
	}

	r.Model.State = constants.StatePlaying
	now := time.Now()
	r.Model.StartedAt = &now

	return nil
}

// IsEmpty vérifie si la salle est vide
func (r *Room) IsEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.players) == 0
}

// GetPlayerCount retourne le nombre de joueurs
func (r *Room) GetPlayerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.players)
}

// Run exécute la boucle principale de la salle
func (r *Room) Run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg := <-r.messages:
			// Traiter le message
			r.handleMessage(msg)
		case <-ticker.C:
			// Vérification périodique
			if r.IsEmpty() {
				return
			}
		case <-r.done:
			return
		}
	}
}

// handleMessage traite un message de la salle
func (r *Room) handleMessage(msg *RoomMessage) {
	// À implémenter : broadcast aux joueurs
}

// Close ferme la salle
func (r *Room) Close() {
	close(r.done)
}
