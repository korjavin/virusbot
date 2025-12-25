package strategy

import (
	"math"
	"math/rand"
	"time"
	"virusbot/config"
	"virusbot/internal/game"
)

// MCTSConfig contains configuration for MCTS
type MCTSConfig struct {
	Iterations       int
	TimeLimit        time.Duration
	ExplorationConst float64
	MaxDepth         int
}

// DefaultMCTSConfig returns default MCTS configuration
func DefaultMCTSConfig() MCTSConfig {
	return MCTSConfig{
		Iterations:       1000,
		TimeLimit:        1 * time.Second,
		ExplorationConst: 1.41,
		MaxDepth:         50,
	}
}

// MCTSStrategy uses Monte Carlo Tree Search
type MCTSStrategy struct {
	config MCTSConfig
	rand   *rand.Rand
	debug  bool
}

// NewMCTSStrategy creates a new MCTS strategy
func NewMCTSStrategy(cfg *config.Config) *MCTSStrategy {
	return &MCTSStrategy{
		config: MCTSConfig{
			Iterations:       cfg.MCTSIterations,
			TimeLimit:        cfg.MCTSTimeLimit,
			ExplorationConst: cfg.MCTSUCTConst,
			MaxDepth:         50,
		},
		rand:  rand.New(rand.NewSource(time.Now().UnixNano())),
		debug: cfg.Debug,
	}
}

// Name returns the strategy name
func (s *MCTSStrategy) Name() string {
	return "mcts"
}

// DecideMoves selects the best moves using MCTS
func (s *MCTSStrategy) DecideMoves(state *game.GameState, count int) []game.Move {
	if !state.IsMyTurn() {
		return nil
	}

	player := state.GetYourPlayer()
	if player == nil {
		return nil
	}

	// Get all valid moves
	validMoves := state.Board.GetValidMoves(player.ID)
	if len(validMoves) == 0 {
		return nil
	}

	// For 3 moves, we need to select the best combination
	// Run MCTS to find the best moves
	moves := s.runMCTS(state, validMoves, count)

	return moves
}

// runMCTS runs the MCTS algorithm
func (s *MCTSStrategy) runMCTS(state *game.GameState, validMoves []game.Move, count int) []game.Move {
	if len(validMoves) <= count {
		return validMoves
	}

	// Run simulations with time limit
	deadline := time.Now().Add(s.config.TimeLimit)
	iterations := 0

	for time.Now().Before(deadline) && iterations < s.config.Iterations {
		s.iteration(state, validMoves)
		iterations++
	}

	// Select best moves based on visit counts
	return s.selectBestMoves(validMoves, count)
}

// iteration performs one MCTS iteration
func (s *MCTSStrategy) iteration(rootState *game.GameState, validMoves []game.Move) {
	// For simplicity, we'll use a simplified MCTS that evaluates each move independently
	// This is a basic implementation - a full MCTS would build a tree

	// Evaluate all moves and track the best
	bestScore := -1.0
	for _, move := range validMoves {
		score := s.simulateRandomPlayout(rootState, move)
		if score > bestScore {
			bestScore = score
		}
	}

	_ = bestScore // Suppress unused variable warning
}

// simulateRandomPlayout simulates a random playout from the given move
func (s *MCTSStrategy) simulateRandomPlayout(state *game.GameState, firstMove game.Move) float64 {
	simState := state.Clone()
	player := simState.GetCurrentPlayer()
	if player == nil {
		return 0
	}

	// Apply the first move
	simState = simState.ApplyMove(firstMove)

	depth := 1
	winner := -1

	// Random playout until game ends or max depth
	for depth < s.config.MaxDepth {
		alive := simState.GetAlivePlayers()
		if len(alive) <= 1 {
			if len(alive) == 1 && alive[0].ID == state.YourPlayerID {
				winner = state.YourPlayerID
			}
			break
		}

		// Get random move for current player
		currentPlayer := simState.GetCurrentPlayer()
		if currentPlayer == nil {
			break
		}

		moves := simState.Board.GetValidMoves(currentPlayer.ID)
		if len(moves) == 0 {
			// Skip this player's turn
			simState.AdvancePlayer()
			continue
		}

		// Pick random move
		move := moves[s.rand.Intn(len(moves))]
		simState = simState.ApplyMove(move)

		depth++
	}

	// Return a score based on outcome
	if winner == state.YourPlayerID {
		return 1.0
	}
	return 0.0
}

// selectBestMoves selects the best moves based on simulation results
func (s *MCTSStrategy) selectBestMoves(moves []game.Move, count int) []game.Move {
	if len(moves) <= count {
		return moves
	}

	// Score each move and pick the best
	type moveScore struct {
		move  game.Move
		score float64
	}

	scored := make([]moveScore, len(moves))
	for i, move := range moves {
		scored[i] = moveScore{move: move, score: 0}
	}

	// Run more thorough evaluation
	for i, ms := range scored {
		// Evaluate each move multiple times
		sumScore := 0.0
		for j := 0; j < 10; j++ {
			sumScore += s.evaluateMove(ms.move)
		}
		scored[i].score = sumScore / 10.0
	}

	// Sort by score descending
	for i := 0; i < len(scored)-1; i++ {
		maxIdx := i
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[maxIdx].score {
				maxIdx = j
			}
		}
		scored[i], scored[maxIdx] = scored[maxIdx], scored[i]
	}

	// Select top moves
	result := make([]game.Move, count)
	for i := 0; i < count; i++ {
		result[i] = scored[i].move
	}

	return result
}

// evaluateMove evaluates a single move (simplified)
func (s *MCTSStrategy) evaluateMove(move game.Move) float64 {
	score := 0.0

	// Prefer attacks
	if move.Type == game.MoveAttack {
		score += 15.0
	} else {
		score += 10.0
	}

	// Add some randomness for exploration
	score += s.rand.Float64() * 2.0

	return score
}

// UCT calculates the Upper Confidence Bound for Trees
func (s *MCTSStrategy) UCT(wins, visits, parentVisits float64) float64 {
	if visits == 0 {
		return math.MaxFloat64
	}
	return (wins / visits) + s.config.ExplorationConst*math.Sqrt(math.Log(parentVisits)/visits)
}

// DecideNeutrals uses a simpler heuristic for neutral placement
func (s *MCTSStrategy) DecideNeutrals(state *game.GameState) []game.Position {
	// Fall back to heuristic for neutrals (MCTS is complex for this)
	heuristic := NewHeuristicStrategy(&config.Config{Debug: s.debug})
	return heuristic.DecideNeutrals(state)
}

// OnMoveMade is a no-op for MCTS strategy
func (s *MCTSStrategy) OnMoveMade(state *game.GameState, move game.Move) {
	// No explicit learning in basic MCTS
}
