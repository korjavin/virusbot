package game

import (
	"virusbot/internal/protocol"
)

// Position represents a cell on the board
type Position struct {
	Row, Col int
}

// Board represents the game board
type Board struct {
	Size    int
	Cells   [][]protocol.CellType
	BasePos map[int]Position // playerID -> base position
}

// NewBoard creates a new empty board
func NewBoard(size int) *Board {
	cells := make([][]protocol.CellType, size)
	for i := range cells {
		cells[i] = make([]protocol.CellType, size)
		for j := range cells[i] {
			cells[i][j] = protocol.CellEmpty
		}
	}

	return &Board{
		Size:    size,
		Cells:   cells,
		BasePos: make(map[int]Position),
	}
}

// NewBoardFromData creates a board from existing data
func NewBoardFromData(cells [][]protocol.CellType, basePos map[int]Position) *Board {
	size := len(cells)
	return &Board{
		Size:    size,
		Cells:   cells,
		BasePos: basePos,
	}
}

// GetCell returns the cell type at the given position
func (b *Board) GetCell(pos Position) protocol.CellType {
	if !b.IsValid(pos) {
		return protocol.CellEmpty
	}
	return b.Cells[pos.Row][pos.Col]
}

// SetCell sets the cell type at the given position
func (b *Board) SetCell(pos Position, cellType protocol.CellType) {
	if b.IsValid(pos) {
		b.Cells[pos.Row][pos.Col] = cellType
	}
}

// IsValid checks if a position is within the board
func (b *Board) IsValid(pos Position) bool {
	return pos.Row >= 0 && pos.Row < b.Size &&
		pos.Col >= 0 && pos.Col < b.Size
}

// IsEmpty checks if a cell is empty
func (b *Board) IsEmpty(pos Position) bool {
	return b.GetCell(pos) == protocol.CellEmpty
}

// IsOwnedBy checks if a cell is owned by a specific player
func (b *Board) IsOwnedBy(pos Position, playerID int) bool {
	cell := b.GetCell(pos)
	// Player IDs are 1-4, cell types are 1-4 (Player1-Player4)
	return int(cell) == playerID+1 && cell != protocol.CellNeutral
}

// IsNeutral checks if a cell is neutral
func (b *Board) IsNeutral(pos Position) bool {
	return b.GetCell(pos) == protocol.CellNeutral
}

// IsOpponent checks if a cell is owned by an opponent
func (b *Board) IsOpponent(pos Position, playerID int) bool {
	cell := b.GetCell(pos)
	if cell == protocol.CellEmpty || cell == protocol.CellNeutral {
		return false
	}
	// Player IDs are 1-4, cell types are 1-4
	return int(cell) != playerID+1
}

// GetNeighbors returns all adjacent positions (up, down, left, right)
func (b *Board) GetNeighbors(pos Position) []Position {
	neighbors := make([]Position, 0, 4)
	directions := []struct{ dr, dc int }{
		{-1, 0}, {1, 0}, {0, -1}, {0, 1},
	}

	for _, d := range directions {
		n := Position{Row: pos.Row + d.dr, Col: pos.Col + d.dc}
		if b.IsValid(n) {
			neighbors = append(neighbors, n)
		}
	}

	return neighbors
}

// GetAdjacentCells returns adjacent positions filtered by cell type
func (b *Board) GetAdjacentCells(pos Position, cellType protocol.CellType) []Position {
	neighbors := b.GetNeighbors(pos)
	result := make([]Position, 0)
	for _, n := range neighbors {
		if b.GetCell(n) == cellType {
			result = append(result, n)
		}
	}
	return result
}

// GetEmptyNeighbors returns all empty adjacent positions
func (b *Board) GetEmptyNeighbors(pos Position) []Position {
	return b.GetAdjacentCells(pos, protocol.CellEmpty)
}

// GetOpponentNeighbors returns all opponent-occupied adjacent positions
func (b *Board) GetOpponentNeighbors(pos Position, playerID int) []Position {
	neighbors := b.GetNeighbors(pos)
	result := make([]Position, 0)
	for _, n := range neighbors {
		if b.IsOpponent(n, playerID) {
			result = append(result, n)
		}
	}
	return result
}

// Clone creates a deep copy of the board
func (b *Board) Clone() *Board {
	newCells := make([][]protocol.CellType, b.Size)
	for i := range newCells {
		newCells[i] = make([]protocol.CellType, b.Size)
		copy(newCells[i], b.Cells[i])
	}

	newBasePos := make(map[int]Position)
	for k, v := range b.BasePos {
		newBasePos[k] = v
	}

	return &Board{
		Size:    b.Size,
		Cells:   newCells,
		BasePos: newBasePos,
	}
}

// ApplyMove applies a move to the board and returns a new board
func (b *Board) ApplyMove(pos Position, playerID int, isAttack bool) *Board {
	newBoard := b.Clone()
	cellType := protocol.CellType(playerID + 1)
	newBoard.SetCell(pos, cellType)
	return newBoard
}

// CountCells counts the number of cells owned by a player
func (b *Board) CountCells(playerID int) int {
	count := 0
	cellType := protocol.CellType(playerID + 1)
	for row := 0; row < b.Size; row++ {
		for col := 0; col < b.Size; col++ {
			if b.Cells[row][col] == cellType {
				count++
			}
		}
	}
	return count
}

// GetPlayerCells returns all positions owned by a player
func (b *Board) GetPlayerCells(playerID int) []Position {
	cellType := protocol.CellType(playerID + 1)
	cells := make([]Position, 0)
	for row := 0; row < b.Size; row++ {
		for col := 0; col < b.Size; col++ {
			if b.Cells[row][col] == cellType {
				cells = append(cells, Position{Row: row, Col: col})
			}
		}
	}
	return cells
}

// GetEmptyCells returns all empty positions
func (b *Board) GetEmptyCells() []Position {
	cells := make([]Position, 0)
	for row := 0; row < b.Size; row++ {
		for col := 0; col < b.Size; col++ {
			if b.Cells[row][col] == protocol.CellEmpty {
				cells = append(cells, Position{Row: row, Col: col})
			}
		}
	}
	return cells
}

// IsEdgePosition checks if a position is on the edge of the board
func (b *Board) IsEdgePosition(pos Position) bool {
	return pos.Row == 0 || pos.Row == b.Size-1 ||
		pos.Col == 0 || pos.Col == b.Size-1
}

// IsCornerPosition checks if a position is in a corner of the board
func (b *Board) IsCornerPosition(pos Position) bool {
	return (pos.Row == 0 || pos.Row == b.Size-1) &&
		(pos.Col == 0 || pos.Col == b.Size-1)
}
