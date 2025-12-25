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
	conn      *websocket.Conn
	config    *config.Config
	userID    string
	userName  string
	gameState *GameState
	callback  Callback
	incoming  chan []byte
	mu        sync.RWMutex
	connected bool
	ctx       context.Context
	cancel    context.CancelFunc
	moveDelay time.Duration
	debug     bool
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

	switch msg.Type {
	case protocol.MsgWelcome:
		return c.handleWelcome(msg.Data)

	case protocol.MsgGameStart:
		return c.handleGameStart(msg.Data)

	case protocol.MsgMoveMade:
		return c.handleMoveMade(msg.Data)

	case protocol.MsgGameEnd:
		return c.handleGameEnd(msg.Data)

	case protocol.MsgUsersUpdate:
		c.handleUsersUpdate(msg.Data)

	default:
		if c.debug {
			log.Printf("Unhandled message type: %s", msg.Type)
		}
	}

	return nil
}

// handleWelcome handles the welcome message after connection
func (c *Client) handleWelcome(data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	welcome, err := protocol.ParseWelcome(jsonData)
	if err != nil {
		return err
	}

	c.userID = welcome.UserID
	c.userName = welcome.Name

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
func (c *Client) handleGameStart(data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	gameStart, err := protocol.ParseGameStart(jsonData)
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
		log.Printf("Game started! You are player %d", gameStart.YourPlayerID)
	}

	if c.callback != nil {
		c.callback("game_start", c.gameState)
	}

	return nil
}

// handleMoveMade handles a move being made
func (c *Client) handleMoveMade(data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	moveMade, err := protocol.ParseMoveMade(jsonData)
	if err != nil {
		return err
	}

	c.mu.Lock()
	if c.gameState != nil {
		c.gameState.Board[moveMade.Row][moveMade.Col] = moveMade.CellType
		c.gameState.CurrentPlayer = (c.gameState.CurrentPlayer + 1) % len(c.gameState.Players)
	}
	c.mu.Unlock()

	if c.debug {
		log.Printf("Player %d moved to (%d, %d)", moveMade.PlayerID, moveMade.Row, moveMade.Col)
	}

	if c.callback != nil {
		c.callback("move_made", moveMade)
	}

	return nil
}

// handleGameEnd handles the end of a game
func (c *Client) handleGameEnd(data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	gameEnd, err := protocol.ParseGameEnd(jsonData)
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

// handleUsersUpdate handles the list of online users
func (c *Client) handleUsersUpdate(data interface{}) {
	if c.callback != nil {
		c.callback("users_update", data)
	}
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

	msg := protocol.NewMoveMessage(row, col)
	return c.SendMessage(msg)
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
	return c.gameState != nil && c.gameState.CurrentPlayer == c.gameState.YourPlayerID
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
