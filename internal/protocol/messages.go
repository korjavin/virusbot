package protocol

import "encoding/json"

// MessageType represents the type of WebSocket message
type MessageType string

const (
	// Connection messages
	MsgConnect     MessageType = "connect"
	MsgWelcome     MessageType = "welcome"
	MsgUsersUpdate MessageType = "users_update"

	// Lobby messages
	MsgCreateLobby      MessageType = "create_lobby"
	MsgJoinLobby        MessageType = "join_lobby"
	MsgLeaveLobby       MessageType = "leave_lobby"
	MsgAddBot           MessageType = "add_bot"
	MsgBotWanted        MessageType = "bot_wanted"
	MsgRemoveBot        MessageType = "remove_bot"
	MsgStartMultiplayer MessageType = "start_multiplayer_game"

	// Game messages
	MsgGameStart MessageType = "game_start"
	MsgMove      MessageType = "move"
	MsgMoveMade  MessageType = "move_made"
	MsgGameEnd   MessageType = "game_end"

	// Challenge messages (legacy)
	MsgChallenge        MessageType = "challenge"
	MsgAcceptChallenge  MessageType = "accept_challenge"
	MsgDeclineChallenge MessageType = "decline_challenge"
)

// CellType represents the type of cell on the board
type CellType int

const (
	CellEmpty   CellType = iota
	CellPlayer1          // X
	CellPlayer2          // O
	CellPlayer3          // Triangle
	CellPlayer4          // Square
	CellNeutral
)

// Position represents a cell on the board
type Position struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

// PlayerInfo contains information about a player
type PlayerInfo struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Symbol   CellType `json:"symbol"`
	Position Position `json:"position"`
	IsAI     bool     `json:"isAI,omitempty"`
}

// Message is the base WebSocket message structure
type Message struct {
	Type MessageType `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// WelcomeMessage is sent when a client connects
type WelcomeMessage struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
}

// UsersUpdateMessage contains the list of online users
type UsersUpdateMessage struct {
	Users []UserInfo `json:"users"`
}

// UserInfo contains user details for the user list
type UserInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Status  string `json:"status"` // "idle", "in_lobby", "in_game"
	LobbyID string `json:"lobbyId,omitempty"`
}

// CreateLobbyMessage is sent to create a new lobby
type CreateLobbyMessage struct {
	BoardSize int `json:"boardSize"`
}

// JoinLobbyMessage is sent to join an existing lobby
type JoinLobbyMessage struct {
	LobbyID string `json:"lobbyId"`
}

// LobbyMessage is the response when joining/creating a lobby
type LobbyMessage struct {
	LobbyID   string       `json:"lobbyId"`
	Players   []PlayerInfo `json:"players"`
	HostID    int          `json:"hostId"`
	BoardSize int          `json:"boardSize"`
}

// GameStartMessage is sent when a game begins
type GameStartMessage struct {
	Board         [][]CellType `json:"board"`
	Players       []PlayerInfo `json:"players"`
	CurrentPlayer int          `json:"currentPlayer"`
	YourPlayerID  int          `json:"yourPlayerId"`
}

// MoveMessage is sent to make a move
type MoveMessage struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

// MoveMadeMessage is broadcast when a move is made
type MoveMadeMessage struct {
	Row      int      `json:"row"`
	Col      int      `json:"col"`
	PlayerID int      `json:"playerId"`
	CellType CellType `json:"cellType"`
}

// GameEndMessage is sent when the game ends
type GameEndMessage struct {
	Winner     int    `json:"winner"`
	Eliminated []int  `json:"eliminated,omitempty"`
	Message    string `json:"message,omitempty"`
}

// ParseMessage parses a raw JSON message into a structured message
func ParseMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// ParseWelcome parses a welcome message
func ParseWelcome(data []byte) (*WelcomeMessage, error) {
	var msg WelcomeMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// ParseGameStart parses a game start message
func ParseGameStart(data []byte) (*GameStartMessage, error) {
	var msg GameStartMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// ParseMoveMade parses a move made message
func ParseMoveMade(data []byte) (*MoveMadeMessage, error) {
	var msg MoveMadeMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// ParseGameEnd parses a game end message
func ParseGameEnd(data []byte) (*GameEndMessage, error) {
	var msg GameEndMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// NewMessage creates a new message with the given type and data
func NewMessage(msgType MessageType, data interface{}) *Message {
	return &Message{
		Type: msgType,
		Data: data,
	}
}

// NewMoveMessage creates a move message
func NewMoveMessage(row, col int) *Message {
	return NewMessage(MsgMove, MoveMessage{Row: row, Col: col})
}

// NewJoinLobbyMessage creates a join lobby message
func NewJoinLobbyMessage(lobbyID string) *Message {
	return NewMessage(MsgJoinLobby, JoinLobbyMessage{LobbyID: lobbyID})
}

// NewCreateLobbyMessage creates a create lobby message
func NewCreateLobbyMessage(boardSize int) *Message {
	return NewMessage(MsgCreateLobby, CreateLobbyMessage{BoardSize: boardSize})
}
