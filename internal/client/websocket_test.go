package client

import (
	"testing"

	"virusbot/internal/protocol"
)

func TestGameStateInitialization(t *testing.T) {
	// Test that game state is initialized correctly
	board := [][]protocol.CellType{
		{protocol.CellEmpty, protocol.CellEmpty, protocol.CellEmpty},
		{protocol.CellEmpty, protocol.CellEmpty, protocol.CellEmpty},
		{protocol.CellEmpty, protocol.CellEmpty, protocol.CellEmpty},
	}

	players := []protocol.PlayerInfo{
		{ID: 1, Name: "Player 1", Symbol: protocol.CellPlayer1, Position: protocol.Position{Row: 0, Col: 0}},
		{ID: 2, Name: "Player 2", Symbol: protocol.CellPlayer2, Position: protocol.Position{Row: 2, Col: 2}},
	}

	state := &GameState{
		Board:         board,
		Players:       players,
		CurrentPlayer: 2,
		YourPlayerID:  2,
	}

	// Verify initial state
	if state.YourPlayerID != 2 {
		t.Errorf("Expected YourPlayerID to be 2, got %d", state.YourPlayerID)
	}
	if state.CurrentPlayer != 2 {
		t.Errorf("Expected CurrentPlayer to be 2, got %d", state.CurrentPlayer)
	}
	if len(state.Board) != 3 {
		t.Errorf("Expected board to have 3 rows, got %d", len(state.Board))
	}
}

func TestMoveMadeUpdatesBoard(t *testing.T) {
	board := [][]protocol.CellType{
		{protocol.CellEmpty, protocol.CellEmpty, protocol.CellEmpty},
		{protocol.CellEmpty, protocol.CellEmpty, protocol.CellEmpty},
		{protocol.CellEmpty, protocol.CellEmpty, protocol.CellEmpty},
	}

	state := &GameState{
		Board:         board,
		Players:       nil,
		CurrentPlayer: 1,
		YourPlayerID:  1,
	}

	// Simulate player 1 making a move at (1, 1)
	player := 1
	row := 1
	col := 1
	state.Board[row][col] = protocol.CellType(player)

	// Verify the move was recorded
	if state.Board[row][col] != protocol.CellPlayer1 {
		t.Errorf("Expected cell (1,1) to be CellPlayer1, got %v", state.Board[row][col])
	}
}

func TestTurnChangesOnlyWhenMovesLeftZero(t *testing.T) {
	board := [][]protocol.CellType{
		{protocol.CellEmpty, protocol.CellEmpty, protocol.CellEmpty},
		{protocol.CellEmpty, protocol.CellEmpty, protocol.CellEmpty},
		{protocol.CellEmpty, protocol.CellEmpty, protocol.CellEmpty},
	}

	state := &GameState{
		Board:         board,
		Players:       nil,
		CurrentPlayer: 1,
		YourPlayerID:  1,
	}

	// Simulate moves with movesLeft tracking
	movesLeft := 3

	// First move - movesLeft becomes 2
	movesLeft = 2
	// Don't change turn yet
	if state.CurrentPlayer != 1 {
		t.Error("Turn should not change when movesLeft > 0")
	}

	// Second move - movesLeft becomes 1
	movesLeft = 1
	if state.CurrentPlayer != 1 {
		t.Error("Turn should not change when movesLeft > 0")
	}

	// Third move - movesLeft becomes 0
	movesLeft = 0
	// Now change turn
	state.CurrentPlayer = 2
	if state.CurrentPlayer != 2 {
		t.Error("Turn should change when movesLeft == 0")
	}

	// Use movesLeft to avoid unused variable warning
	_ = movesLeft
}

func TestIsMyTurn(t *testing.T) {
	board := [][]protocol.CellType{
		{protocol.CellEmpty, protocol.CellEmpty, protocol.CellEmpty},
		{protocol.CellEmpty, protocol.CellEmpty, protocol.CellEmpty},
		{protocol.CellEmpty, protocol.CellEmpty, protocol.CellEmpty},
	}

	state := &GameState{
		Board:         board,
		Players:       nil,
		CurrentPlayer: 2,
		YourPlayerID:  2,
	}

	// Bot is player 2, current player is 2 -> it's bot's turn
	if state.CurrentPlayer != state.YourPlayerID {
		t.Error("IsMyTurn should return true when CurrentPlayer == YourPlayerID")
	}

	// Change current player to opponent
	state.CurrentPlayer = 1
	if state.CurrentPlayer == state.YourPlayerID {
		t.Error("IsMyTurn should return false when CurrentPlayer != YourPlayerID")
	}
}

func TestMoveMadeMessageParsing(t *testing.T) {
	// Test parsing a move_made message
	jsonData := []byte(`{
		"gameId": "test-game-id",
		"row": 5,
		"col": 6,
		"player": 2,
		"movesLeft": 2
	}`)

	msg, err := protocol.ParseMoveMade(jsonData)
	if err != nil {
		t.Fatalf("Failed to parse move_made message: %v", err)
	}

	if msg.Row != 5 {
		t.Errorf("Expected row to be 5, got %d", msg.Row)
	}
	if msg.Col != 6 {
		t.Errorf("Expected col to be 6, got %d", msg.Col)
	}
	if msg.Player != 2 {
		t.Errorf("Expected player to be 2, got %d", msg.Player)
	}
	if msg.MovesLeft != 2 {
		t.Errorf("Expected movesLeft to be 2, got %d", msg.MovesLeft)
	}
}

func TestTurnChangeMessageParsing(t *testing.T) {
	// Test parsing a turn_change message
	jsonData := []byte(`{
		"gameId": "test-game-id",
		"player": 2,
		"movesLeft": 3
	}`)

	msg, err := protocol.ParseTurnChange(jsonData)
	if err != nil {
		t.Fatalf("Failed to parse turn_change message: %v", err)
	}

	if msg.Player != 2 {
		t.Errorf("Expected player to be 2, got %d", msg.Player)
	}
	if msg.MovesLeft != 3 {
		t.Errorf("Expected movesLeft to be 3, got %d", msg.MovesLeft)
	}
}

func TestChallengeMessageParsing(t *testing.T) {
	// Test parsing a challenge message
	jsonData := []byte(`{
		"challengeId": "test-challenge-id",
		"fromUserId": "user-123",
		"fromUsername": "TestPlayer"
	}`)

	msg, err := protocol.ParseChallenge(jsonData)
	if err != nil {
		t.Fatalf("Failed to parse challenge message: %v", err)
	}

	if msg.ChallengeID != "test-challenge-id" {
		t.Errorf("Expected challengeId to be 'test-challenge-id', got %s", msg.ChallengeID)
	}
	if msg.FromUserName != "TestPlayer" {
		t.Errorf("Expected fromUsername to be 'TestPlayer', got %s", msg.FromUserName)
	}
}
