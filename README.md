# 🎲 Ludo King Go - Multiplayer Game

Un clone professionnel de Ludo King développé en Go avec architecture client-serveur, interface graphique moderne et système de jeu complet.

![Version](https://img.shields.io/badge/version-1.0.0-blue)
![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-MIT-green)

## 📋 Table des matières

- [Caractéristiques](#-caractéristiques)
- [Architecture](#-architecture)
- [Prérequis](#-prérequis)
- [Installation](#-installation)
- [Configuration](#-configuration)
- [Utilisation](#-utilisation)
- [Structure du projet](#-structure-du-projet)
- [Technologies utilisées](#-technologies-utilisées)
- [Règles du jeu](#-règles-du-jeu)
- [Développement](#-développement)

## ✨ Caractéristiques

### 🎮 Modes de jeu
- **Play Online** - Multijoueur en ligne via serveur TCP
- **Play with Friends** - Création de rooms privées avec codes
- **Play vs AI** - 3 niveaux de difficulté (Easy, Medium, Hard)
- **Local Multiplayer** - Jeu en réseau local (LAN)

### 🎯 Fonctionnalités principales
- ✅ 2 à 4 joueurs simultanés
- ✅ Plateau graphique interactif avec animations
- ✅ Système de dés sécurisé côté serveur
- ✅ Intelligence artificielle avec stratégies avancées
- ✅ Gestion des salles avec codes de room
- ✅ Statistiques et historique des parties (MySQL)
- ✅ Leaderboard avec classements
- ✅ Paramètres audio et graphiques
- ✅ Reconnexion automatique
- ✅ Anti-triche avec validation serveur

### 🎨 Interface utilisateur
- Interface graphique moderne avec Fyne v2
- Plateau Ludo coloré avec 4 zones (Rouge, Vert, Jaune, Bleu)
- Tokens animés avec ombres et reflets
- Cases de sécurité marquées par des étoiles
- Système de notifications en temps réel

## 🏗️ Architecture


┌─────────────┐         TCP/JSON        ┌─────────────┐
│   Client    │◄──────────────────────►│   Serveur   │
│   (Fyne)    │    Goroutines/Channels │   (TCP)     │
└─────────────┘                         └──────┬──────┘
                                               │
                                               ▼
                                        ┌─────────────┐
                                        │    MySQL    │
                                        │   Database  │
                                        └─────────────┘



## 📦 Prérequis

- **Go** 1.21 ou supérieur
- **MySQL** 8.0 ou supérieur
- **Git** (pour cloner le projet)
- **GCC** (pour compilation Fyne sur Windows)

### Installation des prérequis

**Windows:**
bash
# Installer Go depuis https://go.dev/dl/
# Installer MySQL depuis https://dev.mysql.com/downloads/installer/
# Installer Git depuis https://git-scm.com/download/win


**Linux:**
bash
sudo apt-get update
sudo apt-get install golang mysql-server git build-essential


**macOS:**
bash
brew install go mysql git


## 🚀 Installation

### 1. Cloner le projet

bash
mkdir -p ~/Projects/ludo-king-go
cd ~/Projects/ludo-king-go
git clone <votre-repo-url> .


### 2. Initialiser le module Go

bash
go mod init github.com/yourusername/ludo-king-go
go mod tidy


### 3. Installer les dépendances

bash
go get fyne.io/fyne/v2@latest
go get github.com/go-sql-driver/mysql@latest
go get gopkg.in/yaml.v3@latest


### 4. Configurer MySQL

bash
# Se connecter à MySQL
mysql -u root -p

# Créer la base de données
mysql> source migrations/001_initial_schema.sql

# Créer l'utilisateur
mysql> CREATE USER 'ludo_user'@'localhost' IDENTIFIED BY 'LudoPass2024!';
mysql> GRANT ALL PRIVILEGES ON ludo_king.* TO 'ludo_user'@'localhost';
mysql> FLUSH PRIVILEGES;
mysql> EXIT;


### 5. Configuration du serveur

Éditez `configs/server.yaml`:

yaml
server:
  host: "0.0.0.0"
  port: "8080"

database:
  host: "localhost"
  port: "3306"
  username: "ludo_user"
  password: "LudoPass2024!"  # Changez selon votre config
  database: "ludo_king"

game:
  max_players_per_room: 4
  turn_timeout: 30


### 6. Compiler

bash
# Compiler le serveur
go build -o bin/ludo-server cmd/server/main.go

# Compiler le client
go build -o bin/ludo-client cmd/client/main.go


## 🎮 Utilisation

### Démarrer le serveur

bash
# Windows
.\bin\ludo-server.exe

# Linux/macOS
./bin/ludo-server


Sortie attendue:

✅ Connected to database successfully
🎲 Ludo King Server started on port 8080


### Lancer le client

bash
# Windows
.\bin\ludo-client.exe

# Linux/macOS
./bin/ludo-client


### Modes de jeu

#### 🌐 Play Online
1. Cliquez sur "Play Online"
2. Entrez l'adresse du serveur (ex: `localhost:8080`)
3. Choisissez un nom d'utilisateur
4. Rejoignez ou créez une room

#### 👥 Play with Friends
1. **Créer une room:**
   - Cliquez sur "Play with Friends" → "Create Room"
   - Définissez le nom et nombre de joueurs
   - Un code unique est généré (ex: `ROOM_12345`)
   - Partagez ce code avec vos amis

2. **Rejoindre une room:**
   - Cliquez sur "Join Room"
   - Entrez le code de la room
   - Attendez que tous soient prêts

#### 🤖 Play vs AI
1. Cliquez sur "Play vs AI"
2. Sélectionnez la difficulté (Easy/Medium/Hard)
3. Choisissez le nombre d'adversaires (1-3)
4. Cliquez sur "Start Game"

## 📁 Structure du projet


ludo-king-go/
├── cmd/
│   ├── server/              # Point d'entrée serveur
│   │   └── main.go
│   └── client/              # Point d'entrée client
│       └── main.go
├── internal/
│   ├── server/              # Logique serveur
│   │   ├── game/           # Moteur de jeu
│   │   │   └── engine.go
│   │   ├── room/           # Gestion des salles
│   │   ├── matchmaking/    # Matchmaking
│   │   └── auth/           # Authentification
│   ├── client/              # Logique client
│   │   ├── ui/             # Interface graphique
│   │   ├── network/        # Communication réseau
│   │   └── audio/          # Système audio
│   └── shared/              # Code partagé
│       ├── protocol/       # Protocole réseau
│       ├── models/         # Modèles de données
│       │   └── models.go
│       └── constants/      # Constantes
│           └── constants.go
├── pkg/
│   ├── ai/                  # Intelligence artificielle
│   │   └── ai.go
│   └── database/            # Accès base de données
│       └── database.go
├── assets/                  # Ressources
│   ├── images/
│   ├── sounds/
│   └── fonts/
├── configs/                 # Configuration
│   └── server.yaml
├── migrations/              # Migrations SQL
│   └── 001_initial_schema.sql
├── scripts/                 # Scripts utilitaires
├── bin/                     # Binaires compilés
├── go.mod
├── go.sum
└── gitignore


## 🛠️ Technologies utilisées

### Backend
- **Go 1.21+** - Langage principal
- **MySQL** - Base de données relationnelle
- **TCP/JSON** - Communication réseau
- **Goroutines** - Concurrence
- **Channels** - Synchronisation

### Frontend
- **Fyne v2** - Framework GUI multiplateforme
- **Canvas** - Rendu graphique 2D

### Bibliothèques
go
require (
    fyne.io/fyne/v2 v2.4.0
    github.com/go-sql-driver/mysql v1.7.1
    gopkg.in/yaml.v3 v3.0.1
)


## 🎲 Règles du jeu

### Objectif
Être le premier à faire rentrer tous ses 4 pions dans la zone d'arrivée.

### Déroulement
1. **Démarrage:** Lancer un 6 pour sortir un pion de la base
2. **Déplacement:** Avancer selon le résultat du dé (1-6)
3. **Tour bonus:** Obtenir un 6 donne un tour supplémentaire
4. **Capture:** Atterrir sur un pion adverse le renvoie à sa base
5. **Zones sûres:** Cases étoiles protègent de la capture
6. **3 six consécutifs:** Le joueur perd son tour

### Zones du plateau
- 🔴 **Rouge** - Position de départ: Case 0
- 🟢 **Vert** - Position de départ: Case 13
- 🟡 **Jaune** - Position de départ: Case 26
- 🔵 **Bleu** - Position de départ: Case 39

## 👨‍💻 Développement

### Lancer les tests

bash
# Tester tout le projet
go test ./... -v

# Tester un package spécifique
go test ./pkg/database -v

### Mode développement

bash
# Lancer le serveur en mode watch (avec air)
air -c .air.toml

# Ou directement
go run cmd/server/main.go

# Client
go run cmd/client/main.go


### Ajouter des migrations

sql
-- migrations/002_add_feature.sql
ALTER TABLE users ADD COLUMN new_field VARCHAR(100);


## 🐛 Dépannage

### Erreur MySQL "Access denied"

bash
# Réinitialiser l'utilisateur
mysql -u root -p
mysql> DROP USER IF EXISTS 'ludo_user'@'localhost';
mysql> CREATE USER 'ludo_user'@'localhost' IDENTIFIED BY 'LudoPass2024!';
mysql> GRANT ALL PRIVILEGES ON ludo_king.* TO 'ludo_user'@'localhost';
mysql> FLUSH PRIVILEGES;


### Client ne compile pas (Windows)

bash
 Installer GCC via TDM-GCC ou MinGW  Ou utiliser WSL2

### Serveur ne démarre pas

bash
 Vérifier que le port 8080 est libre
netstat -ano | findstr :8080

 Changer le port dans configs/server.yaml si nécessaire


 👥 Contributeurs

- Tchaleu Foadjo Chatel O'brien Hunter
- Zanga Djerry Vivien
- Stephanie Bessem Ndoumbe
- Tomo Ombolo Dyrlane
