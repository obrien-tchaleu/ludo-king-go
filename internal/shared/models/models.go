// internal/shared/models/models.go
package models

import (
	"time"

	"github.com/obrien-tchaleu/ludo-king-go/internal/shared/constants"
)

// User représente un utilisateur
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	AvatarURL    string    `json:"avatar_url"`
	Level        int       `json:"level"`
	Experience   int       `json:"experience"`
	Coins        int       `json:"coins"`
	CreatedAt    time.Time `json:"created_at"`
	LastLogin    time.Time `json:"last_login"`
}

// PlayerStats représente les statistiques d'un joueur
type PlayerStats struct {
	UserID         int64   `json:"user_id"`
	TotalGames     int     `json:"total_games"`
	GamesWon       int     `json:"games_won"`
	GamesLost      int     `json:"games_lost"`
	TokensCaptured int     `json:"tokens_captured"`
	TokensLost     int     `json:"tokens_lost"`
	SixesRolled    int     `json:"sixes_rolled"`
	TotalDiceRolls int     `json:"total_dice_rolls"`
	WinRate        float64 `json:"win_rate"`
	HighestStreak  int     `json:"highest_streak"`
	CurrentStreak  int     `json:"current_streak"`
}

// Token représente un pion sur le plateau
type Token struct {
	ID       int                   `json:"id"`
	Color    constants.PlayerColor `json:"color"`
	Position int                   `json:"position"` // -1 = base, 0-51 = plateau, 52-57 = maison
	IsHome   bool                  `json:"is_home"`
	IsSafe   bool                  `json:"is_safe"`
}

// Player représente un joueur dans une partie
type Player struct {
	ID             int64                 `json:"id"`
	Username       string                `json:"username"`
	Color          constants.PlayerColor `json:"color"`
	Tokens         []*Token              `json:"tokens"`
	TokensAtHome   int                   `json:"tokens_at_home"`
	IsAI           bool                  `json:"is_ai"`
	AILevel        string                `json:"ai_level,omitempty"` // easy, medium, hard
	IsReady        bool                  `json:"is_ready"`
	IsConnected    bool                  `json:"is_connected"`
	ConsecutiveSix int                   `json:"consecutive_six"`
}

// Room représente une salle de jeu
type Room struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	HostID      int64               `json:"host_id"`
	Players     []*Player           `json:"players"`
	MaxPlayers  int                 `json:"max_players"`
	GameMode    string              `json:"game_mode"` // online, local, ai
	State       constants.GameState `json:"state"`
	CurrentTurn int                 `json:"current_turn"`
	LastDice    int                 `json:"last_dice"`
	CreatedAt   time.Time           `json:"created_at"`
	StartedAt   *time.Time          `json:"started_at,omitempty"`
	IsPrivate   bool                `json:"is_private"`
	Password    string              `json:"-"`
}

// Game représente l'état complet d'une partie
type Game struct {
	Room        *Room        `json:"room"`
	Board       *Board       `json:"board"`
	TurnHistory []TurnAction `json:"turn_history"`
	StartTime   time.Time    `json:"start_time"`
	Winner      *Player      `json:"winner,omitempty"`
	Rankings    []*Player    `json:"rankings"`
}

// Board représente le plateau de jeu
type Board struct {
	Cells         [52]*Cell                          `json:"cells"`
	HomeStretches map[constants.PlayerColor][6]*Cell `json:"home_stretches"`
}

// Cell représente une case du plateau
type Cell struct {
	Position int    `json:"position"`
	IsSafe   bool   `json:"is_safe"`
	Token    *Token `json:"token,omitempty"`
}

// TurnAction représente une action de tour
type TurnAction struct {
	PlayerID   int64     `json:"player_id"`
	DiceValue  int       `json:"dice_value"`
	TokenMoved *Token    `json:"token_moved,omitempty"`
	FromPos    int       `json:"from_pos"`
	ToPos      int       `json:"to_pos"`
	Captured   *Token    `json:"captured,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// NetworkMessage représente un message réseau
type NetworkMessage struct {
	Type      constants.MessageType `json:"type"`
	Payload   interface{}           `json:"payload"`
	Timestamp time.Time             `json:"timestamp"`
	PlayerID  int64                 `json:"player_id,omitempty"`
	RoomID    string                `json:"room_id,omitempty"`
}

// Payloads spécifiques
type JoinRoomPayload struct {
	RoomID   string `json:"room_id"`
	Password string `json:"password,omitempty"`
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
}

type CreateRoomPayload struct {
	Name       string `json:"name"`
	MaxPlayers int    `json:"max_players"`
	GameMode   string `json:"game_mode"`
	IsPrivate  bool   `json:"is_private"`
	Password   string `json:"password,omitempty"`
	UserID     int64  `json:"user_id"`
	Username   string `json:"username"`
}

type RollDicePayload struct {
	PlayerID int64  `json:"player_id"`
	RoomID   string `json:"room_id"`
}

type MoveTokenPayload struct {
	PlayerID int64  `json:"player_id"`
	RoomID   string `json:"room_id"`
	TokenID  int    `json:"token_id"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type GameStatePayload struct {
	Game *Game `json:"game"`
}

type DiceRolledPayload struct {
	PlayerID  int64 `json:"player_id"`
	DiceValue int   `json:"dice_value"`
	ExtraTurn bool  `json:"extra_turn"`
}

type TokenMovedPayload struct {
	PlayerID   int64 `json:"player_id"`
	TokenID    int   `json:"token_id"`
	FromPos    int   `json:"from_pos"`
	ToPos      int   `json:"to_pos"`
	IsComplete bool  `json:"is_complete"`
}

type TokenCapturedPayload struct {
	CapturedBy   int64 `json:"captured_by"`
	CapturedFrom int64 `json:"captured_from"`
	TokenID      int   `json:"token_id"`
	Position     int   `json:"position"`
}

type GameOverPayload struct {
	Winner   *Player   `json:"winner"`
	Rankings []*Player `json:"rankings"`
	Duration int       `json:"duration_seconds"`
}

// NewPlayer crée un nouveau joueur
func NewPlayer(id int64, username string, color constants.PlayerColor) *Player {
	tokens := make([]*Token, constants.TokensPerPlayer)
	for i := 0; i < constants.TokensPerPlayer; i++ {
		tokens[i] = &Token{
			ID:       i,
			Color:    color,
			Position: -1, // Base
			IsHome:   false,
			IsSafe:   true, // Base est sécurisée
		}
	}

	return &Player{
		ID:             id,
		Username:       username,
		Color:          color,
		Tokens:         tokens,
		TokensAtHome:   0,
		IsAI:           false,
		IsReady:        false,
		IsConnected:    true,
		ConsecutiveSix: 0,
	}
}

// NewAIPlayer crée un joueur IA
func NewAIPlayer(color constants.PlayerColor, level string) *Player {
	player := NewPlayer(0, "AI Player", color)
	player.IsAI = true
	player.AILevel = level
	player.IsReady = true
	return player
}

// NewBoard crée un nouveau plateau
func NewBoard() *Board {
	cells := [52]*Cell{}
	for i := 0; i < 52; i++ {
		cells[i] = &Cell{
			Position: i,
			IsSafe:   contains(constants.SafePositions, i),
			Token:    nil,
		}
	}

	homeStretches := make(map[constants.PlayerColor][6]*Cell)
	for color := range constants.StartingPositions {
		stretch := [6]*Cell{}
		for i := 0; i < 6; i++ {
			stretch[i] = &Cell{
				Position: 52 + i,
				IsSafe:   true,
				Token:    nil,
			}
		}
		homeStretches[color] = stretch
	}

	return &Board{
		Cells:         cells,
		HomeStretches: homeStretches,
	}
}

func contains(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
