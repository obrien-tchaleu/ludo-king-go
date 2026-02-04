// pkg/ai/ai.go
package ai

import (
	"math/rand"
	"time"

	"github.com/obrien-tchaleu/ludo-king-go/internal/shared/constants"
	"github.com/obrien-tchaleu/ludo-king-go/internal/shared/models"
)

// AIPlayer représente un joueur IA
type AIPlayer struct {
	Level      string // easy, medium, hard
	ThinkDelay time.Duration
	rand       *rand.Rand
}

// NewAIPlayer crée une nouvelle IA
func NewAIPlayer(level string) *AIPlayer {
	var thinkDelay time.Duration
	switch level {
	case "easy":
		thinkDelay = 2 * time.Second
	case "medium":
		thinkDelay = 1500 * time.Millisecond
	case "hard":
		thinkDelay = 1000 * time.Millisecond
	default:
		thinkDelay = 1500 * time.Millisecond
	}

	return &AIPlayer{
		Level:      level,
		ThinkDelay: thinkDelay,
		rand:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SelectToken sélectionne le meilleur token à déplacer
func (ai *AIPlayer) SelectToken(player *models.Player, diceValue int, board *models.Board) *models.Token {
	// Simuler la réflexion
	time.Sleep(ai.ThinkDelay)

	switch ai.Level {
	case "easy":
		return ai.selectTokenEasy(player, diceValue, board)
	case "medium":
		return ai.selectTokenMedium(player, diceValue, board)
	case "hard":
		return ai.selectTokenHard(player, diceValue, board)
	default:
		return ai.selectTokenMedium(player, diceValue, board)
	}
}

// selectTokenEasy - IA facile: joue aléatoirement
func (ai *AIPlayer) selectTokenEasy(player *models.Player, diceValue int, board *models.Board) *models.Token {
	validTokens := ai.getValidTokens(player, diceValue, board)
	if len(validTokens) == 0 {
		return nil
	}
	return validTokens[ai.rand.Intn(len(validTokens))]
}

// selectTokenMedium - IA moyenne: priorité aux captures et avancement
func (ai *AIPlayer) selectTokenMedium(player *models.Player, diceValue int, board *models.Board) *models.Token {
	validTokens := ai.getValidTokens(player, diceValue, board)
	if len(validTokens) == 0 {
		return nil
	}

	// 1. Priorité: Token qui peut capturer
	for _, token := range validTokens {
		newPos := ai.calculateNewPosition(token, diceValue, player.Color)
		if ai.canCapture(newPos, player.Color, board) {
			return token
		}
	}

	// 2. Sortir un token de la base si possible
	if diceValue == constants.RollToStart {
		for _, token := range validTokens {
			if token.Position == -1 {
				return token
			}
		}
	}

	// 3. Token le plus avancé
	var bestToken *models.Token
	maxPos := -1
	for _, token := range validTokens {
		if token.Position > maxPos {
			maxPos = token.Position
			bestToken = token
		}
	}

	return bestToken
}

// selectTokenHard - IA difficile: stratégie avancée
func (ai *AIPlayer) selectTokenHard(player *models.Player, diceValue int, board *models.Board) *models.Token {
	validTokens := ai.getValidTokens(player, diceValue, board)
	if len(validTokens) == 0 {
		return nil
	}

	type tokenScore struct {
		token *models.Token
		score int
	}

	scores := make([]tokenScore, 0, len(validTokens))

	for _, token := range validTokens {
		score := ai.evaluateMove(token, diceValue, player, board)
		scores = append(scores, tokenScore{token: token, score: score})
	}

	// Trouver le meilleur score
	best := scores[0]
	for _, ts := range scores[1:] {
		if ts.score > best.score {
			best = ts
		}
	}

	return best.token
}

// evaluateMove évalue la qualité d'un déplacement
func (ai *AIPlayer) evaluateMove(token *models.Token, diceValue int, player *models.Player, board *models.Board) int {
	score := 0
	newPos := ai.calculateNewPosition(token, diceValue, player.Color)

	// 1. Capture d'un adversaire (+1000 points)
	if ai.canCapture(newPos, player.Color, board) {
		score += 1000
	}

	// 2. Sortir de la base (+500 points)
	if token.Position == -1 && diceValue == constants.RollToStart {
		score += 500
	}

	// 3. Entrer dans la zone maison (+800 points)
	if newPos >= 52 {
		score += 800
	}

	// 4. Atteindre une zone sécurisée (+300 points)
	if ai.isSafePosition(newPos) {
		score += 300
	}

	// 5. Avancer le token le plus proche de la victoire (+100 points par case)
	score += newPos * 10

	// 6. Éviter de laisser un token isolé (-200 points)
	if ai.isTokenIsolated(token, player.Tokens, board) {
		score -= 200
	}

	// 7. Danger d'être capturé après le déplacement (-400 points)
	if ai.isPositionDangerous(newPos, player.Color, board) {
		score -= 400
	}

	// 8. Bloquer un adversaire proche de la victoire (+600 points)
	if ai.blocksOpponent(newPos, board) {
		score += 600
	}

	return score
}

// getValidTokens retourne les tokens qui peuvent se déplacer
func (ai *AIPlayer) getValidTokens(player *models.Player, diceValue int, board *models.Board) []*models.Token {
	valid := make([]*models.Token, 0, constants.TokensPerPlayer)

	for _, token := range player.Tokens {
		if ai.canMoveToken(token, diceValue, player.Color, board) {
			valid = append(valid, token)
		}
	}

	return valid
}

// canMoveToken vérifie si un token peut se déplacer
func (ai *AIPlayer) canMoveToken(token *models.Token, diceValue int, color constants.PlayerColor, board *models.Board) bool {
	// Token déjà à la maison
	if token.IsHome {
		return false
	}

	// Token en base: doit obtenir un 6
	if token.Position == -1 {
		return diceValue == constants.RollToStart
	}

	// Vérifier que la nouvelle position est valide
	newPos := ai.calculateNewPosition(token, diceValue, color)

	// Dépassement de la maison
	if newPos > 57 {
		return false
	}

	// Vérifier qu'il n'y a pas déjà un token de la même couleur
	if newPos >= 52 {
		// Zone maison
		homeIndex := newPos - 52
		if board.HomeStretches[color][homeIndex].Token != nil &&
			board.HomeStretches[color][homeIndex].Token.Color == color {
			return false
		}
	} else {
		// Plateau normal
		if board.Cells[newPos].Token != nil &&
			board.Cells[newPos].Token.Color == color {
			return false
		}
	}

	return true
}

// calculateNewPosition calcule la nouvelle position d'un token
func (ai *AIPlayer) calculateNewPosition(token *models.Token, diceValue int, color constants.PlayerColor) int {
	if token.Position == -1 {
		// Sortie de la base
		return constants.StartingPositions[color]
	}

	newPos := token.Position + diceValue

	// Vérifier si on entre dans la zone maison
	homeEntry := constants.HomeStretchStart[color]
	if token.Position < homeEntry && newPos >= homeEntry {
		// Entrer dans la maison
		overflow := newPos - homeEntry
		return 52 + overflow
	}

	// Gérer le tour du plateau
	if newPos >= 52 && token.Position < 52 {
		newPos = newPos % 52
	}

	return newPos
}

// canCapture vérifie si on peut capturer à cette position
func (ai *AIPlayer) canCapture(pos int, color constants.PlayerColor, board *models.Board) bool {
	if pos < 0 || pos >= 52 {
		return false
	}

	cell := board.Cells[pos]
	if cell.Token == nil {
		return false
	}

	// Pas de capture sur les zones sécurisées
	if cell.IsSafe {
		return false
	}

	return cell.Token.Color != color
}

// isSafePosition vérifie si la position est sécurisée
func (ai *AIPlayer) isSafePosition(pos int) bool {
	if pos < 0 || pos >= 52 {
		return true // Base et maison sont sécurisées
	}

	for _, safe := range constants.SafePositions {
		if pos == safe {
			return true
		}
	}
	return false
}

// isTokenIsolated vérifie si le token est isolé
func (ai *AIPlayer) isTokenIsolated(token *models.Token, allTokens []*models.Token, board *models.Board) bool {
	if token.Position < 0 || token.Position >= 52 {
		return false
	}

	// Vérifier s'il y a d'autres tokens amis dans un rayon de 6 cases
	for _, other := range allTokens {
		if other.ID == token.ID || other.Position < 0 {
			continue
		}

		distance := abs(token.Position - other.Position)
		if distance <= 6 {
			return false
		}
	}

	return true
}

// isPositionDangerous vérifie si la position est dangereuse
func (ai *AIPlayer) isPositionDangerous(pos int, color constants.PlayerColor, board *models.Board) bool {
	if ai.isSafePosition(pos) {
		return false
	}

	// Vérifier s'il y a des adversaires dans un rayon de 6 cases derrière
	for i := 1; i <= 6; i++ {
		checkPos := (pos - i + 52) % 52
		cell := board.Cells[checkPos]
		if cell.Token != nil && cell.Token.Color != color {
			return true
		}
	}

	return false
}

// blocksOpponent vérifie si on bloque un adversaire
func (ai *AIPlayer) blocksOpponent(pos int, board *models.Board) bool {
	if pos < 0 || pos >= 52 {
		return false
	}

	// Vérifier s'il y a un adversaire proche de la victoire
	for i := 1; i <= 6; i++ {
		checkPos := (pos + i) % 52
		cell := board.Cells[checkPos]
		if cell.Token != nil && cell.Token.Position > 45 {
			return true
		}
	}

	return false
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
