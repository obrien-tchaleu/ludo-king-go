-- migrations/001_initial_schema.sql
CREATE DATABASE IF NOT EXISTS ludo_king CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE ludo_king;

-- Table des utilisateurs
CREATE TABLE users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    avatar_url VARCHAR(255),
    level INT DEFAULT 1,
    experience INT DEFAULT 0,
    coins INT DEFAULT 1000,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    last_login TIMESTAMP NULL,
    INDEX idx_username (username),
    INDEX idx_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Table des statistiques
CREATE TABLE player_stats (
    user_id BIGINT UNSIGNED PRIMARY KEY,
    total_games INT DEFAULT 0,
    games_won INT DEFAULT 0,
    games_lost INT DEFAULT 0,
    tokens_captured INT DEFAULT 0,
    tokens_lost INT DEFAULT 0,
    sixes_rolled INT DEFAULT 0,
    total_dice_rolls INT DEFAULT 0,
    win_rate DECIMAL(5,2) DEFAULT 0.00,
    highest_streak INT DEFAULT 0,
    current_streak INT DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Table de l'historique des parties
CREATE TABLE game_history (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    room_id VARCHAR(50) NOT NULL,
    game_mode ENUM('online', 'local', 'ai') NOT NULL,
    num_players INT NOT NULL,
    winner_id BIGINT UNSIGNED,
    duration_seconds INT,
    started_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP NULL,
    game_data JSON,
    INDEX idx_room (room_id),
    INDEX idx_winner (winner_id),
    FOREIGN KEY (winner_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Table des participants aux parties
CREATE TABLE game_participants (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    game_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED,
    player_position INT NOT NULL,
    color ENUM('red', 'blue', 'green', 'yellow') NOT NULL,
    final_rank INT,
    tokens_at_home INT DEFAULT 0,
    tokens_captured INT DEFAULT 0,
    dice_rolls INT DEFAULT 0,
    is_winner BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (game_id) REFERENCES game_history(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Table des achievements
CREATE TABLE achievements (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    icon_url VARCHAR(255),
    requirement_type ENUM('wins', 'captures', 'streak', 'special') NOT NULL,
    requirement_value INT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Table des achievements débloqués
CREATE TABLE user_achievements (
    user_id BIGINT UNSIGNED,
    achievement_id INT UNSIGNED,
    unlocked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, achievement_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (achievement_id) REFERENCES achievements(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Insérer des achievements de base
INSERT INTO achievements (name, description, requirement_type, requirement_value) VALUES
('First Victory', 'Win your first game', 'wins', 1),
('Serial Winner', 'Win 10 games', 'wins', 10),
('Champion', 'Win 100 games', 'wins', 100),
('Hunter', 'Capture 50 opponent tokens', 'captures', 50),
('Predator', 'Capture 200 opponent tokens', 'captures', 200),
('Hot Streak', 'Win 5 games in a row', 'streak', 5),
('Unstoppable', 'Win 10 games in a row', 'streak', 10);