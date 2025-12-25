package game

import (
	"testing"

	"virusbot/internal/protocol"
)

func TestNewBoard(t *testing.T) {
	board := NewBoard(10)

	if board.Size != 10 {
		t.Errorf("Expected size 10, got %d", board.Size)
	}

	if len(board.Cells) != 10 {
		t.Errorf("Expected 10 rows, got %d", len(board.Cells))
	}

	for i := 0; i < 10; i++ {
		if len(board.Cells[i]) != 10 {
			t.Errorf("Expected 10 cols in row %d, got %d", i, len(board.Cells[i]))
		}
		for j := 0; j < 10; j++ {
			if board.Cells[i][j] != protocol.CellEmpty {
				t.Errorf("Expected empty cell at (%d, %d)", i, j)
			}
		}
	}
}

func TestBoardCellOperations(t *testing.T) {
	board := NewBoard(5)

	// Test SetCell and GetCell
	pos := Position{Row: 2, Col: 3}
	board.SetCell(pos, protocol.CellPlayer1)

	if board.GetCell(pos) != protocol.CellPlayer1 {
		t.Errorf("Expected CellPlayer1, got %v", board.GetCell(pos))
	}

	// Test IsEmpty
	if board.IsEmpty(pos) {
		t.Error("Expected cell to not be empty")
	}

	// Test IsOwnedBy
	if !board.IsOwnedBy(pos, 0) { // Player1 has ID 0
		t.Error("Expected cell to be owned by player 0")
	}

	if board.IsOwnedBy(pos, 1) {
		t.Error("Expected cell to not be owned by player 1")
	}
}

func TestBoardIsValid(t *testing.T) {
	board := NewBoard(5)

	tests := []struct {
		pos      Position
		expected bool
	}{
		{Position{0, 0}, true},
		{Position{4, 4}, true},
		{Position{2, 2}, true},
		{Position{-1, 0}, false},
		{Position{0, -1}, false},
		{Position{5, 0}, false},
		{Position{0, 5}, false},
	}

	for _, tt := range tests {
		if board.IsValid(tt.pos) != tt.expected {
			t.Errorf("IsValid(%v) = %v, want %v", tt.pos, board.IsValid(tt.pos), tt.expected)
		}
	}
}

func TestBoardNeighbors(t *testing.T) {
	board := NewBoard(5)
	pos := Position{Row: 2, Col: 2}

	neighbors := board.GetNeighbors(pos)

	if len(neighbors) != 4 {
		t.Errorf("Expected 4 neighbors, got %d", len(neighbors))
	}

	// Check all directions
	expected := []Position{
		{1, 2}, {3, 2}, {2, 1}, {2, 3},
	}

	for _, exp := range expected {
		found := false
		for _, n := range neighbors {
			if n.Row == exp.Row && n.Col == exp.Col {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected neighbor %v not found", exp)
		}
	}

	// Test corner
	corner := Position{0, 0}
	cornerNeighbors := board.GetNeighbors(corner)
	if len(cornerNeighbors) != 2 {
		t.Errorf("Expected 2 neighbors for corner, got %d", len(cornerNeighbors))
	}
}

func TestBoardClone(t *testing.T) {
	board := NewBoard(5)
	board.SetCell(Position{0, 0}, protocol.CellPlayer1)
	board.BasePos[0] = Position{0, 0}

	cloned := board.Clone()

	// Modify original
	board.SetCell(Position{0, 0}, protocol.CellPlayer2)

	// Cloned should be unchanged
	if cloned.GetCell(Position{0, 0}) != protocol.CellPlayer1 {
		t.Error("Cloned board was affected by original modification")
	}
}

func TestBoardIsEdgePosition(t *testing.T) {
	board := NewBoard(5)

	tests := []struct {
		pos      Position
		expected bool
	}{
		{Position{0, 2}, true},
		{Position{4, 2}, true},
		{Position{2, 0}, true},
		{Position{2, 4}, true},
		{Position{1, 1}, false},
		{Position{2, 2}, false},
	}

	for _, tt := range tests {
		if board.IsEdgePosition(tt.pos) != tt.expected {
			t.Errorf("IsEdgePosition(%v) = %v, want %v", tt.pos, board.IsEdgePosition(tt.pos), tt.expected)
		}
	}
}

func TestBoardIsCornerPosition(t *testing.T) {
	board := NewBoard(5)

	tests := []struct {
		pos      Position
		expected bool
	}{
		{Position{0, 0}, true},
		{Position{0, 4}, true},
		{Position{4, 0}, true},
		{Position{4, 4}, true},
		{Position{0, 1}, false},
		{Position{1, 0}, false},
		{Position{2, 2}, false},
	}

	for _, tt := range tests {
		if board.IsCornerPosition(tt.pos) != tt.expected {
			t.Errorf("IsCornerPosition(%v) = %v, want %v", tt.pos, board.IsCornerPosition(tt.pos), tt.expected)
		}
	}
}
