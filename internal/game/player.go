package game

import (
	"virusbot/internal/protocol"
)

// Player represents a player in the game
type Player struct {
	ID              int
	Name            string
	Symbol          protocol.CellType
	BasePos         Position
	Cells           []Position
	IsAlive         bool
	HasUsedNeutrals bool
}

// NewPlayer creates a new player
func NewPlayer(id int, name string, symbol protocol.CellType, basePos Position) *Player {
	return &Player{
		ID:      id,
		Name:    name,
		Symbol:  symbol,
		BasePos: basePos,
		Cells:   []Position{basePos},
		IsAlive: true,
	}
}

// PlayerFromInfo creates a player from protocol info
func PlayerFromInfo(info protocol.PlayerInfo) *Player {
	return NewPlayer(info.ID, info.Name, info.Symbol, Position{
		Row: info.Position.Row,
		Col: info.Position.Col,
	})
}

// AddCell adds a cell to the player's territory
func (p *Player) AddCell(pos Position) {
	p.Cells = append(p.Cells, pos)
}

// RemoveCell removes a cell from the player's territory
func (p *Player) RemoveCell(pos Position) {
	for i, cell := range p.Cells {
		if cell.Row == pos.Row && cell.Col == pos.Col {
			p.Cells = append(p.Cells[:i], p.Cells[i+1:]...)
			break
		}
	}
	if len(p.Cells) == 0 {
		p.IsAlive = false
	}
}

// CellCount returns the number of cells owned by the player
func (p *Player) CellCount() int {
	return len(p.Cells)
}

// HasBase checks if the player still has their base
func (p *Player) HasBase() bool {
	for _, cell := range p.Cells {
		if cell.Row == p.BasePos.Row && cell.Col == p.BasePos.Col {
			return true
		}
	}
	return false
}

// Clone creates a copy of the player
func (p *Player) Clone() *Player {
	newCells := make([]Position, len(p.Cells))
	copy(newCells, p.Cells)

	return &Player{
		ID:              p.ID,
		Name:            p.Name,
		Symbol:          p.Symbol,
		BasePos:         p.BasePos,
		Cells:           newCells,
		IsAlive:         p.IsAlive,
		HasUsedNeutrals: p.HasUsedNeutrals,
	}
}

// PlayersFromInfo creates players from protocol player info
func PlayersFromInfo(infos []protocol.PlayerInfo) []*Player {
	players := make([]*Player, len(infos))
	for i, info := range infos {
		players[i] = PlayerFromInfo(info)
	}
	return players
}
