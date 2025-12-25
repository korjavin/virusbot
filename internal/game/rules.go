package game

import (
	"virusbot/internal/protocol"
)

// MoveType represents the type of move
type MoveType int

const (
	MoveGrow MoveType = iota
	MoveAttack
)

// Move represents a potential move
type Move struct {
	Position Position
	Type     MoveType
	FromCell Position // The cell we're expanding from
}

// ValidMove checks if a move is legal for a player
func ValidMove(board *Board, playerID int, move Move) bool {
	// Check if the position is within the board
	if !board.IsValid(move.Position) {
		return false
	}

	// Check if the move originates from a cell connected to base
	if !board.IsConnectedToBase(playerID, move.FromCell) {
		return false
	}

	// Check the move type
	switch move.Type {
	case MoveGrow:
		// Must be growing into an empty cell
		return board.IsEmpty(move.Position) && board.IsAdjacent(move.FromCell, move.Position)
	case MoveAttack:
		// Must be attacking an opponent's cell
		return board.IsOpponent(move.Position, playerID) && board.IsAdjacent(move.FromCell, move.Position)
	}

	return false
}

// IsAdjacent checks if two positions are adjacent (8-directional: includes diagonals)
func (b *Board) IsAdjacent(pos1, pos2 Position) bool {
	dr := abs(pos1.Row - pos2.Row)
	dc := abs(pos1.Col - pos2.Col)
	// Adjacent if distance is at most 1 in both directions (allows diagonals)
	return dr <= 1 && dc <= 1 && (dr != 0 || dc != 0)
}

// IsConnectedToBase checks if a cell is connected to the player's base
// This is the critical rule: you can only expand from cells connected to base
func (b *Board) IsConnectedToBase(playerID int, pos Position) bool {
	basePos, exists := b.BasePos[playerID]
	if !exists {
		return false
	}

	// Use BFS to check if pos is connected to base through player's cells
	visited := make(map[Position]bool)
	queue := []Position{basePos}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.Row == pos.Row && current.Col == pos.Col {
			return true
		}

		visited[current] = true

		// Check all player's cells adjacent to current
		for _, neighbor := range b.GetNeighbors(current) {
			if visited[neighbor] {
				continue
			}
			// Can only traverse through player's own cells
			if b.IsOwnedBy(neighbor, playerID) {
				queue = append(queue, neighbor)
			}
		}
	}

	return false
}

// GetReachableCells returns all cells that are connected to the base
func (b *Board) GetReachableCells(playerID int) []Position {
	basePos, exists := b.BasePos[playerID]
	if !exists {
		return nil
	}

	// Check if base is still owned by player (could have been captured)
	if !b.IsOwnedBy(basePos, playerID) {
		// Base was captured - find any remaining cells owned by this player
		// and use the first one as a new starting point for BFS
		playerCells := b.GetPlayerCells(playerID)
		if len(playerCells) == 0 {
			return nil // Player has no cells left
		}
		basePos = playerCells[0] // Use first remaining cell as new "base"
	}

	reachable := make([]Position, 0)
	visited := make(map[Position]bool)
	queue := []Position{basePos}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true
		reachable = append(reachable, current)

		// Check all player's cells adjacent to current
		for _, neighbor := range b.GetNeighbors(current) {
			if !visited[neighbor] && b.IsOwnedBy(neighbor, playerID) {
				queue = append(queue, neighbor)
			}
		}
	}

	return reachable
}

// GetValidMoves returns all valid moves for a player
func (b *Board) GetValidMoves(playerID int) []Move {
	moves := make([]Move, 0)

	// Get all cells connected to base
	reachableCells := b.GetReachableCells(playerID)

	// Special case: if player has no cells yet (first move), they can place anywhere
	if len(reachableCells) == 0 {
		// First move: can place on any empty cell
		for row := 0; row < b.Size; row++ {
			for col := 0; col < b.Size; col++ {
				pos := Position{Row: row, Col: col}
				if b.IsEmpty(pos) {
					moves = append(moves, Move{
						Position: pos,
						Type:     MoveGrow,
						FromCell: pos, // First move, no "from" cell
					})
				}
			}
		}
		return moves
	}

	for _, fromCell := range reachableCells {
		// Check all neighbors for potential moves
		for _, neighbor := range b.GetNeighbors(fromCell) {
			// Skip if this is one of our own cells
			if b.IsOwnedBy(neighbor, playerID) {
				continue
			}

			// Check for grow move (into empty cell)
			if b.IsEmpty(neighbor) {
				moves = append(moves, Move{
					Position: neighbor,
					Type:     MoveGrow,
					FromCell: fromCell,
				})
			}

			// Check for attack move (into opponent cell)
			if b.IsOpponent(neighbor, playerID) {
				moves = append(moves, Move{
					Position: neighbor,
					Type:     MoveAttack,
					FromCell: fromCell,
				})
			}
		}
	}

	return moves
}

// GetAttackMoves returns only attack moves
func (b *Board) GetAttackMoves(playerID int) []Move {
	moves := b.GetValidMoves(playerID)
	attacks := make([]Move, 0)
	for _, move := range moves {
		if move.Type == MoveAttack {
			attacks = append(attacks, move)
		}
	}
	return attacks
}

// GetGrowMoves returns only grow moves
func (b *Board) GetGrowMoves(playerID int) []Move {
	moves := b.GetValidMoves(playerID)
	grows := make([]Move, 0)
	for _, move := range moves {
		if move.Type == MoveGrow {
			grows = append(grows, move)
		}
	}
	return grows
}

// CanPlaceNeutrals checks if the player can place neutral cells
func (b *Board) CanPlaceNeutrals(playerID int) bool {
	// Count non-fortified cells (cells that are owned but not the base)
	// In this game, we can place neutrals on any of our cells
	// The rule says "at least two non-fortified cells"

	// For now, we check if player has at least 2 cells total
	cells := b.GetPlayerCells(playerID)
	return len(cells) >= 2
}

// GetNeutralPositions returns valid positions for neutral placement
func (b *Board) GetNeutralPositions(playerID int) []Position {
	// Can place on any of our own non-fortified cells
	// In this implementation, all player cells are valid for neutrals
	cells := b.GetPlayerCells(playerID)
	neutrals := make([]Position, 0)
	for _, cell := range cells {
		if b.GetCell(cell) != protocol.CellNeutral {
			neutrals = append(neutrals, cell)
		}
	}
	return neutrals
}

// IsAlive checks if a player is still in the game
func (b *Board) IsAlive(playerID int) bool {
	cells := b.GetPlayerCells(playerID)
	return len(cells) > 0
}

// GetAlivePlayers returns the IDs of all alive players
func (b *Board) GetAlivePlayers(players []*Player) []int {
	alive := make([]int, 0)
	for _, p := range players {
		if b.IsAlive(p.ID) {
			alive = append(alive, p.ID)
		}
	}
	return alive
}

// GetOpponents returns the IDs of all opponent players
func (b *Board) GetOpponents(playerID int, allPlayers []*Player) []int {
	opponents := make([]int, 0)
	for _, p := range allPlayers {
		if p.ID != playerID && b.IsAlive(p.ID) {
			opponents = append(opponents, p.ID)
		}
	}
	return opponents
}

// Helper function
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
