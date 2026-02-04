// pkg/database/database.go
package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/obrien-tchaleu/ludo-king-go/internal/shared/models"
)

type DB struct {
	conn *sql.DB
}

// NewDB crée une nouvelle connexion à la base de données
func NewDB(host, port, user, password, dbname string) (*DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4",
		user, password, host, port, dbname)

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configuration du pool de connexions
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	// Test de connexion
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{conn: conn}, nil
}

// Close ferme la connexion
func (db *DB) Close() error {
	return db.conn.Close()
}

// CreateUser crée un nouvel utilisateur
func (db *DB) CreateUser(username, email, passwordHash string) (*models.User, error) {
	query := `INSERT INTO users (username, email, password_hash, level, experience, coins) 
	          VALUES (?, ?, ?, 1, 0, 1000)`

	result, err := db.conn.Exec(query, username, email, passwordHash)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get user id: %w", err)
	}

	// Créer les statistiques du joueur
	statsQuery := `INSERT INTO player_stats (user_id) VALUES (?)`
	if _, err := db.conn.Exec(statsQuery, id); err != nil {
		return nil, fmt.Errorf("failed to create player stats: %w", err)
	}

	return db.GetUserByID(id)
}

// GetUserByID récupère un utilisateur par son ID
func (db *DB) GetUserByID(id int64) (*models.User, error) {
	query := `SELECT id, username, email, avatar_url, level, experience, coins, 
	          created_at, last_login FROM users WHERE id = ?`

	user := &models.User{}
	err := db.conn.QueryRow(query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.AvatarURL,
		&user.Level, &user.Experience, &user.Coins,
		&user.CreatedAt, &user.LastLogin,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByUsername récupère un utilisateur par son username
func (db *DB) GetUserByUsername(username string) (*models.User, error) {
	query := `SELECT id, username, email, password_hash, avatar_url, level, 
	          experience, coins, created_at, last_login FROM users WHERE username = ?`

	user := &models.User{}
	err := db.conn.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.AvatarURL,
		&user.Level, &user.Experience, &user.Coins,
		&user.CreatedAt, &user.LastLogin,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// UpdateLastLogin met à jour la dernière connexion
func (db *DB) UpdateLastLogin(userID int64) error {
	query := `UPDATE users SET last_login = NOW() WHERE id = ?`
	_, err := db.conn.Exec(query, userID)
	return err
}

// GetPlayerStats récupère les statistiques d'un joueur
func (db *DB) GetPlayerStats(userID int64) (*models.PlayerStats, error) {
	query := `SELECT user_id, total_games, games_won, games_lost, tokens_captured,
	          tokens_lost, sixes_rolled, total_dice_rolls, win_rate, 
	          highest_streak, current_streak FROM player_stats WHERE user_id = ?`

	stats := &models.PlayerStats{}
	err := db.conn.QueryRow(query, userID).Scan(
		&stats.UserID, &stats.TotalGames, &stats.GamesWon, &stats.GamesLost,
		&stats.TokensCaptured, &stats.TokensLost, &stats.SixesRolled,
		&stats.TotalDiceRolls, &stats.WinRate, &stats.HighestStreak,
		&stats.CurrentStreak,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return stats, nil
}

// UpdatePlayerStats met à jour les statistiques après une partie
func (db *DB) UpdatePlayerStats(userID int64, won bool, tokensCaptured, tokensLost int) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `UPDATE player_stats SET 
	          total_games = total_games + 1,
	          games_won = games_won + ?,
	          games_lost = games_lost + ?,
	          tokens_captured = tokens_captured + ?,
	          tokens_lost = tokens_lost + ?,
	          win_rate = (games_won + ?) * 100.0 / (total_games + 1),
	          current_streak = CASE WHEN ? = 1 THEN current_streak + 1 ELSE 0 END,
	          highest_streak = GREATEST(highest_streak, 
	                          CASE WHEN ? = 1 THEN current_streak + 1 ELSE 0 END)
	          WHERE user_id = ?`

	wonInt := 0
	lostInt := 0
	if won {
		wonInt = 1
	} else {
		lostInt = 1
	}

	_, err = tx.Exec(query, wonInt, lostInt, tokensCaptured, tokensLost,
		wonInt, wonInt, wonInt, userID)
	if err != nil {
		return err
	}

	// Mettre à jour l'expérience et les coins
	expGain := 100
	coinsGain := 50
	if won {
		expGain = 500
		coinsGain = 200
	}

	updateUser := `UPDATE users SET 
	               experience = experience + ?,
	               coins = coins + ?,
	               level = 1 + FLOOR((experience + ?) / 1000)
	               WHERE id = ?`

	_, err = tx.Exec(updateUser, expGain, coinsGain, expGain, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// SaveGameHistory enregistre une partie terminée
func (db *DB) SaveGameHistory(game *models.Game) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	duration := int(time.Since(game.StartTime).Seconds())
	var winnerID *int64
	if game.Winner != nil {
		winnerID = &game.Winner.ID
	}

	query := `INSERT INTO game_history 
	          (room_id, game_mode, num_players, winner_id, duration_seconds, 
	           started_at, ended_at) 
	          VALUES (?, ?, ?, ?, ?, ?, NOW())`

	result, err := tx.Exec(query, game.Room.ID, game.Room.GameMode,
		len(game.Room.Players), winnerID, duration, game.StartTime)
	if err != nil {
		return err
	}

	gameID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	// Enregistrer les participants
	for i, player := range game.Room.Players {
		if player.IsAI {
			continue // Ne pas enregistrer les joueurs IA
		}

		finalRank := i + 1
		isWinner := game.Winner != nil && player.ID == game.Winner.ID

		participantQuery := `INSERT INTO game_participants 
		                     (game_id, user_id, player_position, color, 
		                      final_rank, tokens_at_home, is_winner) 
		                     VALUES (?, ?, ?, ?, ?, ?, ?)`

		_, err = tx.Exec(participantQuery, gameID, player.ID, i,
			player.Color, finalRank, player.TokensAtHome, isWinner)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetLeaderboard récupère le classement
func (db *DB) GetLeaderboard(limit int) ([]*models.User, error) {
	query := `SELECT u.id, u.username, u.avatar_url, u.level, u.experience,
	          ps.total_games, ps.games_won, ps.win_rate
	          FROM users u
	          JOIN player_stats ps ON u.id = ps.user_id
	          ORDER BY ps.games_won DESC, ps.win_rate DESC
	          LIMIT ?`

	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		var totalGames, gamesWon int
		var winRate float64

		err := rows.Scan(&user.ID, &user.Username, &user.AvatarURL,
			&user.Level, &user.Experience, &totalGames, &gamesWon, &winRate)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}
