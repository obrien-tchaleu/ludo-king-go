// internal/shared/constants/constants.go
package constants

const (
	// Configuration réseau
	DefaultServerPort = "8080"
	MaxPlayers        = 4
	MinPlayers        = 2

	// Configuration du plateau
	BoardSize       = 15
	TotalCells      = 52
	HomeCells       = 6
	SafeCells       = 8
	TokensPerPlayer = 4

	// Règles du jeu
	DiceMin           = 1
	DiceMax           = 6
	RollToStart       = 6
	RollForExtraTurn  = 6
	MaxConsecutiveSix = 3

	// Timeouts
	TurnTimeout      = 30 // secondes
	RollTimeout      = 10 // secondes
	ReconnectTimeout = 60 // secondes

	// Codes d'erreur
	ErrInvalidMove  = "INVALID_MOVE"
	ErrNotYourTurn  = "NOT_YOUR_TURN"
	ErrGameFull     = "GAME_FULL"
	ErrRoomNotFound = "ROOM_NOT_FOUND"
	ErrUnauthorized = "UNAUTHORIZED"
)

// Couleurs des joueurs
type PlayerColor string

const (
	ColorRed    PlayerColor = "red"
	ColorBlue   PlayerColor = "blue"
	ColorGreen  PlayerColor = "green"
	ColorYellow PlayerColor = "yellow"
)

// États du jeu
type GameState string

const (
	StateWaiting  GameState = "waiting"
	StatePlaying  GameState = "playing"
	StateFinished GameState = "finished"
)

// Types de messages réseau
type MessageType string

const (
	// Client -> Serveur
	MsgJoinRoom    MessageType = "JOIN_ROOM"
	MsgCreateRoom  MessageType = "CREATE_ROOM"
	MsgLeaveRoom   MessageType = "LEAVE_ROOM"
	MsgRollDice    MessageType = "ROLL_DICE"
	MsgMoveToken   MessageType = "MOVE_TOKEN"
	MsgChatMessage MessageType = "CHAT_MESSAGE"
	MsgReady       MessageType = "PLAYER_READY"

	// Serveur -> Client
	// Serveur -> Client
	MsgRoomCreated   MessageType = "ROOM_CREATED"
	MsgRoomJoined    MessageType = "ROOM_JOINED" // ✅ AJOUTÉ
	MsgPlayerJoined  MessageType = "PLAYER_JOINED"
	MsgPlayerLeft    MessageType = "PLAYER_LEFT"
	MsgGameStart     MessageType = "GAME_START"
	MsgDiceRolled    MessageType = "DICE_ROLLED"
	MsgTokenMoved    MessageType = "TOKEN_MOVED"
	MsgTokenCaptured MessageType = "TOKEN_CAPTURED"
	MsgTurnChanged   MessageType = "TURN_CHANGED"
	MsgGameOver      MessageType = "GAME_OVER"
	MsgError         MessageType = "ERROR"
	MsgGameState     MessageType = "GAME_STATE"

	// Bidirectionnel
	MsgPing MessageType = "PING"
	MsgPong MessageType = "PONG"
)

// Positions de départ des couleurs
var StartingPositions = map[PlayerColor]int{
	ColorRed:    0,
	ColorBlue:   13,
	ColorGreen:  26,
	ColorYellow: 39,
}

// Positions des zones sécurisées
var SafePositions = []int{0, 8, 13, 21, 26, 34, 39, 47}

// Chemins vers la maison
var HomeStretchStart = map[PlayerColor]int{
	ColorRed:    50,
	ColorBlue:   11,
	ColorGreen:  24,
	ColorYellow: 37,
}
