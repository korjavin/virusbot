# Virus Bot

A Go implementation of an autonomous bot for the Virus Game. This bot connects to a Virus Game server via WebSocket and plays autonomously using either a heuristic-based strategy or Monte Carlo Tree Search (MCTS).

## Game Rules

The Virus Game is a turn-based strategy game played on a 10x10 grid:

- **Players**: 2-4 players (X, O, △, □)
- **Starting Position**: Each player has one base cell in a corner
- **Moves**: 3 moves per turn (Grow or Attack)
- **Key Rule**: Can only expand from cells connected to your base
- **Fortified Cells**: Captured opponent cells become fortified (cannot be re-taken)
- **Neutrals**: Once per game, place 2 neutral blocks on your cells
- **Win**: Last player standing

## Installation

```bash
# Clone the repository
git clone <repository-url>
cd virusbot

# Download dependencies
go mod download

# Build the bot
go build -o virusbot cmd/bot/main.go
```

## Usage

```bash
# Connect to local server with heuristic strategy (default)
./virusbot

# Connect to remote server
./virusbot -server ws://game.example.com/ws

# Join a specific lobby
./virusbot -lobby my-lobby-id

# Create a new lobby
./virusbot -create

# Use heuristic strategy
VIRUSBOT_STRATEGY=heuristic ./virusbot

# Enable debug logging
./virusbot -debug
```

## Configuration

The bot can be configured via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `VIRUSBOT_SERVER_URL` | `ws://localhost:8080/ws` | WebSocket server URL |
| `VIRUSBOT_NAME` | `VirusBot` | Bot display name |
| `VIRUSBOT_LOBBY` | - | Lobby ID to join |
| `VIRUSBOT_AUTO_JOIN` | `false` | Auto-join available lobby |
| `VIRUSBOT_AUTO_CREATE` | `false` | Auto-create new lobby |
| `VIRUSBOT_MOVE_DELAY` | `500ms` | Delay between moves |
| `VIRUSBOT_DEBUG` | `false` | Enable debug logging |
| `VIRUSBOT_STRATEGY` | `mcts` | Strategy: `heuristic` or `mcts` |
| `VIRUSBOT_MCTS_ITERATIONS` | `1000` | MCTS iterations per move |
| `VIRUSBOT_MCTS_TIME_LIMIT` | `1s` | MCTS time limit per move |

### Heuristic Weights

Customize the heuristic strategy weights:

| Variable | Default | Description |
|----------|---------|-------------|
| `VIRUSBOT_WGT_TERRITORY` | `1.0` | Territory gain weight |
| `VIRUSBOT_WGT_STRATEGIC` | `0.5` | Strategic position weight |
| `VIRUSBOT_WGT_THREAT` | `1.5` | Threat removal weight |
| `VIRUSBOT_WGT_CONNECTIVITY` | `0.3` | Connectivity weight |
| `VIRUSBOT_WGT_EXPANSION` | `0.4` | Expansion potential weight |
| `VIRUSBOT_WGT_DEFENSIVE` | `0.2` | Defensive value weight |

## Strategies

### MCTS Strategy (Default)

Monte Carlo Tree Search simulates random game outcomes to evaluate moves:

- **Selection**: Traverse tree using UCT formula
- **Expansion**: Add new child node for unexplored move
- **Simulation**: Random playout from new state
- **Backpropagation**: Update statistics along path

### Heuristic Strategy

Uses a multi-factor scoring system with 6 weighted criteria:

1. **Territory Gain** (+10 per cell captured)
2. **Strategic Position** (+5 for edge, +8 for corner cells)
3. **Threat Removal** (+15 for attacking opponent cells)
4. **Connectivity** (+3 for reconnecting cut-off groups)
5. **Expansion Potential** (+4 for cells with multiple empty neighbors)
6. **Defensive Value** (+2 for cells adjacent to own territory)

## Project Structure

```
virusbot/
├── cmd/
│   └── bot/
│       └── main.go           # Entry point
├── internal/
│   ├── client/
│   │   └── websocket.go      # WebSocket connection handling
│   ├── game/
│   │   ├── board.go          # Board state and operations
│   │   ├── player.go         # Player tracking
│   │   ├── rules.go          # Game rules validation
│   │   └── state.go          # Game state management
│   ├── protocol/
│   │   └── messages.go       # WebSocket message types
│   └── strategy/
│       ├── interface.go      # Strategy interface
│       ├── evaluator.go      # Heuristic move scoring
│       ├── mcts.go           # Monte Carlo Tree Search
│       └── factory.go        # Strategy factory
├── config/
│   └── config.go             # Configuration
├── go.mod
└── README.md
```

## Development

### Running Tests

```bash
go test ./...
```

### Adding New Strategies

Implement the `Strategy` interface in `internal/strategy/`:

```go
type Strategy interface {
    Name() string
    DecideMoves(state *game.GameState, count int) []game.Move
    DecideNeutrals(state *game.GameState) []game.Position
    OnMoveMade(state *game.GameState, move game.Move)
}
```

## License

MIT
