package strategy

import (
	"virusbot/config"
	"virusbot/internal/game"
)

// EvaluationFactors contains weights for different scoring factors
type EvaluationFactors struct {
	TerritoryGain      float64 // +10 per cell captured
	StrategicPosition  float64 // +5 for edge, +8 for corner
	ThreatRemoval      float64 // +15 for attacking
	Connectivity       float64 // +3 for reconnecting cut-off groups
	ExpansionPotential float64 // +4 for cells with multiple empty neighbors
	DefensiveValue     float64 // +2 for cells adjacent to own territory
}

// DefaultFactors returns the default evaluation factors
func DefaultFactors() EvaluationFactors {
	return EvaluationFactors{
		TerritoryGain:      1.0,
		StrategicPosition:  0.5,
		ThreatRemoval:      1.5,
		Connectivity:       0.3,
		ExpansionPotential: 0.4,
		DefensiveValue:     0.2,
	}
}

// HeuristicStrategy uses a multi-factor scoring system
type HeuristicStrategy struct {
	factors EvaluationFactors
	debug   bool
}

// NewHeuristicStrategy creates a new heuristic strategy
func NewHeuristicStrategy(cfg *config.Config) *HeuristicStrategy {
	return &HeuristicStrategy{
		factors: EvaluationFactors{
			TerritoryGain:      cfg.WeightTerritory,
			StrategicPosition:  cfg.WeightStrategic,
			ThreatRemoval:      cfg.WeightThreat,
			Connectivity:       cfg.WeightConnectivity,
			ExpansionPotential: cfg.WeightExpansion,
			DefensiveValue:     cfg.WeightDefensive,
		},
		debug: cfg.Debug,
	}
}

// Name returns the strategy name
func (s *HeuristicStrategy) Name() string {
	return "heuristic"
}

// DecideMoves selects the best moves for the current turn
func (s *HeuristicStrategy) DecideMoves(state *game.GameState, count int) []game.Move {
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

	// Score each move
	scoredMoves := s.scoreMoves(validMoves, state)

	// Select top moves with diversity
	selected := s.selectDiverseMoves(scoredMoves, count)

	return selected
}

// scoreMoves assigns a score to each move
func (s *HeuristicStrategy) scoreMoves(moves []game.Move, state *game.GameState) []scoredMove {
	player := state.GetYourPlayer()
	if player == nil {
		return nil
	}

	scored := make([]scoredMove, 0, len(moves))
	for _, move := range moves {
		score := s.evaluateMove(move, state, player.ID)
		scored = append(scored, scoredMove{
			move:  move,
			score: score,
		})
	}

	return scored
}

// evaluateMove evaluates a single move
func (s *HeuristicStrategy) evaluateMove(move game.Move, state *game.GameState, playerID int) float64 {
	board := state.Board
	score := 0.0

	// 1. Territory Gain
	// +10 for each cell captured (both grow and attack)
	score += 10.0 * s.factors.TerritoryGain

	// 2. Strategic Position
	if board.IsCornerPosition(move.Position) {
		score += 8.0 * s.factors.StrategicPosition
	} else if board.IsEdgePosition(move.Position) {
		score += 5.0 * s.factors.StrategicPosition
	}

	// 3. Threat Removal
	if move.Type == game.MoveAttack {
		score += 15.0 * s.factors.ThreatRemoval
	}

	// 4. Connectivity
	// Check if this move helps reconnect cut-off cells
	if s.improvesConnectivity(move, state, playerID) {
		score += 3.0 * s.factors.Connectivity
	}

	// 5. Expansion Potential
	// How many new cells can we reach from this position?
	emptyNeighbors := len(board.GetEmptyNeighbors(move.Position))
	score += float64(emptyNeighbors) * 4.0 * s.factors.ExpansionPotential

	// 6. Defensive Value
	// Check if this move protects our base or creates a barrier
	if s.hasDefensiveValue(move, state, playerID) {
		score += 2.0 * s.factors.DefensiveValue
	}

	return score
}

// improvesConnectivity checks if a move helps reconnect cells
func (s *HeuristicStrategy) improvesConnectivity(move game.Move, state *game.GameState, playerID int) bool {
	// If the move position is already connected to base, no improvement
	if state.Board.IsConnectedToBase(playerID, move.Position) {
		return false
	}

	// Check if the move connects to the main territory
	connectedCells := state.Board.GetReachableCells(playerID)
	for _, cell := range connectedCells {
		if state.Board.IsAdjacent(cell, move.Position) {
			return true
		}
	}

	return false
}

// hasDefensiveValue checks if a move has defensive value
func (s *HeuristicStrategy) hasDefensiveValue(move game.Move, state *game.GameState, playerID int) bool {
	player := state.GetYourPlayer()
	if player == nil {
		return false
	}

	// Check if near base (defending base)
	if state.Board.IsAdjacent(move.Position, player.BasePos) {
		return true
	}

	// Check if it blocks an opponent's path
	opponents := state.GetOpponents()
	for _, opp := range opponents {
		oppBaseAdjacent := state.Board.GetNeighbors(opp.BasePos)
		for _, adj := range oppBaseAdjacent {
			if adj.Row == move.Position.Row && adj.Col == move.Position.Col {
				return true
			}
		}
	}

	return false
}

// selectDiverseMoves selects moves that are diverse (not in the same cluster)
func (s *HeuristicStrategy) selectDiverseMoves(scored []scoredMove, count int) []game.Move {
	if len(scored) <= count {
		result := make([]game.Move, len(scored))
		for i, sm := range scored {
			result[i] = sm.move
		}
		return result
	}

	// Sort by score descending
	// (Using simple selection sort for now)
	for i := 0; i < len(scored)-1; i++ {
		maxIdx := i
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[maxIdx].score {
				maxIdx = j
			}
		}
		scored[i], scored[maxIdx] = scored[maxIdx], scored[i]
	}

	// Select top moves, preferring diversity
	selected := make([]game.Move, 0, count)
	selectedPositions := make(map[game.Position]bool)

	for _, sm := range scored {
		if len(selected) >= count {
			break
		}

		// Simple diversity: don't select moves from the exact same "from" cell if possible
		if !selectedPositions[sm.move.FromCell] || len(selectedPositions) >= count-1 {
			selected = append(selected, sm.move)
			selectedPositions[sm.move.FromCell] = true
		}
	}

	return selected
}

// scoredMove is a move with its score
type scoredMove struct {
	move  game.Move
	score float64
}

// DecideNeutrals decides where to place neutral cells
func (s *HeuristicStrategy) DecideNeutrals(state *game.GameState) []game.Position {
	player := state.GetYourPlayer()
	if player == nil || player.HasUsedNeutrals {
		return nil
	}

	// Get valid positions for neutrals
	validPositions := state.Board.GetNeutralPositions(player.ID)
	if len(validPositions) < 2 {
		return nil
	}

	// Score each position
	scored := make([]scoredPosition, 0, len(validPositions))
	for _, pos := range validPositions {
		score := s.evaluateNeutralPosition(pos, state, player.ID)
		scored = append(scored, scoredPosition{
			position: pos,
			score:    score,
		})
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

	// Return top 2
	result := make([]game.Position, 0, 2)
	for i := 0; i < 2 && i < len(scored); i++ {
		result = append(result, scored[i].position)
	}

	return result
}

// evaluateNeutralPosition scores a position for neutral placement
func (s *HeuristicStrategy) evaluateNeutralPosition(pos game.Position, state *game.GameState, playerID int) float64 {
	score := 0.0

	// Prefer blocking opponent paths to our base
	opponents := state.GetOpponents()
	for _, opp := range opponents {
		// Check if this position blocks the opponent from reaching our base
		if s.blocksPathToBase(pos, state, opp.ID, playerID) {
			score += 20.0
		}
	}

	// Prefer creating chokepoints
	if s.createsChokepoint(pos, state) {
		score += 15.0
	}

	// Prefer corners for blocking
	if state.Board.IsCornerPosition(pos) {
		score += 10.0
	}

	// Prefer positions adjacent to many empty cells (blocking expansion)
	emptyNeighbors := len(state.Board.GetEmptyNeighbors(pos))
	score += float64(emptyNeighbors) * 3.0

	// Avoid placing near our base (don't block ourselves)
	player := state.GetYourPlayer()
	if player != nil && state.Board.IsAdjacent(pos, player.BasePos) {
		score -= 10.0
	}

	return score
}

// blocksPathToBase checks if placing a neutral blocks an opponent's path to our base
func (s *HeuristicStrategy) blocksPathToBase(pos game.Position, state *game.GameState, opponentID, ourID int) bool {
	// Simplified: check if position is adjacent to opponent's potential expansion area
	// near our base
	ourBase := state.Board.BasePos[ourID]
	baseNeighbors := state.Board.GetNeighbors(ourBase)

	for _, neighbor := range baseNeighbors {
		if neighbor.Row == pos.Row && neighbor.Col == pos.Col {
			return true
		}
	}

	return false
}

// createsChokepoint checks if a position creates a chokepoint
func (s *HeuristicStrategy) createsChokepoint(pos game.Position, state *game.GameState) bool {
	// A chokepoint is where we force opponents to go through a narrow passage
	// Simplified: check if surrounded by our cells or board edges
	neighbors := state.Board.GetNeighbors(pos)
	ourCells := 0
	for _, n := range neighbors {
		// Would be our cell after placement - this is a simplification
		if state.Board.IsEdgePosition(n) {
			ourCells++
		}
	}
	return ourCells >= 2
}

// OnMoveMade is a no-op for heuristic strategy
func (s *HeuristicStrategy) OnMoveMade(state *game.GameState, move game.Move) {
	// No learning in basic heuristic strategy
}

// scoredPosition is a position with its score for neutral placement
type scoredPosition struct {
	position game.Position
	score    float64
}
