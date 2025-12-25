package strategy

import (
	"virusbot/internal/game"
)

// Strategy defines the interface for game playing strategies
type Strategy interface {
	// Name returns the name of the strategy
	Name() string

	// DecideMoves decides which moves to make
	DecideMoves(state *game.GameState, count int) []game.Move

	// DecideNeutrals decides where to place neutral cells
	DecideNeutrals(state *game.GameState) []game.Position

	// OnMoveMade is called when a move is made (for learning strategies)
	OnMoveMade(state *game.GameState, move game.Move)
}
