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
	MsgGameStart  MessageType = "game_start"
	MsgMove       MessageType = "move"
	MsgMoveMade   MessageType = "move_made"
	MsgTurnChange MessageType = "turn_change"
	MsgGameEnd    MessageType = "game_end"

	// Challenge messages
	MsgChallenge        MessageType = "challenge_received"
	MsgAcceptChallenge  MessageType = "accept_challenge"
	MsgDeclineChallenge MessageType = "decline_challenge"
)

// Cell flags (encoded in high 2 bits)
const (
	CellFlagNormal    byte = 0x00
	CellFlagBase      byte = 0x10
	CellFlagFortified byte = 0x20
	CellFlagKilled    byte = 0x30

	FlagMask   byte = 0x30
	PlayerMask byte = 0x0F
)

// CellType represents the type of cell on the board
// It encodes both the player ID and cell flags using bit fields:
// - Low 4 bits (0x0F): Player ID (0=empty, 1-4=players, 5=neutral)
// - High 2 bits (0x30): Flags (0x00=normal, 0x10=base, 0x20=fortified, 0x30=killed/neutral)
type CellType int

const (
	CellEmpty   CellType = 0
	CellPlayer1 CellType = 1
	CellPlayer2 CellType = 2
	CellPlayer3 CellType = 3
	CellPlayer4 CellType = 4
	CellNeutral CellType = 5
)

// Player extracts the player ID from a CellType
func (c CellType) Player() int {
	return int(byte(c) & PlayerMask)
}

// Flag extracts the cell flag from a CellType
func (c CellType) Flag() byte {
	return byte(c) & FlagMask
}

// IsBase returns true if the cell is a base cell
func (c CellType) IsBase() bool {
	return c.Flag() == CellFlagBase
}

// IsFortified returns true if the cell is fortified
func (c CellType) IsFortified() bool {
	return c.Flag() == CellFlagFortified
}

// IsKilled returns true if the cell is killed/neutral
func (c CellType) IsKilled() bool {
	return c.Flag() == CellFlagKilled
}

// CanBeAttacked returns true if the cell can be attacked (only normal cells)
func (c CellType) CanBeAttacked() bool {
	return c.Flag() == CellFlagNormal
}

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
	UserID   string `json:"userId"`
	UserName string `json:"username"`
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

// GameStartV2Message is sent when a game begins (new format without board data)
type GameStartV2Message struct {
	GameID           string `json:"gameId"`
	OpponentID       string `json:"opponentId"`
	OpponentUsername string `json:"opponentUsername"`
	YourPlayer       int    `json:"yourPlayer"`
	Rows             int    `json:"rows"`
	Cols             int    `json:"cols"`
}

// MoveMessage is sent to make a move
type MoveMessage struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

// MoveMadeMessage is broadcast when a move is made
type MoveMadeMessage struct {
	GameID    string `json:"gameId"`
	Row       int    `json:"row"`
	Col       int    `json:"col"`
	Player    int    `json:"player"`
	MovesLeft int    `json:"movesLeft"`
}

// GameEndMessage is sent when the game ends
type GameEndMessage struct {
	Winner     int    `json:"winner"`
	Eliminated []int  `json:"eliminated,omitempty"`
	Message    string `json:"message,omitempty"`
}

// TurnChangeMessage is sent when the turn changes
type TurnChangeMessage struct {
	GameID    string `json:"gameId"`
	Player    int    `json:"player"`
	MovesLeft int    `json:"movesLeft"`
}

// ParseTurnChange parses a turn change message
func ParseTurnChange(data []byte) (*TurnChangeMessage, error) {
	var msg TurnChangeMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
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

// ParseGameStartV2 parses a game start message (new format)
func ParseGameStartV2(data []byte) (*GameStartV2Message, error) {
	var msg GameStartV2Message
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

// ChallengeMessage contains challenge information
type ChallengeMessage struct {
	ChallengeID  string `json:"challengeId"`
	FromUserID   string `json:"fromUserId"`
	FromUserName string `json:"fromUsername"`
}

// ParseChallenge parses a challenge message
func ParseChallenge(data []byte) (*ChallengeMessage, error) {
	var msg ChallengeMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// NewAcceptChallengeMessage creates an accept challenge message
func NewAcceptChallengeMessage(challengeID string) *Message {
	return &Message{
		Type: MsgAcceptChallenge,
		Data: map[string]interface{}{"challengeId": challengeID},
	}
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
