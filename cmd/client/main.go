// cmd/client/main.go - Version COMPLÈTE avec réseau et déplacement fonctionnels
package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"math"
	"net"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/obrien-tchaleu/ludo-king-go/internal/shared/constants"
	"github.com/obrien-tchaleu/ludo-king-go/internal/shared/models"
)

// ============================================================================
// THEME
// ============================================================================

type LudoTheme struct{}

func (m LudoTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNameBackground {
		return color.NRGBA{R: 0x30, G: 0x30, B: 0x30, A: 0xff}
	}
	return theme.DefaultTheme().Color(name, variant)
}
func (m LudoTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}
func (m LudoTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}
func (m LudoTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

// ============================================================================
// CONSTANTES
// ============================================================================

const BOARD_GRID = 15
const HOME_SIZE = 6
const PATH_LEN = 52
const HOME_STRETCH_LEN = 5

var boardPath = [PATH_LEN][2]int{
	{6, 13}, {6, 12}, {6, 11}, {6, 10}, {6, 9}, {6, 8},
	{5, 8}, {4, 8}, {3, 8}, {2, 8}, {1, 8}, {0, 8},
	{0, 7}, {0, 6},
	{1, 6}, {2, 6}, {3, 6}, {4, 6}, {5, 6}, {6, 6},
	{6, 5}, {6, 4}, {6, 3}, {6, 2}, {6, 1}, {6, 0},
	{7, 0}, {8, 0},
	{8, 1}, {8, 2}, {8, 3}, {8, 4}, {8, 5}, {8, 6},
	{9, 6}, {10, 6}, {11, 6}, {12, 6}, {13, 6}, {14, 6},
	{14, 7}, {14, 8},
	{13, 8}, {12, 8}, {11, 8}, {10, 8}, {9, 8}, {8, 8},
	{8, 9}, {8, 10}, {8, 11}, {8, 12},
}

var homePositions = map[constants.PlayerColor][4][2]int{
	constants.ColorRed:    {{1, 1}, {4, 1}, {1, 4}, {4, 4}},
	constants.ColorGreen:  {{10, 1}, {13, 1}, {10, 4}, {13, 4}},
	constants.ColorYellow: {{10, 10}, {13, 10}, {10, 13}, {13, 13}},
	constants.ColorBlue:   {{1, 10}, {4, 10}, {1, 13}, {4, 13}},
}

var startIndex = map[constants.PlayerColor]int{
	constants.ColorRed:    0,
	constants.ColorGreen:  13,
	constants.ColorYellow: 26,
	constants.ColorBlue:   39,
}

var safeCells = map[int]bool{
	1: true, 9: true, 14: true, 22: true, 27: true, 35: true, 40: true, 48: true,
}

// ============================================================================
// CLIENT STRUCTURE
// ============================================================================

type Client struct {
	app           fyne.App
	window        fyne.Window
	conn          net.Conn
	user          *models.User
	gameState     *models.Game
	mainMenu      *fyne.Container
	gameBoard     *fyne.Container
	boardImage    *canvas.Image
	diceButton    *widget.Button
	diceDisplay   *canvas.Text
	diceValue     *canvas.Text
	statusLabel   *widget.Label
	playersList   *widget.List
	send          chan *models.NetworkMessage
	receive       chan *models.NetworkMessage
	done          chan bool
	currentDice   int
	isMyTurn      bool
	boardSize     float32
	mu            sync.Mutex
	rollCount     int
	selectedToken *SelectedToken // Pion sélectionné
	connected     bool
	serverAddress string
}

// SelectedToken représente un pion sélectionné
type SelectedToken struct {
	PlayerIndex int
	TokenIndex  int
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	myApp := app.NewWithID("com.ludoking.game")
	myApp.Settings().SetTheme(&LudoTheme{})
	client := &Client{
		app:       myApp,
		window:    myApp.NewWindow("Ludo King - Go Edition"),
		send:      make(chan *models.NetworkMessage, 256),
		receive:   make(chan *models.NetworkMessage, 256),
		done:      make(chan bool),
		rollCount: 0,
		connected: false,
	}

	client.window.Resize(fyne.NewSize(1280, 800))
	client.window.CenterOnScreen()
	client.showMainMenu()
	client.window.ShowAndRun()
}

// ============================================================================
// MENU PRINCIPAL
// ============================================================================

func (c *Client) showMainMenu() {
	title := canvas.NewText("LUDO KING", color.White)
	title.TextSize = 48
	title.Alignment = fyne.TextAlignCenter

	subtitle := widget.NewLabel("Go Edition")
	subtitle.Alignment = fyne.TextAlignCenter

	playOnlineBtn := widget.NewButton("🌐 Play Online", func() {
		c.showServerConnect()
	})
	playOnlineBtn.Importance = widget.HighImportance

	playWithFriendsBtn := widget.NewButton("👥 Play with Friends", func() {
		c.showFriendsMenu()
	})

	playVsAIBtn := widget.NewButton("🤖 Play vs AI", func() {
		c.showAISetup()
	})

	settingsBtn := widget.NewButton("⚙️ Settings", func() {
		c.showSettings()
	})

	leaderboardBtn := widget.NewButton("🏆 Leaderboard", func() {
		c.showLeaderboard()
	})

	quitBtn := widget.NewButton("Exit", func() {
		c.window.Close()
	})

	buttonsContainer := container.NewVBox(
		playOnlineBtn,
		playWithFriendsBtn,
		playVsAIBtn,
		leaderboardBtn,
		settingsBtn,
		quitBtn,
	)

	titleContainer := container.NewVBox(
		container.NewCenter(title),
		container.NewCenter(subtitle),
		layout.NewSpacer(),
	)

	c.mainMenu = container.NewBorder(
		titleContainer,
		nil, nil, nil,
		container.NewCenter(buttonsContainer),
	)

	c.window.SetContent(c.mainMenu)
}

// ============================================================================
// CONNEXION RÉSEAU CORRIGÉE
// ============================================================================

func (c *Client) showServerConnect() {
	serverEntry := widget.NewEntry()
	serverEntry.SetPlaceHolder("Server address")
	serverEntry.SetText("localhost:8080")

	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("Username")
	usernameEntry.SetText(fmt.Sprintf("Player%d", time.Now().Unix()%1000))

	connectBtn := widget.NewButton("Connect", func() {
		server := serverEntry.Text
		username := usernameEntry.Text

		if username == "" {
			dialog.ShowError(fmt.Errorf("please enter username"), c.window)
			return
		}

		// Afficher dialogue de chargement
		progress := dialog.NewInformation("Connecting", "Connecting to server...", c.window)
		progress.Show()

		// Connexion dans une goroutine
		go func() {
			err := c.connectToServer(server, username)

			fyne.Do(func() {
				progress.Hide()

				if err != nil {
					dialog.ShowError(
						fmt.Errorf("Connection failed: %v\n\nMake sure the server is running:\ngo run cmd/server/main.go", err),
						c.window,
					)
				} else {
					dialog.ShowInformation(
						"Connected",
						fmt.Sprintf("✅ Connected as %s!", username),
						c.window,
					)
					c.showFriendsMenu()
				}
			})
		}()
	})
	connectBtn.Importance = widget.HighImportance

	backBtn := widget.NewButton("Back", func() {
		c.showMainMenu()
	})

	form := container.NewVBox(
		widget.NewLabelWithStyle("Connect to Server", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("Server Address:"),
		serverEntry,
		widget.NewLabel("Username:"),
		usernameEntry,
		widget.NewSeparator(),
		connectBtn,
		backBtn,
	)

	c.window.SetContent(container.NewCenter(form))
}

func (c *Client) connectToServer(address, username string) error {
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.conn = conn
	c.serverAddress = address
	c.user = &models.User{
		ID:       time.Now().Unix(),
		Username: username,
	}

	// Démarrer les goroutines de communication
	go c.readMessages()
	go c.writeMessages()
	go c.processMessages()

	c.connected = true
	log.Printf("✅ Connected to server %s as %s", address, username)

	return nil
}

func (c *Client) readMessages() {
	decoder := json.NewDecoder(c.conn)
	for {
		var msg models.NetworkMessage
		if err := decoder.Decode(&msg); err != nil {
			if c.connected {
				log.Printf("❌ Connection lost: %v", err)
				c.connected = false

				fyne.Do(func() {
					dialog.ShowError(
						fmt.Errorf("Connection to server lost"),
						c.window,
					)
					c.showMainMenu()
				})
			}
			c.done <- true
			return
		}

		log.Printf("📨 Received: %s", msg.Type)
		c.receive <- &msg
	}
}

func (c *Client) writeMessages() {
	encoder := json.NewEncoder(c.conn)
	for msg := range c.send {
		if err := encoder.Encode(msg); err != nil {
			log.Printf("❌ Failed to send: %v", err)
			return
		}
		log.Printf("📤 Sent: %s", msg.Type)
	}
}

func (c *Client) processMessages() {
	for {
		select {
		case msg := <-c.receive:
			c.handleServerMessage(msg)
		case <-c.done:
			return
		}
	}
}

func (c *Client) handleServerMessage(msg *models.NetworkMessage) {
	switch msg.Type {
	case constants.MsgRoomCreated:
		c.handleRoomCreated(msg)
	case constants.MsgRoomJoined:
		c.handleRoomJoined(msg)
	case constants.MsgPlayerJoined:
		c.handlePlayerJoined(msg)
	case constants.MsgGameStart:
		c.handleGameStart(msg)
	case constants.MsgDiceRolled:
		c.handleDiceRolled(msg)
	case constants.MsgTokenMoved:
		c.handleTokenMoved(msg)
	case constants.MsgTurnChanged:
		c.handleTurnChanged(msg)
	case constants.MsgError:
		c.handleError(msg)
	}
}

func (c *Client) handleRoomCreated(msg *models.NetworkMessage) {
	payload := msg.Payload.(map[string]interface{})
	roomID := payload["room_id"].(string)

	log.Printf("✅ Room created: %s", roomID)

	fyne.Do(func() {
		dialog.ShowInformation(
			"Room Created",
			fmt.Sprintf("🔑 Room Code: %s\n\nShare this code with your friends!", roomID),
			c.window,
		)
		// TODO: Afficher le lobby en attente
	})
}

func (c *Client) handleRoomJoined(msg *models.NetworkMessage) {
	log.Printf("✅ Joined room successfully")

	fyne.Do(func() {
		dialog.ShowInformation(
			"Joined",
			"✅ You joined the room!",
			c.window,
		)
	})
}

func (c *Client) handlePlayerJoined(msg *models.NetworkMessage) {
	log.Printf("👤 Player joined")
	// Rafraîchir la liste des joueurs
}

func (c *Client) handleGameStart(msg *models.NetworkMessage) {
	log.Printf("🎮 Game starting!")

	fyne.Do(func() {
		c.showGameBoard()
	})
}

func (c *Client) handleDiceRolled(msg *models.NetworkMessage) {
	payload := msg.Payload.(map[string]interface{})
	diceValue := int(payload["dice_value"].(float64))

	c.mu.Lock()
	c.currentDice = diceValue
	c.mu.Unlock()

	fyne.Do(func() {
		c.diceValue.Text = fmt.Sprintf("%d", diceValue)
		c.diceValue.Refresh()
		c.refreshBoard()
	})
}

func (c *Client) handleTokenMoved(msg *models.NetworkMessage) {
	log.Printf("🎯 Token moved")

	fyne.Do(func() {
		c.refreshBoard()
	})
}

func (c *Client) handleTurnChanged(msg *models.NetworkMessage) {
	payload := msg.Payload.(map[string]interface{})
	playerID := int64(payload["player_id"].(float64))

	c.mu.Lock()
	c.isMyTurn = (playerID == c.user.ID)
	c.currentDice = 0
	c.selectedToken = nil
	c.mu.Unlock()

	fyne.Do(func() {
		if c.isMyTurn {
			c.statusLabel.SetText("🎲 Your turn! Roll the dice.")
			c.diceButton.Enable()
		} else {
			c.statusLabel.SetText("⏳ Opponent's turn...")
			c.diceButton.Disable()
		}
		c.refreshBoard()
	})
}

func (c *Client) handleError(msg *models.NetworkMessage) {
	payload := msg.Payload.(models.ErrorPayload)

	log.Printf("❌ Server error: %s", payload.Message)

	fyne.Do(func() {
		dialog.ShowError(
			fmt.Errorf("Server: %s", payload.Message),
			c.window,
		)
	})
}

// ============================================================================
// JOINTURE DE ROOM
// ============================================================================

func (c *Client) showFriendsMenu() {
	if !c.connected {
		dialog.ShowError(fmt.Errorf("Not connected to server"), c.window)
		c.showMainMenu()
		return
	}

	title := widget.NewLabelWithStyle("Play with Friends", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	createRoomBtn := widget.NewButton("Create Room", func() {
		c.showRoomCreation()
	})
	createRoomBtn.Importance = widget.HighImportance

	joinRoomBtn := widget.NewButton("Join Room", func() {
		c.showJoinRoomDialog()
	})

	backBtn := widget.NewButton("Back", func() {
		c.showMainMenu()
	})

	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		widget.NewLabel("Choose an option:"),
		createRoomBtn,
		joinRoomBtn,
		widget.NewSeparator(),
		backBtn,
	)

	c.window.SetContent(container.NewCenter(content))
}

func (c *Client) showJoinRoomDialog() {
	roomCodeEntry := widget.NewEntry()
	roomCodeEntry.SetPlaceHolder("Enter Room Code (ex: ROOM_83985)")

	joinBtn := widget.NewButton("Join", func() {
		roomCode := roomCodeEntry.Text
		if roomCode == "" {
			dialog.ShowError(fmt.Errorf("Please enter a room code"), c.window)
			return
		}

		// Envoyer le message de jointure au serveur
		c.send <- &models.NetworkMessage{
			Type: constants.MsgJoinRoom,
			Payload: map[string]interface{}{
				"room_id":  roomCode,
				"user_id":  c.user.ID,
				"username": c.user.Username,
			},
			Timestamp: time.Now(),
		}

		dialog.ShowInformation(
			"Joining",
			fmt.Sprintf("Joining room %s...", roomCode),
			c.window,
		)
	})
	joinBtn.Importance = widget.HighImportance

	backBtn := widget.NewButton("Back", func() {
		c.showFriendsMenu()
	})

	form := container.NewVBox(
		widget.NewLabelWithStyle("Join Game Room", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("📝 Enter the room code:"),
		roomCodeEntry,
		widget.NewSeparator(),
		joinBtn,
		backBtn,
	)

	c.window.SetContent(container.NewCenter(form))
}

func (c *Client) showRoomCreation() {
	roomNameEntry := widget.NewEntry()
	roomNameEntry.SetPlaceHolder("Room Name")
	roomNameEntry.SetText("Game Room")

	maxPlayersSelect := widget.NewSelect([]string{"2", "3", "4"}, func(value string) {})
	maxPlayersSelect.SetSelected("4")

	createBtn := widget.NewButton("Create Room", func() {
		roomName := roomNameEntry.Text
		if roomName == "" {
			roomName = "Game Room"
		}

		maxPlayers := 4
		switch maxPlayersSelect.Selected {
		case "2":
			maxPlayers = 2
		case "3":
			maxPlayers = 3
		}

		// Envoyer au serveur
		c.send <- &models.NetworkMessage{
			Type: constants.MsgCreateRoom,
			Payload: map[string]interface{}{
				"name":        roomName,
				"max_players": maxPlayers,
				"game_mode":   "online",
				"is_private":  false,
				"user_id":     c.user.ID,
				"username":    c.user.Username,
			},
			Timestamp: time.Now(),
		}
	})
	createBtn.Importance = widget.HighImportance

	backBtn := widget.NewButton("Back", func() {
		c.showFriendsMenu()
	})

	form := container.NewVBox(
		widget.NewLabelWithStyle("Create New Room", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("Room Name:"),
		roomNameEntry,
		widget.NewLabel("Max Players:"),
		maxPlayersSelect,
		widget.NewSeparator(),
		createBtn,
		backBtn,
	)

	c.window.SetContent(container.NewCenter(form))
}

// ============================================================================
// MODE IA (LOCAL)
// ============================================================================

func (c *Client) showAISetup() {
	if c.user == nil {
		c.user = &models.User{
			ID:       time.Now().Unix(),
			Username: fmt.Sprintf("Player%d", time.Now().Unix()%1000),
		}
	}

	aiLevelSelect := widget.NewSelect([]string{"Easy", "Medium", "Hard"}, func(value string) {})
	aiLevelSelect.SetSelected("Medium")

	numOpponentsSelect := widget.NewSelect([]string{"1", "2", "3"}, func(value string) {})
	numOpponentsSelect.SetSelected("1")

	startBtn := widget.NewButton("Start Game", func() {
		numOpponents := 1
		switch numOpponentsSelect.Selected {
		case "2":
			numOpponents = 2
		case "3":
			numOpponents = 3
		}
		c.createAIGame(aiLevelSelect.Selected, numOpponents)
	})
	startBtn.Importance = widget.HighImportance

	backBtn := widget.NewButton("Back", func() {
		c.showMainMenu()
	})

	form := container.NewVBox(
		widget.NewLabelWithStyle("Play vs AI", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("AI Difficulty:"),
		aiLevelSelect,
		widget.NewLabel("Number of Opponents:"),
		numOpponentsSelect,
		widget.NewSeparator(),
		startBtn,
		backBtn,
	)

	c.window.SetContent(container.NewCenter(form))
}

func (c *Client) createAIGame(aiLevel string, numOpponents int) {
	room := &models.Room{
		ID:          fmt.Sprintf("AI_%d", time.Now().Unix()),
		Name:        "AI Game",
		HostID:      c.user.ID,
		Players:     make([]*models.Player, 0),
		MaxPlayers:  numOpponents + 1,
		GameMode:    "ai",
		State:       constants.StateWaiting,
		CreatedAt:   time.Now(),
		CurrentTurn: 0,
	}

	player := models.NewPlayer(c.user.ID, c.user.Username, constants.ColorRed)
	room.Players = append(room.Players, player)

	colors := []constants.PlayerColor{constants.ColorBlue, constants.ColorGreen, constants.ColorYellow}
	for i := 0; i < numOpponents; i++ {
		aiPlayer := models.NewAIPlayer(colors[i], aiLevel)
		aiPlayer.Username = fmt.Sprintf("AI Bot %d", i+1)
		room.Players = append(room.Players, aiPlayer)
	}

	c.gameState = &models.Game{
		Room:      room,
		Board:     models.NewBoard(),
		StartTime: time.Now(),
	}

	c.showGameBoard()
}

// ============================================================================
// PLATEAU DE JEU
// ============================================================================

func (c *Client) showGameBoard() {
	if c.gameState == nil || c.gameState.Room == nil {
		dialog.ShowError(fmt.Errorf("no game state"), c.window)
		return
	}

	log.Printf("🎮 Starting game board...")

	c.currentDice = 0
	c.isMyTurn = c.gameState.Room.CurrentTurn == 0
	c.boardSize = 600
	c.selectedToken = nil

	boardPixelSize := int(c.boardSize)
	rendered := c.renderBoard(boardPixelSize, boardPixelSize)
	c.boardImage = canvas.NewImageFromImage(rendered)
	c.boardImage.Resize(fyne.NewSize(c.boardSize, c.boardSize))
	c.boardImage.SetMinSize(fyne.NewSize(c.boardSize, c.boardSize))

	boardContainer := container.NewWithoutLayout(c.boardImage)
	boardContainer.Resize(fyne.NewSize(c.boardSize, c.boardSize))

	boardTapHandler := NewTappableRect(c.boardSize, func(pos fyne.Position) {
		c.onBoardTapped(pos)
	})
	boardContainer.Add(boardTapHandler)

	c.diceDisplay = canvas.NewText("🎲", color.White)
	c.diceDisplay.TextSize = 64
	c.diceDisplay.Alignment = fyne.TextAlignCenter

	c.diceValue = canvas.NewText("", color.White)
	c.diceValue.Alignment = fyne.TextAlignCenter
	c.diceValue.TextSize = 32
	c.diceValue.TextStyle = fyne.TextStyle{Bold: true}

	diceBackground := canvas.NewRectangle(color.NRGBA{R: 50, G: 50, B: 50, A: 255})
	diceBox := container.NewStack(
		diceBackground,
		container.NewPadded(
			container.NewVBox(
				widget.NewLabelWithStyle("🎲 Dice", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				container.NewCenter(c.diceDisplay),
				container.NewCenter(c.diceValue),
			),
		),
	)

	c.statusLabel = widget.NewLabel("🎲 Your turn! Roll the dice.")
	if !c.isMyTurn {
		c.statusLabel.SetText("⏳ Waiting for opponent...")
	}

	c.diceButton = widget.NewButton("🎲 Roll Dice", func() {
		c.onDiceRoll()
	})
	c.diceButton.Importance = widget.HighImportance
	if !c.isMyTurn {
		c.diceButton.Disable()
	}

	c.playersList = c.createPlayersList()

	rightPanel := container.NewVBox(
		diceBox,
		container.NewPadded(c.diceButton),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("👥 Players", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		c.playersList,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("💡 Rules", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("• Roll 6 to move out\n• Click pawn to select (yellow)\n• Click again to move\n• Exact number to finish"),
	)

	rightPanelScroll := container.NewVScroll(container.NewPadded(rightPanel))
	rightPanelScroll.SetMinSize(fyne.NewSize(300, 0))

	c.statusLabel.TextStyle = fyne.TextStyle{Bold: true}
	c.statusLabel.Alignment = fyne.TextAlignCenter

	leaveButton := widget.NewButton("← Leave Game", func() {
		c.showMainMenu()
	})

	bottomPanel := container.NewVBox(
		widget.NewSeparator(),
		container.NewPadded(
			container.NewHBox(
				layout.NewSpacer(),
				c.statusLabel,
				layout.NewSpacer(),
			),
		),
		container.NewCenter(leaveButton),
	)

	mainLayout := container.NewBorder(
		nil,
		bottomPanel,
		nil,
		rightPanelScroll,
		container.NewCenter(boardContainer),
	)

	c.gameBoard = mainLayout
	c.window.SetContent(c.gameBoard)

	if !c.isMyTurn {
		go c.playAITurns()
	}
}

// ============================================================================
// RENDU DU PLATEAU (suite dans le prochain message car limite de caractères)
// ============================================================================
// PARTIE 2 - À ajouter après la partie 1

// ============================================================================
// RENDU DU PLATEAU
// ============================================================================

func (c *Client) renderBoard(width, height int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.NRGBA{255, 255, 255, 255}}, image.Point{}, draw.Src)

	cs := float64(width) / float64(BOARD_GRID)

	// Zones home colorées
	drawHomeZone(img, 0, 0, cs, redColor())
	drawHomeZone(img, 9, 0, cs, greenColor())
	drawHomeZone(img, 9, 9, cs, yellowColor())
	drawHomeZone(img, 0, 9, cs, blueColor())

	// Chemin principal
	for _, pos := range boardPath {
		drawWhiteCell(img, pos[0], pos[1], cs)
	}

	// Home stretches
	redStretch := [][2]int{{7, 13}, {7, 12}, {7, 11}, {7, 10}, {7, 9}}
	for _, pos := range redStretch {
		drawColoredCell(img, pos[0], pos[1], cs, redColor())
	}

	greenStretch := [][2]int{{1, 7}, {2, 7}, {3, 7}, {4, 7}, {5, 7}}
	for _, pos := range greenStretch {
		drawColoredCell(img, pos[0], pos[1], cs, greenColor())
	}

	yellowStretch := [][2]int{{7, 1}, {7, 2}, {7, 3}, {7, 4}, {7, 5}}
	for _, pos := range yellowStretch {
		drawColoredCell(img, pos[0], pos[1], cs, yellowColor())
	}

	blueStretch := [][2]int{{13, 7}, {12, 7}, {11, 7}, {10, 7}, {9, 7}}
	for _, pos := range blueStretch {
		drawColoredCell(img, pos[0], pos[1], cs, blueColor())
	}

	// Centre
	drawCenterTriangle(img, 7, 7, cs)

	// Cases de départ
	drawStarCell(img, boardPath[0][0], boardPath[0][1], cs, redColor())
	drawStarCell(img, boardPath[13][0], boardPath[13][1], cs, greenColor())
	drawStarCell(img, boardPath[26][0], boardPath[26][1], cs, yellowColor())
	drawStarCell(img, boardPath[39][0], boardPath[39][1], cs, blueColor())

	// Flèches
	drawArrow(img, 6, 13, cs, "right", redColor())
	drawArrow(img, 0, 7, cs, "down", greenColor())
	drawArrow(img, 8, 1, cs, "left", yellowColor())
	drawArrow(img, 14, 7, cs, "up", blueColor())

	// 🎯 DESSINER LES TOKENS
	if c.gameState != nil && c.gameState.Room != nil {
		for pi, player := range c.gameState.Room.Players {
			pColor := getColorForPlayerColor(player.Color).(color.NRGBA)

			for ti, token := range player.Tokens {
				px, py := c.getTokenPixelPosition(player, ti, token, cs)

				// Ombre
				drawCircle(img, px+2, py+2, cs*0.3, color.NRGBA{0, 0, 0, 60})

				// 🎯 Déterminer la couleur
				tokenColor := pColor
				isSelected := c.selectedToken != nil &&
					c.selectedToken.PlayerIndex == pi &&
					c.selectedToken.TokenIndex == ti

				if isSelected {
					// Token sélectionné = JAUNE VIF
					tokenColor = color.NRGBA{255, 255, 0, 255}
				}

				// Token
				drawCircle(img, px, py, cs*0.3, tokenColor)

				// Bordure noire
				drawCircleOutline(img, px, py, cs*0.3, color.NRGBA{0, 0, 0, 200}, 2)

				// Highlight blanc
				drawCircle(img, px-cs*0.08, py-cs*0.08, cs*0.1, color.NRGBA{255, 255, 255, 120})

				// 🎯 Bordure verte si déplaçable
				if c.canMoveToken(player, ti) && !isSelected {
					drawCircleOutline(img, px, py, cs*0.35, color.NRGBA{0, 255, 0, 255}, 3)
				}
			}
		}
	}

	// Grille
	drawCompleteGrid(img, width, height, cs)
	return img
}

func (c *Client) getTokenPixelPosition(player *models.Player, tokenIndex int, token *models.Token, cs float64) (float64, float64) {
	if token.Position == -1 {
		hp := homePositions[player.Color]
		return (float64(hp[tokenIndex][0]) + 0.5) * cs, (float64(hp[tokenIndex][1]) + 0.5) * cs
	} else if token.Position < PATH_LEN {
		pathPos := boardPath[token.Position]
		return (float64(pathPos[0]) + 0.5) * cs, (float64(pathPos[1]) + 0.5) * cs
	} else {
		offset := token.Position - PATH_LEN
		return getHomeStretchPixelPos(player.Color, offset, cs)
	}
}

func getHomeStretchPixelPos(playerColor constants.PlayerColor, offset int, cs float64) (float64, float64) {
	switch playerColor {
	case constants.ColorRed:
		return (7.0 + 0.5) * cs, (float64(13-offset) + 0.5) * cs
	case constants.ColorGreen:
		return (float64(1+offset) + 0.5) * cs, (7.0 + 0.5) * cs
	case constants.ColorYellow:
		return (7.0 + 0.5) * cs, (float64(1+offset) + 0.5) * cs
	case constants.ColorBlue:
		return (float64(13-offset) + 0.5) * cs, (7.0 + 0.5) * cs
	}
	return 0, 0
}

func (c *Client) refreshBoard() {
	size := int(c.boardSize)
	if size < 450 {
		size = 450
	}
	rendered := c.renderBoard(size, size)
	fyne.Do(func() {
		c.boardImage.Image = rendered
		c.boardImage.Refresh()
	})
}

// ============================================================================
// 🎯 SYSTÈME DE SÉLECTION ET DÉPLACEMENT
// ============================================================================

func (c *Client) canMoveToken(player *models.Player, tokenIndex int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isMyTurn || c.currentDice == 0 {
		return false
	}
	if player.ID != c.user.ID {
		return false
	}

	token := player.Tokens[tokenIndex]

	// En base: besoin d'un 6
	if token.Position == -1 {
		return c.currentDice == 6
	}

	// Sur le plateau: vérifier dépassement
	relativePos := (token.Position - startIndex[player.Color] + PATH_LEN) % PATH_LEN
	newRelative := relativePos + c.currentDice

	// Ne peut pas dépasser la maison
	return newRelative <= PATH_LEN+HOME_STRETCH_LEN
}

func (c *Client) onBoardTapped(pos fyne.Position) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isMyTurn {
		log.Println("❌ Pas votre tour!")
		fyne.Do(func() {
			c.statusLabel.SetText("⏳ Wait for your turn!")
		})
		return
	}

	if c.currentDice == 0 {
		log.Println("⚠️ Lancez d'abord le dé!")
		fyne.Do(func() {
			c.statusLabel.SetText("🎲 Roll the dice first!")
		})
		return
	}

	if c.gameState == nil || c.gameState.Room == nil {
		return
	}

	cs := float64(c.boardSize) / float64(BOARD_GRID)
	clickCol := int(float64(pos.X) / cs)
	clickRow := int(float64(pos.Y) / cs)

	// Chercher le joueur actuel
	var myPlayer *models.Player
	var myPlayerIndex int
	for pi, player := range c.gameState.Room.Players {
		if player.ID == c.user.ID {
			myPlayer = player
			myPlayerIndex = pi
			break
		}
	}

	if myPlayer == nil {
		return
	}

	// 🎯 ÉTAPE 1: Chercher si on clique sur un token
	for ti, token := range myPlayer.Tokens {
		px, py := c.getTokenPixelPosition(myPlayer, ti, token, cs)
		tokenCol := int(px / cs)
		tokenRow := int(py / cs)

		if clickCol == tokenCol && clickRow == tokenRow {
			// Clic sur un token!

			if !c.canMoveToken(myPlayer, ti) {
				log.Printf("⚠️ Token %d ne peut pas bouger", ti)
				fyne.Do(func() {
					c.statusLabel.SetText(fmt.Sprintf("❌ This pawn cannot move with a %d", c.currentDice))
				})
				return
			}

			// 🎯 SÉLECTIONNER le token
			if c.selectedToken != nil && c.selectedToken.TokenIndex == ti {
				// Déjà sélectionné → DÉPLACER
				c.moveSelectedToken(myPlayer, myPlayerIndex, ti)
			} else {
				// Sélectionner
				c.selectedToken = &SelectedToken{
					PlayerIndex: myPlayerIndex,
					TokenIndex:  ti,
				}

				log.Printf("✅ Token %d sélectionné (devient jaune)", ti)
				fyne.Do(func() {
					c.statusLabel.SetText(fmt.Sprintf("🎯 Pawn selected! Click again to move %d spaces", c.currentDice))
				})
			}

			c.refreshBoard()
			return
		}
	}

	// 🎯 ÉTAPE 2: Si un token est sélectionné et qu'on clique ailleurs, on le déplace
	if c.selectedToken != nil {
		c.moveSelectedToken(myPlayer, myPlayerIndex, c.selectedToken.TokenIndex)
		c.refreshBoard()
	}
}

func (c *Client) moveSelectedToken(player *models.Player, playerIndex int, tokenIndex int) {
	token := player.Tokens[tokenIndex]
	oldPos := token.Position

	log.Printf("🚀 Déplacement du token %d depuis position %d", tokenIndex, oldPos)

	// Calculer nouvelle position
	if token.Position == -1 {
		// Sortir de la base avec un 6
		if c.currentDice == 6 {
			token.Position = startIndex[player.Color]
			log.Printf("🏠→🚀 Token sort en position %d", token.Position)
		} else {
			return
		}
	} else {
		// Déplacement normal
		relativePos := (token.Position - startIndex[player.Color] + PATH_LEN) % PATH_LEN
		newRelative := relativePos + c.currentDice

		if newRelative > PATH_LEN+HOME_STRETCH_LEN {
			log.Println("❌ Dépassement interdit!")
			return
		}

		if newRelative == PATH_LEN+HOME_STRETCH_LEN {
			token.Position = PATH_LEN + HOME_STRETCH_LEN
			log.Println("🏁 Token arrivé à la maison!")
		} else if newRelative >= PATH_LEN {
			token.Position = PATH_LEN + (newRelative - PATH_LEN)
		} else {
			newPos := (startIndex[player.Color] + newRelative) % PATH_LEN
			token.Position = newPos
		}
	}

	log.Printf("📍 Nouvelle position: %d", token.Position)

	// Vérifier capture
	c.checkCapture(player.Color, token.Position)

	// Vérifier victoire
	if c.checkWin(player) {
		fyne.Do(func() {
			c.statusLabel.SetText("🏆 YOU WIN!")
			dialog.ShowInformation("Victory!", "🏆 Congratulations! You won the game!", c.window)
		})
	}

	// Réinitialiser
	c.selectedToken = nil

	// Gérer le tour suivant
	if c.currentDice == 6 {
		log.Println("🎲 Vous avez fait un 6! Relancez!")
		c.currentDice = 0
		fyne.Do(func() {
			c.statusLabel.SetText("🎲 You got a 6! Roll again!")
			c.diceButton.Enable()
		})
	} else {
		c.currentDice = 0
		c.nextTurn()
	}
}

func (c *Client) checkCapture(myColor constants.PlayerColor, position int) {
	if position < 0 || position >= PATH_LEN {
		return
	}
	if safeCells[position] {
		return
	}

	for _, player := range c.gameState.Room.Players {
		if player.Color == myColor {
			continue
		}
		for _, token := range player.Tokens {
			if token.Position == position {
				token.Position = -1
				log.Printf("💥 CAPTURE! Token de %s renvoyé", player.Username)
				fyne.Do(func() {
					c.statusLabel.SetText(fmt.Sprintf("💥 Captured %s's pawn!", player.Username))
				})
			}
		}
	}
}

func (c *Client) checkWin(player *models.Player) bool {
	for _, token := range player.Tokens {
		if token.Position != PATH_LEN+HOME_STRETCH_LEN {
			return false
		}
	}
	return true
}

// ============================================================================
// DÉ TRUQUÉ
// ============================================================================

func (c *Client) rollDiceWithCheat() int {
	c.rollCount++
	if c.rollCount == 1 || c.rollCount%5 == 0 {
		log.Printf("🎲 DÉ TRUQUÉ! Lancer #%d → 6", c.rollCount)
		return 6
	}
	dice := int(time.Now().UnixNano()%6) + 1
	log.Printf("🎲 Lancer #%d → %d", c.rollCount, dice)
	return dice
}

func (c *Client) onDiceRoll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isMyTurn {
		fyne.Do(func() {
			c.statusLabel.SetText("⏳ Wait for your turn!")
		})
		return
	}

	c.currentDice = c.rollDiceWithCheat()

	fyne.Do(func() {
		c.diceValue.Text = fmt.Sprintf("%d", c.currentDice)
		c.diceValue.Refresh()
		c.diceButton.Disable()
	})

	log.Printf("🎲 Dé lancé: %d", c.currentDice)

	// Vérifier mouvements possibles
	hasMove := false
	for _, player := range c.gameState.Room.Players {
		if player.ID == c.user.ID {
			for ti := range player.Tokens {
				if c.canMoveToken(player, ti) {
					hasMove = true
					break
				}
			}
			break
		}
	}

	if !hasMove {
		log.Println("❌ Aucun mouvement possible")
		fyne.Do(func() {
			c.statusLabel.SetText(fmt.Sprintf("🎯 Rolled %d - No valid moves!", c.currentDice))
		})

		go func() {
			time.Sleep(2 * time.Second)
			c.mu.Lock()
			c.currentDice = 0
			c.nextTurn()
			c.mu.Unlock()
		}()
	} else {
		fyne.Do(func() {
			c.statusLabel.SetText(fmt.Sprintf("🎯 Rolled %d! Click a pawn to select (yellow)", c.currentDice))
		})
	}

	c.refreshBoard()
}

// ============================================================================
// TOUR SUIVANT
// ============================================================================

func (c *Client) nextTurn() {
	if c.gameState == nil || c.gameState.Room == nil {
		return
	}

	c.gameState.Room.CurrentTurn = (c.gameState.Room.CurrentTurn + 1) % len(c.gameState.Room.Players)
	currentPlayer := c.gameState.Room.Players[c.gameState.Room.CurrentTurn]

	c.isMyTurn = currentPlayer.ID == c.user.ID
	c.currentDice = 0
	c.selectedToken = nil

	fyne.Do(func() {
		if c.playersList != nil {
			c.playersList.Refresh()
		}

		if c.isMyTurn {
			c.statusLabel.SetText("🎲 Your turn! Roll the dice.")
			c.diceButton.Enable()
		} else {
			c.statusLabel.SetText(fmt.Sprintf("⏳ %s's turn...", currentPlayer.Username))
			c.diceButton.Disable()
		}
	})

	c.refreshBoard()

	if !c.isMyTurn {
		go c.playAITurns()
	}
}

// ============================================================================
// IA
// ============================================================================

func (c *Client) playAITurns() {
	if c.gameState == nil || c.gameState.Room == nil {
		return
	}

	currentPlayer := c.gameState.Room.Players[c.gameState.Room.CurrentTurn]
	if !currentPlayer.IsAI {
		return
	}

	time.Sleep(1 * time.Second)

	c.mu.Lock()
	aiDice := c.rollDiceWithCheat()
	c.currentDice = aiDice
	c.mu.Unlock()

	fyne.Do(func() {
		c.diceValue.Text = fmt.Sprintf("%d", aiDice)
		c.diceValue.Refresh()
		c.statusLabel.SetText(fmt.Sprintf("🤖 %s rolled %d", currentPlayer.Username, aiDice))
	})

	time.Sleep(1 * time.Second)

	c.mu.Lock()
	moved := false
	player := c.gameState.Room.Players[c.gameState.Room.CurrentTurn]

	for ti := range player.Tokens {
		token := player.Tokens[ti]

		if token.Position == -1 && aiDice == 6 {
			token.Position = startIndex[player.Color]
			c.checkCapture(player.Color, token.Position)
			moved = true
			break
		} else if token.Position >= 0 && token.Position < PATH_LEN+HOME_STRETCH_LEN {
			relativePos := (token.Position - startIndex[player.Color] + PATH_LEN) % PATH_LEN
			newRelative := relativePos + aiDice

			if newRelative <= PATH_LEN+HOME_STRETCH_LEN {
				if newRelative == PATH_LEN+HOME_STRETCH_LEN {
					token.Position = PATH_LEN + HOME_STRETCH_LEN
				} else if newRelative >= PATH_LEN {
					token.Position = PATH_LEN + (newRelative - PATH_LEN)
				} else {
					token.Position = (startIndex[player.Color] + newRelative) % PATH_LEN
				}
				c.checkCapture(player.Color, token.Position)
				moved = true
				break
			}
		}
	}
	c.mu.Unlock()

	time.Sleep(1 * time.Second)

	c.refreshBoard()

	if aiDice == 6 && moved {
		c.mu.Lock()
		c.currentDice = 0
		c.mu.Unlock()
		go c.playAITurns()
	} else {
		c.mu.Lock()
		c.currentDice = 0
		c.nextTurn()
		c.mu.Unlock()
	}
}

// ============================================================================
// LISTE DES JOUEURS
// ============================================================================

func (c *Client) createPlayersList() *widget.List {
	if c.gameState == nil || c.gameState.Room == nil {
		return widget.NewList(
			func() int { return 0 },
			func() fyne.CanvasObject { return widget.NewLabel("") },
			func(id widget.ListItemID, item fyne.CanvasObject) {},
		)
	}

	return widget.NewList(
		func() int { return len(c.gameState.Room.Players) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				canvas.NewCircle(color.White),
				widget.NewLabel("Player"),
				widget.NewLabel(""),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < len(c.gameState.Room.Players) {
				player := c.gameState.Room.Players[id]
				cont := item.(*fyne.Container)
				circle := cont.Objects[0].(*canvas.Circle)
				circle.FillColor = getColorForPlayerColor(player.Color)
				circle.Resize(fyne.NewSize(20, 20))
				circle.Refresh()

				label := cont.Objects[1].(*widget.Label)
				label.SetText(player.Username)

				turnMarker := cont.Objects[2].(*widget.Label)
				if c.gameState.Room.CurrentTurn == id {
					turnMarker.SetText(" ◄")
					label.TextStyle = fyne.TextStyle{Bold: true}
				} else {
					turnMarker.SetText("")
					label.TextStyle = fyne.TextStyle{}
				}
				label.Refresh()
				turnMarker.Refresh()
			}
		},
	)
}

// ============================================================================
// TAPPABLE RECTANGLE
// ============================================================================

type TappableRect struct {
	widget.BaseWidget
	size  float32
	onTap func(pos fyne.Position)
}

func NewTappableRect(size float32, onTap func(pos fyne.Position)) *TappableRect {
	t := &TappableRect{size: size, onTap: onTap}
	t.ExtendBaseWidget(t)
	return t
}

func (t *TappableRect) Tapped(pos *fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap(pos.Position)
	}
}

func (t *TappableRect) CreateRenderer() fyne.WidgetRenderer {
	rect := canvas.NewRectangle(color.NRGBA{0, 0, 0, 0})
	rect.Resize(fyne.NewSize(t.size, t.size))
	return &tappableRectRenderer{rect: rect, size: t.size}
}

type tappableRectRenderer struct {
	rect *canvas.Rectangle
	size float32
}

func (r *tappableRectRenderer) Layout(size fyne.Size)        { r.rect.Resize(size) }
func (r *tappableRectRenderer) MinSize() fyne.Size           { return fyne.NewSize(r.size, r.size) }
func (r *tappableRectRenderer) Refresh()                     {}
func (r *tappableRectRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.rect} }
func (r *tappableRectRenderer) Destroy()                     {}

// ============================================================================
// AUTRES MENUS
// ============================================================================

func (c *Client) showSettings() {
	dialog.ShowInformation("Settings", "Settings feature coming soon!", c.window)
}

func (c *Client) showLeaderboard() {
	dialog.ShowInformation("Leaderboard", "Leaderboard feature coming soon!", c.window)
}

// ============================================================================
// UTILITAIRES
// ============================================================================

func getColorForPlayerColor(playerColor constants.PlayerColor) color.Color {
	switch playerColor {
	case constants.ColorRed:
		return color.NRGBA{R: 230, G: 50, B: 50, A: 255}
	case constants.ColorGreen:
		return color.NRGBA{R: 50, G: 200, B: 50, A: 255}
	case constants.ColorYellow:
		return color.NRGBA{R: 255, G: 200, B: 50, A: 255}
	case constants.ColorBlue:
		return color.NRGBA{R: 50, G: 100, B: 230, A: 255}
	default:
		return color.Gray{Y: 128}
	}
}

// PARTIE 3 - Fonctions de dessin - À ajouter après la partie 2

// ============================================================================
// FONCTIONS DE DESSIN
// ============================================================================

func drawHomeZone(img *image.NRGBA, startCol, startRow int, cs float64, bgColor color.NRGBA) {
	// Fond coloré 6x6
	for r := 0; r < HOME_SIZE; r++ {
		for col := 0; col < HOME_SIZE; col++ {
			drawFilledRect(img, startCol+col, startRow+r, cs, bgColor)
		}
	}

	// Zone blanche intérieure 4x4
	for r := 1; r < 5; r++ {
		for col := 1; col < 5; col++ {
			drawFilledRect(img, startCol+col, startRow+r, cs, color.NRGBA{255, 255, 255, 255})
		}
	}

	// Cercles gris pour positions
	positions := [4][2]int{{1, 1}, {4, 1}, {1, 4}, {4, 4}}
	for _, p := range positions {
		cx := (float64(startCol+p[0]) + 0.5) * cs
		cy := (float64(startRow+p[1]) + 0.5) * cs
		drawCircle(img, cx, cy, cs*0.35, color.NRGBA{200, 200, 200, 255})
	}
}

func drawWhiteCell(img *image.NRGBA, col, row int, cs float64) {
	drawFilledRect(img, col, row, cs, color.NRGBA{255, 255, 255, 255})
	drawRectBorder(img, col, row, cs, color.NRGBA{0, 0, 0, 255})
}

func drawColoredCell(img *image.NRGBA, col, row int, cs float64, c color.NRGBA) {
	drawFilledRect(img, col, row, cs, c)
	drawRectBorder(img, col, row, cs, color.NRGBA{0, 0, 0, 255})
}

func drawStarCell(img *image.NRGBA, col, row int, cs float64, c color.NRGBA) {
	drawFilledRect(img, col, row, cs, color.NRGBA{255, 255, 255, 255})
	drawRectBorder(img, col, row, cs, color.NRGBA{0, 0, 0, 255})
	drawStar(img, col, row, cs, c)
}

func drawCenterTriangle(img *image.NRGBA, col, row int, cs float64) {
	cx := (float64(col) + 0.5) * cs
	cy := (float64(row) + 0.5) * cs
	size := cs * 0.7

	drawFilledRect(img, col, row, cs, color.NRGBA{255, 255, 255, 255})

	// 4 triangles colorés
	drawTriangle(img, cx, cy-size/3, cx-size/2, cy+size/3, cx+size/2, cy+size/3, redColor())
	drawTriangle(img, cx-size/3, cy, cx+size/3, cy-size/2, cx+size/3, cy+size/2, greenColor())
	drawTriangle(img, cx, cy+size/3, cx-size/2, cy-size/3, cx+size/2, cy-size/3, yellowColor())
	drawTriangle(img, cx+size/3, cy, cx-size/3, cy-size/2, cx-size/3, cy+size/2, blueColor())

	drawRectBorder(img, col, row, cs, color.NRGBA{0, 0, 0, 255})
}

func drawTriangle(img *image.NRGBA, x1, y1, x2, y2, x3, y3 float64, c color.NRGBA) {
	minX := int(math.Min(x1, math.Min(x2, x3)))
	maxX := int(math.Max(x1, math.Max(x2, x3)))
	minY := int(math.Min(y1, math.Min(y2, y3)))
	maxY := int(math.Max(y1, math.Max(y2, y3)))

	sign := func(px, py, ax, ay, bx, by float64) float64 {
		return (px-bx)*(ay-by) - (ax-bx)*(py-by)
	}

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px, py := float64(x), float64(y)
			d1 := sign(px, py, x1, y1, x2, y2)
			d2 := sign(px, py, x2, y2, x3, y3)
			d3 := sign(px, py, x3, y3, x1, y1)

			hasNeg := (d1 < 0) || (d2 < 0) || (d3 < 0)
			hasPos := (d1 > 0) || (d2 > 0) || (d3 > 0)

			if !(hasNeg && hasPos) {
				if x >= 0 && y >= 0 && x < img.Bounds().Max.X && y < img.Bounds().Max.Y {
					img.SetNRGBA(x, y, c)
				}
			}
		}
	}
}

func drawArrow(img *image.NRGBA, col, row int, cs float64, direction string, c color.NRGBA) {
	cx := (float64(col) + 0.5) * cs
	cy := (float64(row) + 0.5) * cs
	size := cs * 0.4

	var x1, y1, x2, y2, x3, y3 float64

	switch direction {
	case "right":
		x1, y1, x2, y2, x3, y3 = cx-size, cy-size/2, cx-size, cy+size/2, cx+size, cy
	case "left":
		x1, y1, x2, y2, x3, y3 = cx+size, cy-size/2, cx+size, cy+size/2, cx-size, cy
	case "down":
		x1, y1, x2, y2, x3, y3 = cx-size/2, cy-size, cx+size/2, cy-size, cx, cy+size
	case "up":
		x1, y1, x2, y2, x3, y3 = cx-size/2, cy+size, cx+size/2, cy+size, cx, cy-size
	}

	drawTriangle(img, x1, y1, x2, y2, x3, y3, c)
}

func drawCompleteGrid(img *image.NRGBA, width, height int, cs float64) {
	// Lignes horizontales
	for row := 0; row <= BOARD_GRID; row++ {
		y := int(float64(row) * cs)
		for x := 0; x < width; x++ {
			if y >= 0 && y < height {
				img.SetNRGBA(x, y, color.NRGBA{0, 0, 0, 255})
			}
		}
	}

	// Lignes verticales
	for col := 0; col <= BOARD_GRID; col++ {
		x := int(float64(col) * cs)
		for y := 0; y < height; y++ {
			if x >= 0 && x < width {
				img.SetNRGBA(x, y, color.NRGBA{0, 0, 0, 255})
			}
		}
	}
}

func drawFilledRect(img *image.NRGBA, col, row int, cs float64, c color.NRGBA) {
	x0 := int(math.Round(float64(col) * cs))
	y0 := int(math.Round(float64(row) * cs))
	x1 := int(math.Round(float64(col+1) * cs))
	y1 := int(math.Round(float64(row+1) * cs))

	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if x >= 0 && y >= 0 && x < img.Bounds().Max.X && y < img.Bounds().Max.Y {
				img.SetNRGBA(x, y, c)
			}
		}
	}
}

func drawRectBorder(img *image.NRGBA, col, row int, cs float64, c color.NRGBA) {
	x0 := int(math.Round(float64(col) * cs))
	y0 := int(math.Round(float64(row) * cs))
	x1 := int(math.Round(float64(col+1)*cs)) - 1
	y1 := int(math.Round(float64(row+1)*cs)) - 1

	// Top
	for x := x0; x <= x1; x++ {
		if x >= 0 && y0 >= 0 && x < img.Bounds().Max.X && y0 < img.Bounds().Max.Y {
			img.SetNRGBA(x, y0, c)
		}
	}
	// Bottom
	for x := x0; x <= x1; x++ {
		if x >= 0 && y1 >= 0 && x < img.Bounds().Max.X && y1 < img.Bounds().Max.Y {
			img.SetNRGBA(x, y1, c)
		}
	}
	// Left
	for y := y0; y <= y1; y++ {
		if x0 >= 0 && y >= 0 && x0 < img.Bounds().Max.X && y < img.Bounds().Max.Y {
			img.SetNRGBA(x0, y, c)
		}
	}
	// Right
	for y := y0; y <= y1; y++ {
		if x1 >= 0 && y >= 0 && x1 < img.Bounds().Max.X && y < img.Bounds().Max.Y {
			img.SetNRGBA(x1, y, c)
		}
	}
}

func drawCircle(img *image.NRGBA, cx, cy, radius float64, c color.NRGBA) {
	x0 := int(cx - radius - 1)
	y0 := int(cy - radius - 1)
	x1 := int(cx + radius + 1)
	y1 := int(cy + radius + 1)
	r2 := radius * radius

	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			if dx*dx+dy*dy <= r2 {
				if x >= 0 && y >= 0 && x < img.Bounds().Max.X && y < img.Bounds().Max.Y {
					img.SetNRGBA(x, y, c)
				}
			}
		}
	}
}

func drawCircleOutline(img *image.NRGBA, cx, cy, radius float64, c color.NRGBA, thickness int) {
	for t := 0; t < thickness; t++ {
		r := radius + float64(t) - float64(thickness)/2.0
		steps := int(2 * math.Pi * r * 2)
		if steps < 100 {
			steps = 100
		}

		for i := 0; i < steps; i++ {
			angle := 2 * math.Pi * float64(i) / float64(steps)
			x := int(math.Round(cx + r*math.Cos(angle)))
			y := int(math.Round(cy + r*math.Sin(angle)))
			if x >= 0 && y >= 0 && x < img.Bounds().Max.X && y < img.Bounds().Max.Y {
				img.SetNRGBA(x, y, c)
			}
		}
	}
}

func drawStar(img *image.NRGBA, col, row int, cs float64, c color.NRGBA) {
	cx := (float64(col) + 0.5) * cs
	cy := (float64(row) + 0.5) * cs
	outerR := cs * 0.25
	innerR := cs * 0.10
	points := 5

	var coords [][2]float64
	for i := 0; i < points*2; i++ {
		angle := math.Pi*float64(i)/float64(points) - math.Pi/2
		var r float64
		if i%2 == 0 {
			r = outerR
		} else {
			r = innerR
		}
		x := cx + r*math.Cos(angle)
		y := cy + r*math.Sin(angle)
		coords = append(coords, [2]float64{x, y})
	}

	for i := 0; i < len(coords); i++ {
		next := (i + 1) % len(coords)
		drawTriangle(img, cx, cy, coords[i][0], coords[i][1], coords[next][0], coords[next][1], c)
	}
}

func redColor() color.NRGBA    { return color.NRGBA{230, 50, 50, 255} }
func greenColor() color.NRGBA  { return color.NRGBA{50, 200, 50, 255} }
func yellowColor() color.NRGBA { return color.NRGBA{255, 200, 50, 255} }
func blueColor() color.NRGBA   { return color.NRGBA{50, 100, 230, 255} }
