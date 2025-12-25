package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"virusbot/config"
	"virusbot/internal/protocol"

	"github.com/gorilla/websocket"
)

// GameState represents the current state of the game
type GameState struct {
	Board         [][]protocol.CellType
	Players       []protocol.PlayerInfo
	CurrentPlayer int
	YourPlayerID  int
}

// Callback is a function that handles game events
type Callback func(event string, data interface{})

// Client represents a WebSocket client for the game
type Client struct {
	conn             *websocket.Conn
	config           *config.Config
	userID           string
	userName         string
	gameState        *GameState
	callback         Callback
	incoming         chan []byte
	mu               sync.RWMutex
	connected        bool
	ctx              context.Context
	cancel           context.CancelFunc
	moveDelay        time.Duration
	debug            bool
	currentChallenge string
	gameID           string
}

// NewClient creates a new WebSocket client
func NewClient(cfg *config.Config, callback Callback) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		config:    cfg,
		callback:  callback,
		incoming:  make(chan []byte, 100),
		ctx:       ctx,
		cancel:    cancel,
		moveDelay: cfg.MoveDelay,
		debug:     cfg.Debug,
	}
}

// Connect establishes a WebSocket connection
func (c *Client) Connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(c.config.ServerURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	c.conn = conn
	c.connected = true

	if c.debug {
		log.Printf("Connected to %s", c.config.ServerURL)
	}

	return nil
}

// Run starts the message handling loop
func (c *Client) Run() error {
	go c.readLoop()
	return c.writeLoop()
}

// readLoop continuously reads messages from the WebSocket
func (c *Client) readLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, data, err := c.conn.ReadMessage()
			if err != nil {
				if c.debug {
					log.Printf("Read error: %v", err)
				}
				c.handleDisconnect()
				return
			}
			c.incoming <- data
		}
	}
}

// writeLoop processes incoming messages
func (c *Client) writeLoop() error {
	for {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		case data := <-c.incoming:
			if err := c.handleMessage(data); err != nil {
				if c.debug {
					log.Printf("Message handling error: %v", err)
				}
				return err
			}
		}
	}
}

// handleMessage processes a single WebSocket message
func (c *Client) handleMessage(data []byte) error {
	msg, err := protocol.ParseMessage(data)
	if err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}

	if c.debug {
		log.Printf("Raw message: %s", string(data))
	}

	switch msg.Type {
	case protocol.MsgWelcome:
		return c.handleWelcome(data)

	case protocol.MsgChallenge:
		return c.handleChallenge(data)

	case protocol.MsgGameStart:
		return c.handleGameStart(data)

	case protocol.MsgMoveMade:
		return c.handleMoveMade(data)

	case protocol.MsgTurnChange:
		return c.handleTurnChange(data)

	case protocol.MsgGameEnd:
		return c.handleGameEnd(data)

	case protocol.MsgUsersUpdate:
		c.handleUsersUpdate(data)

	default:
		if c.debug {
			log.Printf("Unhandled message type: %s", msg.Type)
		}
	}

	return nil
}

// handleWelcome handles the welcome message after connection
func (c *Client) handleWelcome(data []byte) error {
	if c.debug {
		log.Printf("Welcome data: %s", string(data))
	}

	welcome, err := protocol.ParseWelcome(data)
	if err != nil {
		return err
	}

	c.userID = welcome.UserID
	c.userName = welcome.UserName

	if c.debug {
		log.Printf("Connected as %s (ID: %s)", c.userName, c.userID)
	}

	if c.callback != nil {
		c.callback("connected", welcome)
	}

	// Auto-join or create lobby if configured
	if c.config.LobbyID != "" {
		return c.JoinLobby(c.config.LobbyID)
	}
	if c.config.AutoJoin {
		// Would need to get lobby list first
		if c.debug {
			log.Println("Auto-join enabled but no lobby ID specified")
		}
	}
	if c.config.AutoCreate {
		return c.CreateLobby(10)
	}

	return nil
}

// handleGameStart handles the start of a game
func (c *Client) handleGameStart(data []byte) error {
	// Try to parse as new format first (without board data)
	gameStartV2, err := protocol.ParseGameStartV2(data)
	if err == nil && gameStartV2.Rows > 0 {
		// New format: initialize empty board
		board := make([][]protocol.CellType, gameStartV2.Rows)
		for i := range board {
			board[i] = make([]protocol.CellType, gameStartV2.Cols)
		}

		// Create players - we need base positions for valid moves
		// In the new protocol, base positions come from initial board state
		// For now, create placeholder players
		players := []protocol.PlayerInfo{
			{ID: 1, Name: "Player 1", Symbol: protocol.CellPlayer1, Position: protocol.Position{Row: 0, Col: 0}, IsAI: true},
			{ID: 2, Name: "Player 2", Symbol: protocol.CellPlayer2, Position: protocol.Position{Row: gameStartV2.Rows - 1, Col: gameStartV2.Cols - 1}, IsAI: true},
		}

		c.mu.Lock()
		c.gameState = &GameState{
			Board:         board,
			Players:       players,
			CurrentPlayer: gameStartV2.YourPlayer,
			YourPlayerID:  gameStartV2.YourPlayer,
		}
		c.gameID = gameStartV2.GameID
		c.mu.Unlock()

		if c.debug {
			log.Printf("Game started: you are player %d (gameId: %s)", gameStartV2.YourPlayer, gameStartV2.GameID)
		}
	} else {
		// Old format with board data
		gameStart, err := protocol.ParseGameStart(data)
		if err != nil {
			return err
		}

		c.mu.Lock()
		c.gameState = &GameState{
			Board:         gameStart.Board,
			Players:       gameStart.Players,
			CurrentPlayer: gameStart.CurrentPlayer,
			YourPlayerID:  gameStart.YourPlayerID,
		}
		c.mu.Unlock()

		if c.debug {
			log.Printf("Game started: you are player %d", gameStart.YourPlayerID)
		}
	}

	if c.callback != nil {
		c.callback("game_start", c.gameState)
	}

	return nil
}

// handleMoveMade handles a move being made
func (c *Client) handleMoveMade(data []byte) error {
	moveMade, err := protocol.ParseMoveMade(data)
	if err != nil {
		return err
	}

	c.mu.Lock()
	if c.gameState != nil && c.gameState.Board != nil && len(c.gameState.Board) > moveMade.Row {
		if len(c.gameState.Board[moveMade.Row]) > moveMade.Col {
			// Mark the cell with the player's cell type
			cellType := protocol.CellType(moveMade.Player)
			c.gameState.Board[moveMade.Row][moveMade.Col] = cellType
		}
		// Only change turn when movesLeft reaches 0
		if moveMade.MovesLeft == 0 {
			c.gameState.CurrentPlayer = (c.gameState.CurrentPlayer + 1) % 2
		}
	}
	c.mu.Unlock()

	if c.debug {
		log.Printf("Player %d moved to (%d, %d), movesLeft=%d", moveMade.Player, moveMade.Row, moveMade.Col, moveMade.MovesLeft)
	}

	if c.callback != nil {
		c.callback("move_made", moveMade)
	}

	return nil
}

// handleGameEnd handles the end of a game
func (c *Client) handleGameEnd(data []byte) error {
	gameEnd, err := protocol.ParseGameEnd(data)
	if err != nil {
		return err
	}

	if c.debug {
		log.Printf("Game ended! Winner: Player %d", gameEnd.Winner)
	}

	if c.callback != nil {
		c.callback("game_end", gameEnd)
	}

	return nil
}

// handleTurnChange handles turn change notifications
func (c *Client) handleTurnChange(data []byte) error {
	turnChange, err := protocol.ParseTurnChange(data)
	if err != nil {
		return err
	}

	c.mu.Lock()
	if c.gameState != nil {
		c.gameState.CurrentPlayer = turnChange.Player
		log.Printf("Turn changed to player %d", turnChange.Player)
	} else {
		log.Printf("Turn change ignored: no game state")
	}
	c.mu.Unlock()

	return nil
}

// handleUsersUpdate handles the list of online users
func (c *Client) handleUsersUpdate(data interface{}) {
	if c.callback != nil {
		c.callback("users_update", data)
	}
}

// handleChallenge handles incoming challenge messages
func (c *Client) handleChallenge(data []byte) error {
	if c.debug {
		log.Printf("Challenge data: %s", string(data))
	}

	challenge, err := protocol.ParseChallenge(data)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.currentChallenge = challenge.ChallengeID
	c.mu.Unlock()

	if c.debug {
		log.Printf("Challenge received from %s (ID: %s)", challenge.FromUserName, challenge.ChallengeID)
	}

	if c.callback != nil {
		if c.debug {
			log.Printf("Calling challenge callback...")
		}
		c.callback("challenge", challenge)
		if c.debug {
			log.Printf("Challenge callback returned")
		}
	}

	// Auto-accept challenge if configured
	if c.debug {
		log.Printf("AutoAcceptChallenge: %v", c.config.AutoAcceptChallenge)
	}
	if c.config.AutoAcceptChallenge {
		return c.AcceptChallenge(challenge.ChallengeID)
	}

	return nil
}

// AcceptChallenge accepts a challenge by ID
func (c *Client) AcceptChallenge(challengeID string) error {
	if c.debug {
		log.Printf("Accepting challenge: %s", challengeID)
	}

	// Send the correct format without nested "data" field
	msg := map[string]interface{}{
		"type":        protocol.MsgAcceptChallenge,
		"challengeId": challengeID,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal accept challenge: %w", err)
	}

	if c.debug {
		log.Printf("Sending message: %s", string(data))
	}

	c.mu.RLock()
	connected := c.connected
	c.mu.RUnlock()

	if !connected {
		return fmt.Errorf("not connected")
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// handleDisconnect handles connection loss
func (c *Client) handleDisconnect() {
	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()

	if c.callback != nil {
		c.callback("disconnected", nil)
	}
}

// SendMessage sends a message to the server
func (c *Client) SendMessage(msg *protocol.Message) error {
	c.mu.RLock()
	connected := c.connected
	c.mu.RUnlock()

	if !connected {
		return fmt.Errorf("not connected")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if c.debug {
		log.Printf("Sending message: %s", string(data))
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// MakeMove sends a move to the server
func (c *Client) MakeMove(row, col int) error {
	// Add delay if configured
	if c.moveDelay > 0 {
		time.Sleep(c.moveDelay)
	}

	c.mu.RLock()
	gameID := c.gameID
	c.mu.RUnlock()

	// Send with correct format (no nested data field)
	msg := map[string]interface{}{
		"type":  protocol.MsgMove,
		"row":   row,
		"col":   col,
		"gameId": gameID,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal move: %w", err)
	}

	if c.debug {
		log.Printf("Sending move: %s", string(data))
	}

	c.mu.RLock()
	connected := c.connected
	c.mu.RUnlock()

	if !connected {
		return fmt.Errorf("not connected")
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("failed to send move: %w", err)
	}

	return nil
}

// CreateLobby creates a new game lobby
func (c *Client) CreateLobby(boardSize int) error {
	msg := protocol.NewCreateLobbyMessage(boardSize)
	return c.SendMessage(msg)
}

// JoinLobby joins an existing lobby
func (c *Client) JoinLobby(lobbyID string) error {
	msg := protocol.NewJoinLobbyMessage(lobbyID)
	return c.SendMessage(msg)
}

// GetGameState returns the current game state
func (c *Client) GetGameState() *GameState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.gameState
}

// IsMyTurn returns true if it's the bot's turn
func (c *Client) IsMyTurn() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.gameState == nil {
		return false
	}
	return c.gameState.CurrentPlayer == c.gameState.YourPlayerID
}

// GetUserID returns the user's ID
func (c *Client) GetUserID() string {
	return c.userID
}

// GetUserName returns the user's name
func (c *Client) GetUserName() string {
	return c.userName
}

// IsConnected returns the connection status
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// Disconnect closes the WebSocket connection
func (c *Client) Disconnect() {
	c.cancel()
	if c.conn != nil {
		c.conn.Close()
	}
}
