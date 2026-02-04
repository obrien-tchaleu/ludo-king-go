// internal/server/room/manager.go
package room

import (
	"fmt"
	"sync"
	"time"

	"github.com/obrien-tchaleu/ludo-king-go/internal/shared/constants"
	"github.com/obrien-tchaleu/ludo-king-go/internal/shared/models"
)

// Manager gère toutes les salles de jeu
type Manager struct {
	rooms map[string]*Room
	mu    sync.RWMutex
}

// NewManager crée un nouveau gestionnaire de salles
func NewManager() *Manager {
	return &Manager{
		rooms: make(map[string]*Room),
	}
}

// CreateRoom crée une nouvelle salle
func (m *Manager) CreateRoom(name string, hostID int64, hostName string, maxPlayers int, gameMode string, isPrivate bool) (*Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Générer un ID unique
	roomID := generateRoomID()

	// Créer la room model
	roomModel := &models.Room{
		ID:         roomID,
		Name:       name,
		HostID:     hostID,
		Players:    make([]*models.Player, 0),
		MaxPlayers: maxPlayers,
		GameMode:   gameMode,
		State:      constants.StateWaiting,
		CreatedAt:  time.Now(),
		IsPrivate:  isPrivate,
	}

	// Créer le joueur hôte
	hostPlayer := models.NewPlayer(hostID, hostName, constants.ColorRed)
	roomModel.Players = append(roomModel.Players, hostPlayer)

	// Créer la Room wrapper
	room := &Room{
		Model:    roomModel,
		players:  make(map[int64]*PlayerConnection),
		messages: make(chan *RoomMessage, 100),
	}

	// Ajouter l'hôte
	room.players[hostID] = &PlayerConnection{
		PlayerID: hostID,
		Username: hostName,
		JoinedAt: time.Now(),
	}

	// Enregistrer la salle
	m.rooms[roomID] = room

	// Démarrer le processus de la salle
	go room.Run()

	return room, nil
}

// GetRoom récupère une salle par son ID
func (m *Manager) GetRoom(roomID string) (*Room, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	room, exists := m.rooms[roomID]
	if !exists {
		return nil, fmt.Errorf("room not found")
	}

	return room, nil
}

// JoinRoom permet à un joueur de rejoindre une salle
func (m *Manager) JoinRoom(roomID string, playerID int64, username string) (*Room, error) {
	room, err := m.GetRoom(roomID)
	if err != nil {
		return nil, err
	}

	if err := room.AddPlayer(playerID, username); err != nil {
		return nil, err
	}

	return room, nil
}

// LeaveRoom permet à un joueur de quitter une salle
func (m *Manager) LeaveRoom(roomID string, playerID int64) error {
	room, err := m.GetRoom(roomID)
	if err != nil {
		return err
	}

	room.RemovePlayer(playerID)

	// Si la salle est vide, la supprimer
	if room.IsEmpty() {
		m.mu.Lock()
		delete(m.rooms, roomID)
		m.mu.Unlock()
	}

	return nil
}

// ListRooms retourne la liste des salles publiques disponibles
func (m *Manager) ListRooms() []*models.Room {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rooms := make([]*models.Room, 0)
	for _, room := range m.rooms {
		if !room.Model.IsPrivate && room.Model.State == constants.StateWaiting {
			rooms = append(rooms, room.Model)
		}
	}

	return rooms
}

// GetRoomCount retourne le nombre total de salles
func (m *Manager) GetRoomCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.rooms)
}

// CleanupEmptyRooms supprime les salles vides
func (m *Manager) CleanupEmptyRooms() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, room := range m.rooms {
		if room.IsEmpty() {
			delete(m.rooms, id)
		}
	}
}

// generateRoomID génère un ID unique pour une salle
func generateRoomID() string {
	return fmt.Sprintf("ROOM_%d", time.Now().UnixNano())
}
