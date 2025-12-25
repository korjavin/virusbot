package strategy

import (
	"testing"

	"virusbot/config"
	"virusbot/internal/game"
	"virusbot/internal/protocol"
)

func createTestBoard() *game.Board {
	board := game.NewBoard(10)
	board.BasePos[0] = game.Position{Row: 0, Col: 0}
	board.BasePos[1] = game.Position{Row: 9, Col: 9}

	// Set up player 1's base
	board.SetCell(game.Position{Row: 0, Col: 0}, protocol.CellPlayer1)
	board.SetCell(game.Position{Row: 0, Col: 1}, protocol.CellPlayer1)

	// Set up player 2's base
	board.SetCell(game.Position{Row: 9, Col: 9}, protocol.CellPlayer2)
	board.SetCell(game.Position{Row: 9, Col: 8}, protocol.CellPlayer2)

	return board
}

func TestHeuristicStrategyNeverReturnsInvalidMoves(t *testing.T) {
	cfg := &config.Config{Debug: false}
	strategy := NewHeuristicStrategy(cfg)

	// Create game state
	board := createTestBoard()
	state := &game.GameState{
		Board:         board,
		Players:       nil,
		CurrentPlayer: 2, // Player 2's turn
		YourPlayerID:  2,
	}

	// Get moves
	moves := strategy.DecideMoves(state, 3)

	// Verify no moves target occupied cells
	for _, move := range moves {
		cell := state.Board.GetCell(move.Position)

		// Cell should not be player 2's own cell (can't move to own cell)
		if cell == protocol.CellPlayer2 {
			t.Errorf("Strategy returned move to own cell at %v - this should never happen", move.Position)
		}

		// For grow moves, target must be empty
		if move.Type == game.MoveGrow && !state.Board.IsEmpty(move.Position) {
			t.Errorf("Strategy returned grow move to occupied cell at %v", move.Position)
		}

		// For attack moves, target must be opponent's cell
		if move.Type == game.MoveAttack && !state.Board.IsOpponent(move.Position, state.YourPlayerID) {
			t.Errorf("Strategy returned attack move to non-opponent cell at %v", move.Position)
		}
	}
}

func TestMCTSStrategyNeverReturnsInvalidMoves(t *testing.T) {
	cfg := &config.Config{Debug: false, MCTSIterations: 100}
	strategy := NewMCTSStrategy(cfg)

	// Create game state
	board := createTestBoard()
	state := &game.GameState{
		Board:         board,
		Players:       nil,
		CurrentPlayer: 2, // Player 2's turn
		YourPlayerID:  2,
	}

	// Get moves
	moves := strategy.DecideMoves(state, 3)

	// Verify no moves target occupied cells
	for _, move := range moves {
		cell := state.Board.GetCell(move.Position)

		// Cell should not be player 2's own cell (can't move to own cell)
		if cell == protocol.CellPlayer2 {
			t.Errorf("Strategy returned move to own cell at %v - this should never happen", move.Position)
		}

		// For grow moves, target must be empty
		if move.Type == game.MoveGrow && !state.Board.IsEmpty(move.Position) {
			t.Errorf("Strategy returned grow move to occupied cell at %v", move.Position)
		}

		// For attack moves, target must be opponent's cell
		if move.Type == game.MoveAttack && !state.Board.IsOpponent(move.Position, state.YourPlayerID) {
			t.Errorf("Strategy returned attack move to non-opponent cell at %v", move.Position)
		}
	}
}

func TestStrategyWithCompletelyOccupiedBoard(t *testing.T) {
	cfg := &config.Config{Debug: false}
	heuristic := NewHeuristicStrategy(cfg)
	mcts := &MCTSStrategy{config: DefaultMCTSConfig(), debug: false}

	// Create a board where player 2 has no valid moves
	board := game.NewBoard(5)
	board.BasePos[0] = game.Position{Row: 0, Col: 0}
	board.BasePos[1] = game.Position{Row: 4, Col: 4}

	// Player 1 occupies all cells around player 2's base
	board.SetCell(game.Position{Row: 4, Col: 4}, protocol.CellPlayer2) // Player 2's base
	for r := 3; r < 5; r++ {
		for c := 3; c < 5; c++ {
			if r == 4 && c == 4 {
				continue
			}
			board.SetCell(game.Position{Row: r, Col: c}, protocol.CellPlayer1)
		}
	}

	state := &game.GameState{
		Board:         board,
		Players:       nil,
		CurrentPlayer: 2,
		YourPlayerID:  2,
	}

	// Both strategies should return empty when no valid moves
	heuristicMoves := heuristic.DecideMoves(state, 3)
	if len(heuristicMoves) != 0 {
		t.Errorf("Heuristic strategy returned %d moves when no valid moves exist", len(heuristicMoves))
	}

	mctsMoves := mcts.DecideMoves(state, 3)
	if len(mctsMoves) != 0 {
		t.Errorf("MCTS strategy returned %d moves when no valid moves exist", len(mctsMoves))
	}
}

func TestStrategyFiltersOccupiedCells(t *testing.T) {
	cfg := &config.Config{Debug: false}
	strategy := NewHeuristicStrategy(cfg)

	board := game.NewBoard(5)
	board.BasePos[0] = game.Position{Row: 0, Col: 0}
	board.BasePos[1] = game.Position{Row: 4, Col: 4}

	// Set up player 2's territory
	board.SetCell(game.Position{Row: 4, Col: 4}, protocol.CellPlayer2)
	board.SetCell(game.Position{Row: 4, Col: 3}, protocol.CellPlayer2)
	board.SetCell(game.Position{Row: 3, Col: 4}, protocol.CellPlayer2)

	// Set up opponent territory that blocks expansion
	board.SetCell(game.Position{Row: 3, Col: 3}, protocol.CellPlayer1)

	state := &game.GameState{
		Board:         board,
		Players:       nil,
		CurrentPlayer: 2,
		YourPlayerID:  2,
	}

	moves := strategy.DecideMoves(state, 3)

	// Verify all moves are valid
	for _, move := range moves {
		// Should never suggest moving to player 2's own cell
		if state.Board.GetCell(move.Position) == protocol.CellPlayer2 {
			t.Errorf("Suggested move to own cell at %v", move.Position)
		}
	}
}
