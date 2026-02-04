// internal/shared/protocol/validator.go
package protocol

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/obrien-tchaleu/ludo-king-go/internal/shared/models"
)

// Validator valide les messages et payloads
type Validator struct{}

// NewValidator crée un nouveau validateur
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateMessage valide un message
func (v *Validator) ValidateMessage(msg *models.NetworkMessage) error {
	if msg == nil {
		return fmt.Errorf("message is nil")
	}

	if msg.Type == "" {
		return fmt.Errorf("message type is empty")
	}

	// Valider selon le type de message
	switch msg.Type {
	case "create_room":
		return v.validateCreateRoom(msg.Payload)
	case "join_room":
		return v.validateJoinRoom(msg.Payload)
	case "connect":
		return v.validateConnect(msg.Payload)
	default:
		// Pas de validation spécifique pour les autres types
		return nil
	}
}

// ExtractPayload extrait et convertit le payload
func ExtractPayload(payload interface{}, target interface{}) error {
	// Convertir le payload en JSON
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Décoder dans la structure cible
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return nil
}

// CreateRoomPayload pour créer une salle
type CreateRoomPayload struct {
	Name       string `json:"name"`
	MaxPlayers int    `json:"max_players"`
	GameMode   string `json:"game_mode"`
	IsPrivate  bool   `json:"is_private"`
	Password   string `json:"password,omitempty"`
	UserID     int64  `json:"user_id"`
	Username   string `json:"username"`
}

// JoinRoomPayload pour rejoindre une salle
type JoinRoomPayload struct {
	RoomID   string `json:"room_id"`
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
}

// ConnectPayload contient les informations de connexion
type ConnectPayload struct {
	Username string `json:"username"`
	Token    string `json:"token,omitempty"`
	Version  string `json:"version"`
}

// validateCreateRoom valide le payload de création de salle
func (v *Validator) validateCreateRoom(payload interface{}) error {
	var data CreateRoomPayload
	if err := ExtractPayload(payload, &data); err != nil {
		return err
	}

	if strings.TrimSpace(data.Name) == "" {
		return fmt.Errorf("room name cannot be empty")
	}

	if data.MaxPlayers < 2 || data.MaxPlayers > 4 {
		return fmt.Errorf("max players must be between 2 and 4")
	}

	if data.Username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	return nil
}

// validateJoinRoom valide le payload de join room
func (v *Validator) validateJoinRoom(payload interface{}) error {
	var data JoinRoomPayload
	if err := ExtractPayload(payload, &data); err != nil {
		return err
	}

	if data.RoomID == "" {
		return fmt.Errorf("room ID cannot be empty")
	}

	if data.Username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	return nil
}

// validateConnect valide le payload de connexion
func (v *Validator) validateConnect(payload interface{}) error {
	var data ConnectPayload
	if err := ExtractPayload(payload, &data); err != nil {
		return err
	}

	if strings.TrimSpace(data.Username) == "" {
		return fmt.Errorf("username cannot be empty")
	}

	if len(data.Username) < 3 || len(data.Username) > 20 {
		return fmt.Errorf("username must be between 3 and 20 characters")
	}

	return nil
}

// ValidateUsername valide un nom d'utilisateur
func ValidateUsername(username string) error {
	username = strings.TrimSpace(username)

	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	if len(username) < 3 {
		return fmt.Errorf("username must be at least 3 characters")
	}

	if len(username) > 20 {
		return fmt.Errorf("username must be at most 20 characters")
	}

	// Vérifier les caractères valides
	for _, char := range username {
		if !isValidUsernameChar(char) {
			return fmt.Errorf("username contains invalid characters")
		}
	}

	return nil
}

// isValidUsernameChar vérifie si un caractère est valide pour un username
func isValidUsernameChar(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '_' || char == '-'
}

// ValidateRoomName valide un nom de salle
func ValidateRoomName(name string) error {
	name = strings.TrimSpace(name)

	if name == "" {
		return fmt.Errorf("room name cannot be empty")
	}

	if len(name) < 3 {
		return fmt.Errorf("room name must be at least 3 characters")
	}

	if len(name) > 50 {
		return fmt.Errorf("room name must be at most 50 characters")
	}

	return nil
}
