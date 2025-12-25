package game

import (
	"testing"

	"virusbot/internal/protocol"
)

func TestIsAdjacent(t *testing.T) {
	board := NewBoard(5)

	tests := []struct {
		pos1     Position
		pos2     Position
		adjacent bool
	}{
		{pos1: Position{Row: 0, Col: 0}, pos2: Position{Row: 0, Col: 1}, adjacent: true},
		{pos1: Position{Row: 0, Col: 0}, pos2: Position{Row: 1, Col: 0}, adjacent: true},
		{pos1: Position{Row: 0, Col: 0}, pos2: Position{Row: 1, Col: 1}, adjacent: false},
		{pos1: Position{Row: 0, Col: 0}, pos2: Position{Row: 0, Col: 2}, adjacent: false},
		{pos1: Position{Row: 2, Col: 2}, pos2: Position{Row: 2, Col: 3}, adjacent: true},
		{pos1: Position{Row: 2, Col: 2}, pos2: Position{Row: 3, Col: 2}, adjacent: true},
		{pos1: Position{Row: 2, Col: 2}, pos2: Position{Row: 2, Col: 2}, adjacent: false},
	}

	for _, tt := range tests {
		if board.IsAdjacent(tt.pos1, tt.pos2) != tt.adjacent {
			t.Errorf("IsAdjacent(%v, %v) = %v, want %v", tt.pos1, tt.pos2, board.IsAdjacent(tt.pos1, tt.pos2), tt.adjacent)
		}
	}
}

func TestIsConnectedToBase(t *testing.T) {
	board := NewBoard(5)
	board.BasePos[1] = Position{Row: 0, Col: 0} // Player 1 base at top-left

	// Place some player cells
	board.SetCell(Position{Row: 0, Col: 0}, protocol.CellPlayer1) // Base
	board.SetCell(Position{Row: 0, Col: 1}, protocol.CellPlayer1)
	board.SetCell(Position{Row: 1, Col: 0}, protocol.CellPlayer1)

	tests := []struct {
		playerID  int
		pos       Position
		connected bool
	}{
		{playerID: 1, pos: Position{Row: 0, Col: 0}, connected: true},  // Base
		{playerID: 1, pos: Position{Row: 0, Col: 1}, connected: true},  // Adjacent to base
		{playerID: 1, pos: Position{Row: 1, Col: 0}, connected: true},  // Adjacent to base
		{playerID: 1, pos: Position{Row: 1, Col: 1}, connected: false}, // Not connected
		{playerID: 1, pos: Position{Row: 0, Col: 2}, connected: false}, // Not connected
	}

	for _, tt := range tests {
		if board.IsConnectedToBase(tt.playerID, tt.pos) != tt.connected {
			t.Errorf("IsConnectedToBase(player %d, %v) = %v, want %v", tt.playerID, tt.pos, board.IsConnectedToBase(tt.playerID, tt.pos), tt.connected)
		}
	}
}

func TestGetReachableCells(t *testing.T) {
	board := NewBoard(5)
	board.BasePos[1] = Position{Row: 0, Col: 0}

	// Create a connected chain
	board.SetCell(Position{Row: 0, Col: 0}, protocol.CellPlayer1)
	board.SetCell(Position{Row: 0, Col: 1}, protocol.CellPlayer1)
	board.SetCell(Position{Row: 0, Col: 2}, protocol.CellPlayer1)

	// Create a disconnected group
	board.SetCell(Position{Row: 4, Col: 4}, protocol.CellPlayer1)

	reachable := board.GetReachableCells(0)

	// Should find 3 connected cells
	if len(reachable) != 3 {
		t.Errorf("Expected 3 reachable cells, got %d", len(reachable))
	}

	// Check that we found the connected cells
	found := make(map[Position]bool)
	for _, pos := range reachable {
		found[pos] = true
	}

	if !found[Position{Row: 0, Col: 0}] {
		t.Error("Base position should be reachable")
	}
	if !found[Position{Row: 0, Col: 1}] {
		t.Error("Position (0,1) should be reachable")
	}
	if !found[Position{Row: 0, Col: 2}] {
		t.Error("Position (0,2) should be reachable")
	}
	if found[Position{Row: 4, Col: 4}] {
		t.Error("Position (4,4) should NOT be reachable (disconnected)")
	}
}

func TestGetValidMoves(t *testing.T) {
	board := NewBoard(5)
	board.BasePos[1] = Position{Row: 0, Col: 0}
	board.BasePos[1] = Position{Row: 4, Col: 4}

	// Set up player's territory
	board.SetCell(Position{Row: 0, Col: 0}, protocol.CellPlayer1)
	board.SetCell(Position{Row: 0, Col: 1}, protocol.CellPlayer1)
	board.SetCell(Position{Row: 0, Col: 2}, protocol.CellPlayer1)

	// Set up opponent's territory
	board.SetCell(Position{Row: 4, Col: 4}, protocol.CellPlayer2)
	board.SetCell(Position{Row: 4, Col: 3}, protocol.CellPlayer2)

	moves := board.GetValidMoves(0)

	// Should find moves around the player's territory
	if len(moves) == 0 {
		t.Error("Expected some valid moves")
	}

	// Verify all moves are valid
	for _, move := range moves {
		if !board.IsValid(move.Position) {
			t.Errorf("Move has invalid position %v", move.Position)
		}
	}
}

func TestGetAttackMoves(t *testing.T) {
	board := NewBoard(5)
	board.BasePos[1] = Position{Row: 0, Col: 0}
	board.BasePos[1] = Position{Row: 0, Col: 4}

	// Player 0 at (0,0)
	board.SetCell(Position{Row: 0, Col: 0}, protocol.CellPlayer1)

	// Player 1 at (0,4) with neighbor at (0,3)
	board.SetCell(Position{Row: 0, Col: 4}, protocol.CellPlayer2)
	board.SetCell(Position{Row: 0, Col: 3}, protocol.CellPlayer2)

	// Player 0 has an attack available at (0,1)
	board.SetCell(Position{Row: 0, Col: 1}, protocol.CellPlayer1)
	board.SetCell(Position{Row: 0, Col: 2}, protocol.CellPlayer2) // This is adjacent to (0,1)

	attacks := board.GetAttackMoves(0)

	// Should find the attack at (0,2)
	found := false
	for _, move := range attacks {
		if move.Position.Row == 0 && move.Position.Col == 2 && move.Type == MoveAttack {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find attack move at (0,2)")
	}
}

func TestIsAlive(t *testing.T) {
	board := NewBoard(5)
	board.BasePos[1] = Position{Row: 0, Col: 0}
	board.BasePos[1] = Position{Row: 4, Col: 4}

	// Player 0 has cells
	board.SetCell(Position{Row: 0, Col: 0}, protocol.CellPlayer1)
	board.SetCell(Position{Row: 0, Col: 1}, protocol.CellPlayer1)

	// Player 1 has cells
	board.SetCell(Position{Row: 4, Col: 4}, protocol.CellPlayer2)

	if !board.IsAlive(1) {
		t.Error("Player 0 should be alive")
	}
	if !board.IsAlive(1) {
		t.Error("Player 1 should be alive")
	}

	// Remove player 1's cells
	board.SetCell(Position{Row: 0, Col: 0}, protocol.CellEmpty)
	board.SetCell(Position{Row: 0, Col: 1}, protocol.CellEmpty)

	if board.IsAlive(1) {
		t.Error("Player 0 should be dead")
	}
}

func TestGetNeutralPositions(t *testing.T) {
	board := NewBoard(5)
	board.BasePos[1] = Position{Row: 0, Col: 0}

	board.SetCell(Position{Row: 0, Col: 0}, protocol.CellPlayer1)
	board.SetCell(Position{Row: 0, Col: 1}, protocol.CellPlayer1)
	board.SetCell(Position{Row: 0, Col: 2}, protocol.CellPlayer1)

	neutrals := board.GetNeutralPositions(1)

	// Should find 3 positions
	if len(neutrals) != 3 {
		t.Errorf("Expected 3 neutral positions, got %d", len(neutrals))
	}
}

func TestValidMove(t *testing.T) {
	board := NewBoard(5)
	board.BasePos[1] = Position{Row: 0, Col: 0}
	board.BasePos[1] = Position{Row: 4, Col: 4}

	// Set up player 1's territory
	board.SetCell(Position{Row: 0, Col: 0}, protocol.CellPlayer1)
	board.SetCell(Position{Row: 0, Col: 1}, protocol.CellPlayer1)

	// Set up player 2's territory (for attack test)
	board.SetCell(Position{Row: 0, Col: 3}, protocol.CellPlayer2)

	tests := []struct {
		name     string
		move     Move
		playerID int
		valid    bool
	}{
		{
			name:     "Valid grow move",
			move:     Move{Position: Position{Row: 0, Col: 2}, Type: MoveGrow, FromCell: Position{Row: 0, Col: 1}},
			playerID: 1, // Player 1 has cells at row 0
			valid:    true,
		},
		{
			name:     "Invalid - from disconnected cell",
			move:     Move{Position: Position{Row: 2, Col: 2}, Type: MoveGrow, FromCell: Position{Row: 2, Col: 2}},
			playerID: 1,
			valid:    false,
		},
		{
			name:     "Invalid - not adjacent",
			move:     Move{Position: Position{Row: 0, Col: 3}, Type: MoveGrow, FromCell: Position{Row: 0, Col: 0}},
			playerID: 1,
			valid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ValidMove(board, tt.playerID, tt.move) != tt.valid {
				t.Errorf("ValidMove() = %v, want %v", !tt.valid, tt.valid)
			}
		})
	}
}
