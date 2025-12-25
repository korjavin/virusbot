package game

import (
	"virusbot/internal/protocol"
)

// GameState represents the complete state of a game
type GameState struct {
	Board         *Board
	Players       []*Player
	CurrentPlayer int
	YourPlayerID  int
}

// NewGameState creates a new game state from protocol data
func NewGameState(boardData [][]protocol.CellType, players []protocol.PlayerInfo, currentPlayer, yourPlayerID int) *GameState {
	// Build base positions from players
	basePos := make(map[int]Position)
	for _, p := range players {
		basePos[p.ID] = Position{
			Row: p.Position.Row,
			Col: p.Position.Col,
		}
	}

	board := NewBoardFromData(boardData, basePos)
	gamePlayers := PlayersFromInfo(players)

	return &GameState{
		Board:         board,
		Players:       gamePlayers,
		CurrentPlayer: currentPlayer,
		YourPlayerID:  yourPlayerID,
	}
}

// GetCurrentPlayer returns the current player
func (s *GameState) GetCurrentPlayer() *Player {
	for _, p := range s.Players {
		if p.ID == s.CurrentPlayer {
			return p
		}
	}
	return nil
}

// GetYourPlayer returns the player controlled by the bot
func (s *GameState) GetYourPlayer() *Player {
	for _, p := range s.Players {
		if p.ID == s.YourPlayerID {
			return p
		}
	}
	return nil
}

// GetPlayer returns a player by ID
func (s *GameState) GetPlayer(playerID int) *Player {
	for _, p := range s.Players {
		if p.ID == playerID {
			return p
		}
	}
	return nil
}

// IsMyTurn returns true if it's the bot's turn
func (s *GameState) IsMyTurn() bool {
	return s.CurrentPlayer == s.YourPlayerID
}

// GetOpponents returns all opponent players
func (s *GameState) GetOpponents() []*Player {
	opponents := make([]*Player, 0)
	for _, p := range s.Players {
		if p.ID != s.YourPlayerID && p.IsAlive {
			opponents = append(opponents, p)
		}
	}
	return opponents
}

// GetAlivePlayers returns all alive players
func (s *GameState) GetAlivePlayers() []*Player {
	alive := make([]*Player, 0)
	for _, p := range s.Players {
		if p.IsAlive {
			alive = append(alive, p)
		}
	}
	return alive
}

// Clone creates a deep copy of the game state
func (s *GameState) Clone() *GameState {
	newPlayers := make([]*Player, len(s.Players))
	for i, p := range s.Players {
		newPlayers[i] = p.Clone()
	}

	return &GameState{
		Board:         s.Board.Clone(),
		Players:       newPlayers,
		CurrentPlayer: s.CurrentPlayer,
		YourPlayerID:  s.YourPlayerID,
	}
}

// ApplyMove applies a move and returns a new game state
func (s *GameState) ApplyMove(move Move) *GameState {
	newState := s.Clone()
	player := newState.GetCurrentPlayer()
	if player == nil {
		return newState
	}

	// Apply the move to the board
	newState.Board.ApplyMove(move.Position, player.ID, move.Type == MoveAttack)

	// Update player's cell list
	if move.Type == MoveGrow {
		player.AddCell(move.Position)
	} else if move.Type == MoveAttack {
		// Remove the cell from the opponent and add to current player
		for _, opp := range newState.GetOpponents() {
			opp.RemoveCell(move.Position)
		}
		player.AddCell(move.Position)
	}

	// Advance to next player
	newState.AdvancePlayer()

	return newState
}

// AdvancePlayer moves to the next alive player
func (s *GameState) AdvancePlayer() {
	alive := s.GetAlivePlayers()
	if len(alive) == 0 {
		return
	}

	// Find current player's index
	currentIdx := -1
	for i, p := range alive {
		if p.ID == s.CurrentPlayer {
			currentIdx = i
			break
		}
	}

	// Move to next player
	nextIdx := (currentIdx + 1) % len(alive)
	s.CurrentPlayer = alive[nextIdx].ID
}

// ApplyNeutrals applies neutral placement and returns a new game state
func (s *GameState) ApplyNeutrals(positions []Position) *GameState {
	newState := s.Clone()
	player := newState.GetYourPlayer()
	if player == nil {
		return newState
	}

	player.HasUsedNeutrals = true

	for _, pos := range positions {
		newState.Board.SetCell(pos, protocol.CellNeutral)
		// Remove from player's cells
		player.RemoveCell(pos)
	}

	// Advance player (using neutrals ends your turn)
	newState.AdvancePlayer()

	return newState
}
