// internal/server/game/engine.go
package game

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/obrien-tchaleu/ludo-king-go/internal/shared/constants"
	"github.com/obrien-tchaleu/ludo-king-go/internal/shared/models"
	"github.com/obrien-tchaleu/ludo-king-go/pkg/ai"
)

// Engine gère la logique du jeu
type Engine struct {
	game      *models.Game
	ai        map[int64]*ai.AIPlayer // IA par joueur
	mu        sync.RWMutex
	rand      *rand.Rand
	turnTimer *time.Timer
	callbacks EngineCallbacks
	rollCount map[int64]int // Compte les lancers par joueur
}

// EngineCallbacks définit les callbacks pour les événements du jeu
type EngineCallbacks struct {
	OnDiceRolled    func(playerID int64, value int, extraTurn bool)
	OnTokenMoved    func(playerID int64, token *models.Token, from, to int)
	OnTokenCaptured func(capturer, victim int64, token *models.Token, pos int)
	OnTurnChanged   func(playerID int64)
	OnGameOver      func(winner *models.Player, rankings []*models.Player)
}

// NewEngine crée un nouveau moteur de jeu
func NewEngine(room *models.Room, callbacks EngineCallbacks) *Engine {
	board := models.NewBoard()

	engine := &Engine{
		game: &models.Game{
			Room:        room,
			Board:       board,
			TurnHistory: make([]models.TurnAction, 0),
			StartTime:   time.Now(),
			Rankings:    make([]*models.Player, 0),
		},
		ai:        make(map[int64]*ai.AIPlayer),
		rand:      rand.New(rand.NewSource(time.Now().UnixNano())),
		callbacks: callbacks,
		rollCount: make(map[int64]int),
	}

	// Initialiser les IA si nécessaire
	for _, player := range room.Players {
		if player.IsAI {
			engine.ai[player.ID] = ai.NewAIPlayer(player.AILevel)
		}
		engine.rollCount[player.ID] = 0
	}

	return engine
}

// Start démarre la partie
func (e *Engine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.game.Room.State != constants.StateWaiting {
		return fmt.Errorf("game already started")
	}

	// Vérifier le nombre de joueurs
	if len(e.game.Room.Players) < constants.MinPlayers {
		return fmt.Errorf("not enough players")
	}

	// Choisir un joueur aléatoire pour commencer
	e.game.Room.CurrentTurn = e.rand.Intn(len(e.game.Room.Players))
	e.game.Room.State = constants.StatePlaying
	now := time.Now()
	e.game.Room.StartedAt = &now

	// Notifier le premier joueur
	currentPlayer := e.game.Room.Players[e.game.Room.CurrentTurn]
	if e.callbacks.OnTurnChanged != nil {
		e.callbacks.OnTurnChanged(currentPlayer.ID)
	}

	// Si c'est une IA, lancer automatiquement
	if currentPlayer.IsAI {
		go e.handleAITurn(currentPlayer)
	} else {
		e.startTurnTimer(currentPlayer.ID)
	}

	return nil
}

// RollDice lance le dé pour un joueur (avec système de dés truqués)
func (e *Engine) RollDice(playerID int64) (int, bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Vérifier que c'est le tour du joueur
	currentPlayer := e.game.Room.Players[e.game.Room.CurrentTurn]
	if currentPlayer.ID != playerID {
		return 0, false, fmt.Errorf(constants.ErrNotYourTurn)
	}

	// Incrémenter le compteur de lancers pour ce joueur
	e.rollCount[playerID]++
	rollNumber := e.rollCount[playerID]

	var diceValue int

	// 🎲 SYSTÈME DE DÉS TRUQUÉS
	// Premier lancer OU tous les 5 lancers = 6 automatique
	if rollNumber == 1 || rollNumber%5 == 0 {
		diceValue = 6
	} else {
		// Lancer normal
		diceValue = e.rand.Intn(constants.DiceMax) + constants.DiceMin
	}

	e.game.Room.LastDice = diceValue

	// Vérifier les 6 consécutifs (règle des 3 six)
	extraTurn := false
	if diceValue == constants.RollForExtraTurn {
		currentPlayer.ConsecutiveSix++
		if currentPlayer.ConsecutiveSix >= constants.MaxConsecutiveSix {
			// Perdre le tour après 3 six consécutifs
			currentPlayer.ConsecutiveSix = 0
			e.nextTurn()
			if e.callbacks.OnDiceRolled != nil {
				e.callbacks.OnDiceRolled(playerID, diceValue, false)
			}
			return diceValue, false, nil
		}
		extraTurn = true
	} else {
		currentPlayer.ConsecutiveSix = 0
	}

	// Vérifier si le joueur peut jouer
	canMove := e.hasValidMove(currentPlayer, diceValue)
	if !canMove {
		// Pas de mouvement possible, tour suivant
		if !extraTurn {
			e.nextTurn()
		}
	}

	if e.callbacks.OnDiceRolled != nil {
		e.callbacks.OnDiceRolled(playerID, diceValue, extraTurn)
	}

	return diceValue, extraTurn, nil
}

// MoveToken déplace un token
func (e *Engine) MoveToken(playerID int64, tokenID int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	currentPlayer := e.game.Room.Players[e.game.Room.CurrentTurn]
	if currentPlayer.ID != playerID {
		return fmt.Errorf(constants.ErrNotYourTurn)
	}

	if tokenID < 0 || tokenID >= len(currentPlayer.Tokens) {
		return fmt.Errorf("invalid token id")
	}

	token := currentPlayer.Tokens[tokenID]
	diceValue := e.game.Room.LastDice

	// Valider le mouvement
	if !e.canMoveToken(token, diceValue, currentPlayer.Color) {
		return fmt.Errorf(constants.ErrInvalidMove)
	}

	oldPos := token.Position
	newPos := e.calculateNewPosition(token, diceValue, currentPlayer.Color)

	// Effectuer le déplacement
	e.moveTokenToPosition(token, newPos, currentPlayer.Color)

	// Vérifier capture
	captured := e.checkCapture(newPos, currentPlayer)

	// Enregistrer l'action
	action := models.TurnAction{
		PlayerID:   playerID,
		DiceValue:  diceValue,
		TokenMoved: token,
		FromPos:    oldPos,
		ToPos:      newPos,
		Captured:   captured,
		Timestamp:  time.Now(),
	}
	e.game.TurnHistory = append(e.game.TurnHistory, action)

	// Notifier
	if e.callbacks.OnTokenMoved != nil {
		e.callbacks.OnTokenMoved(playerID, token, oldPos, newPos)
	}

	if captured != nil && e.callbacks.OnTokenCaptured != nil {
		// Trouver le joueur propriétaire du token capturé
		var victimPlayerID int64
		for _, p := range e.game.Room.Players {
			if p.Color == captured.Color {
				victimPlayerID = p.ID
				break
			}
		}
		e.callbacks.OnTokenCaptured(playerID, victimPlayerID, captured, newPos)
	}

	// Vérifier victoire
	if e.checkWin(currentPlayer) {
		e.endGame(currentPlayer)
		return nil
	}

	// Tour suivant si pas de 6
	if diceValue != constants.RollForExtraTurn {
		e.nextTurn()
	}

	return nil
}

// hasValidMove vérifie si le joueur a un mouvement valide
func (e *Engine) hasValidMove(player *models.Player, diceValue int) bool {
	for _, token := range player.Tokens {
		if e.canMoveToken(token, diceValue, player.Color) {
			return true
		}
	}
	return false
}

// canMoveToken vérifie si un token peut bouger
func (e *Engine) canMoveToken(token *models.Token, diceValue int, color constants.PlayerColor) bool {
	if token.IsHome {
		return false
	}

	if token.Position == -1 && diceValue != constants.RollToStart {
		return false
	}

	newPos := e.calculateNewPosition(token, diceValue, color)

	// Vérifier dépassement
	if newPos > 57 {
		return false
	}

	// Vérifier collision avec son propre token
	if newPos >= 52 {
		homeIdx := newPos - 52
		if e.game.Board.HomeStretches[color][homeIdx].Token != nil {
			return false
		}
	} else {
		cell := e.game.Board.Cells[newPos]
		if cell.Token != nil && cell.Token.Color == color {
			return false
		}
	}

	return true
}

// calculateNewPosition calcule la nouvelle position
func (e *Engine) calculateNewPosition(token *models.Token, diceValue int, color constants.PlayerColor) int {
	if token.Position == -1 {
		return constants.StartingPositions[color]
	}

	newPos := token.Position + diceValue
	homeEntry := constants.HomeStretchStart[color]

	// Vérifier entrée dans la zone maison
	if token.Position < homeEntry && newPos >= homeEntry {
		overflow := newPos - homeEntry
		return 52 + overflow
	}

	// Boucler sur le plateau
	if newPos >= 52 && token.Position < 52 {
		newPos = newPos % 52
	}

	return newPos
}

// moveTokenToPosition déplace effectivement le token
func (e *Engine) moveTokenToPosition(token *models.Token, newPos int, color constants.PlayerColor) {
	// Retirer de l'ancienne position
	if token.Position >= 0 && token.Position < 52 {
		e.game.Board.Cells[token.Position].Token = nil
	} else if token.Position >= 52 {
		homeIdx := token.Position - 52
		e.game.Board.HomeStretches[color][homeIdx].Token = nil
	}

	// Placer à la nouvelle position
	token.Position = newPos
	if newPos >= 52 {
		homeIdx := newPos - 52
		if homeIdx >= 6 {
			token.IsHome = true
		} else {
			e.game.Board.HomeStretches[color][homeIdx].Token = token
		}
	} else {
		e.game.Board.Cells[newPos].Token = token
		token.IsSafe = e.game.Board.Cells[newPos].IsSafe
	}
}

// checkCapture vérifie et effectue une capture
func (e *Engine) checkCapture(pos int, capturer *models.Player) *models.Token {
	if pos < 0 || pos >= 52 {
		return nil
	}

	cell := e.game.Board.Cells[pos]
	if cell.Token == nil || cell.IsSafe {
		return nil
	}

	victim := cell.Token
	if victim.Color == capturer.Color {
		return nil
	}

	// Capturer le token
	victim.Position = -1
	victim.IsHome = false
	victim.IsSafe = true
	cell.Token = nil

	return victim
}

// checkWin vérifie si le joueur a gagné
func (e *Engine) checkWin(player *models.Player) bool {
	for _, token := range player.Tokens {
		if !token.IsHome {
			return false
		}
	}
	player.TokensAtHome = constants.TokensPerPlayer
	return true
}

// nextTurn passe au tour suivant
func (e *Engine) nextTurn() {
	e.game.Room.CurrentTurn = (e.game.Room.CurrentTurn + 1) % len(e.game.Room.Players)
	currentPlayer := e.game.Room.Players[e.game.Room.CurrentTurn]

	if e.callbacks.OnTurnChanged != nil {
		e.callbacks.OnTurnChanged(currentPlayer.ID)
	}

	if currentPlayer.IsAI {
		go e.handleAITurn(currentPlayer)
	} else {
		e.startTurnTimer(currentPlayer.ID)
	}
}

// handleAITurn gère le tour d'une IA
func (e *Engine) handleAITurn(player *models.Player) {
	aiPlayer := e.ai[player.ID]

	// Lancer le dé
	diceValue, extraTurn, _ := e.RollDice(player.ID)

	// Sélectionner et déplacer un token
	time.Sleep(500 * time.Millisecond) // Petit délai

	token := aiPlayer.SelectToken(player, diceValue, e.game.Board)
	if token != nil {
		e.MoveToken(player.ID, token.ID)
	} else if !extraTurn {
		e.mu.Lock()
		e.nextTurn()
		e.mu.Unlock()
	}
}

// startTurnTimer démarre le timer du tour
func (e *Engine) startTurnTimer(playerID int64) {
	if e.turnTimer != nil {
		e.turnTimer.Stop()
	}

	e.turnTimer = time.AfterFunc(time.Duration(constants.TurnTimeout)*time.Second, func() {
		e.mu.Lock()
		defer e.mu.Unlock()

		currentPlayer := e.game.Room.Players[e.game.Room.CurrentTurn]
		if currentPlayer.ID == playerID {
			// Timeout: passer au tour suivant
			e.nextTurn()
		}
	})
}

// endGame termine la partie
func (e *Engine) endGame(winner *models.Player) {
	e.game.Winner = winner
	e.game.Room.State = constants.StateFinished

	// Calculer les classements
	rankings := make([]*models.Player, 0, len(e.game.Room.Players))
	rankings = append(rankings, winner)

	for _, player := range e.game.Room.Players {
		if player.ID != winner.ID {
			rankings = append(rankings, player)
		}
	}

	e.game.Rankings = rankings

	if e.callbacks.OnGameOver != nil {
		e.callbacks.OnGameOver(winner, rankings)
	}
}

// GetGameState retourne l'état actuel du jeu
func (e *Engine) GetGameState() *models.Game {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.game
}
