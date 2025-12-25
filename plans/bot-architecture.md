# Virus Bot - Go Implementation Architecture

## Overview

This document outlines the architecture for a fresh Go implementation of a Virus Game bot that connects to the existing virusgame backend via WebSocket.

## Game Rules Summary

- **Grid**: 10x10 (customizable)
- **Players**: 2-4 players (symbols: X, O, △, □)
- **Starting**: Each player has one base cell in a corner
- **Moves**: 3 moves per turn (Grow or Attack)
- **Key Rule**: Can only expand from cells connected to base
- **Fortified**: Captured opponent cells become fortified (cannot be re-taken)
- **Neutrals**: Once per game, place 2 neutral blocks on your cells
- **Win**: Last player standing

## WebSocket Protocol

### Connection Flow
```
Client → Server: connect
Server → Client: welcome {userId, name}
Client → Server: join_lobby {lobbyId}
Server → Client: game_start {board, players, yourPlayerId}
... game loop ...
Client → Server: move {row, col}
Server → Client: move_made {row, col, playerId}
Server → Client: game_end {winner}
```

### Message Types

| Direction | Message | Payload |
|-----------|---------|---------|
| C→S | `connect` | - |
| S→C | `welcome` | `{userId, name}` |
| C→S | `join_lobby` | `{lobbyId}` |
| C→S | `create_lobby` | `{boardSize}` |
| S→C | `game_start` | `{board, players, currentPlayer, yourPlayerId}` |
| C→S | `move` | `{row, col}` |
| S→C | `move_made` | `{row, col, playerId, cellType}` |
| S→C | `game_end` | `{winner, eliminated}` |

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
│   │   └── rules.go          # Game rules validation
│   ├── strategy/
│   │   ├── evaluator.go      # Move scoring/evaluation
│   │   ├── analyzer.go       # Board analysis
│   │   ├── mcts.go           # Monte Carlo Tree Search
│   │   └── neutral.go        # Neutral placement strategy
│   └── protocol/
│       └── messages.go       # WebSocket message types
├── pkg/
│   └── utils/
│       └── logger.go         # Logging utilities
├── config/
│   └── config.go             # Configuration
├── go.mod
└── README.md
```

## Core Components

### 1. Board Representation (`internal/game/board.go`)

```go
type CellType int

const (
    Empty CellType = iota
    Player1     // X
    Player2     // O
    Player3     // Triangle
    Player4     // Square
    Neutral
)

type Position struct {
    Row, Col int
}

type Board struct {
    Size     int
    Cells    [][]CellType
    BasePos  map[PlayerID]Position  // Starting base positions
}
```

### 2. Player State (`internal/game/player.go`)

```go
type Player struct {
    ID       PlayerID
    Symbol   CellType
    BasePos  Position
    Cells    []Position
    IsAlive  bool
    HasUsedNeutrals bool
}
```

### 3. Move Validator (`internal/game/rules.go`)

```go
// ValidMove checks if a move is legal
func ValidMove(board *Board, playerID PlayerID, pos Position) bool

// IsConnectedToBase checks if cell is reachable from base
func IsConnectedToBase(board *Board, playerID PlayerID, pos Position) bool

// GetReachableCells returns all cells connected to base
func GetReachableCells(board *Board, playerID PlayerID) []Position
```

### 4. Strategy Interface (`internal/strategy/interface.go`)

```go
type Strategy interface {
    Name() string
    DecideMoves(gameState *GameState, count int) []Position
    DecideNeutrals(gameState *GameState) []Position
}
```

### 5. Heuristic Strategy (`internal/strategy/evaluator.go`)

Multi-factor scoring with 6 weighted criteria. Fast and deterministic.

```go
type MoveScore struct {
    Position    Position
    MoveType    MoveType  // Grow, Attack
    TotalScore  float64
    Factors     map[string]float64
}

type EvaluationFactors struct {
    TerritoryGain      float64  // Weight for gaining territory
    StrategicPosition  float64  // Weight for corner/edge control
    ThreatRemoval      float64  // Weight for eliminating opponent
    Connectivity       float64  // Weight for maintaining base connection
    ExpansionPotential float64  // Weight for opening new paths
    DefensiveValue     float64  // Weight for protecting own cells
}
```

**Scoring Factors**:
1. **Territory Gain**: +10 per cell captured
2. **Strategic Position**: +5 for edge cells, +8 for corner cells
3. **Threat Removal**: +15 for attacking fortified-able opponent cells
4. **Connectivity**: +3 for cells that reconnect cut-off groups
5. **Expansion Potential**: +4 for cells with multiple empty neighbors
6. **Defensive Value**: +2 for cells adjacent to own territory

### 6. MCTS Strategy (`internal/strategy/mcts.go`)

Monte Carlo Tree Search - probabilistic algorithm that simulates random game outcomes.

```go
type MCTSConfig struct {
    Iterations       int           // Number of simulations per move (default: 1000)
    TimeLimit        time.Duration // Max time for decision (default: 1s)
    ExplorationConst float64       // UCT exploration parameter (default: 1.41)
    MaxDepth         int           // Max simulation depth (default: 50)
}

type MCTS struct {
    Root   *Node
    Config *MCTSConfig
}

type Node struct {
    State           *GameState
    Parent          *Node
    Children        []*Node
    Visits          int
    Wins            float64
    UnexpandedMoves []Position
}
```

**MCTS Phases**:
1. **Selection**: Traverse tree using UCT formula
2. **Expansion**: Add new child node for unexplored move
3. **Simulation**: Random playout from new state
4. **Backpropagation**: Update statistics along path

**UCT Formula**:
```
UCT = (wins/visits) + C * sqrt(ln(parent_visits) / visits)
```

**Virus Game MCTS Adaptations**:
- Handle 3 moves per turn (select best combination)
- Account for base connection rule in simulations
- Simulate neutral placement when available and beneficial
- Multi-player handling (4-way interactions)

### 7. Turn Decision Logic (`internal/strategy/analyzer.go`)

```go
func (s *Strategy) DecideMoves(gameState *GameState, count int) []Position {
    // 1. Generate all valid moves
    validMoves := s.GetValidMoves(gameState, gameState.CurrentPlayer)
    
    // 2. Score each move using strategy
    scoredMoves := s.ScoreMoves(validMoves, gameState)
    
    // 3. Select top count moves (with diversity check)
    selected := s.SelectDiverseMoves(scoredMoves, count)
    
    return selected
}
```

### 8. Neutral Placement Strategy (`internal/strategy/neutral.go`)

Neutrals are placed strategically to block opponent advancement:

```go
func (s *Strategy) DecideNeutrals(gameState *GameState) []Position {
    // Find best defensive positions:
    // 1. Cells that block opponent's path to our base
    // 2. Cells that create chokepoints
    // 3. Cells in strategic corners
    
    return bestPositions
}
```

## Game State Machine

```
┌─────────┐
│  Idle   │ ←──────────────┐
└────┬────┘                │
     │ connect              │
     ▼                     │
┌─────────┐     ┌──────────┘
│ Connecting──►│ In Lobby │
└─────────┘     └────┬─────┘
               join_lobby │
                       ┌──┴──┐
                       ▼     │
                  ┌────────  │
                  │ Playing  │
                  └────┬─────┘
                       │
                  game_end │
                       ▼
                  ┌──────────┐
                  │ Ended    │
                  └──────────┘
```

## WebSocket Client (`internal/client/websocket.go`)

```go
type Client struct {
    Conn        *websocket.Conn
    UserID      string
    PlayerID    int
    CurrentGame *GameState
    Strategy    Strategy
    Incoming    chan []byte
}

func (c *Client) Run(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case msg := <-c.Incoming:
            c.handleMessage(msg)
        }
    }
}
```

## Configuration

```go
type Config struct {
    ServerURL    string        `env:"VIRUSBOT_SERVER_URL" default:"ws://localhost:8080/ws"`
    BotName      string        `env:"VIRUSBOT_NAME" default:"VirusBot"`
    LobbyID      string        `env:"VIRUSBOT_LOBBY"`
    AutoJoin     bool          `env:"VIRUSBOT_AUTO_JOIN"`
    AutoCreate   bool          `env:"VIRUSBOT_AUTO_CREATE"`
    MoveDelay    time.Duration `env:"VIRUSBOT_MOVE_DELAY" default:"500ms"`
    Debug        bool          `env:"VIRUSBOT_DEBUG"`
    Strategy     string        `env:"VIRUSBOT_STRATEGY" default:"mcts"` // "heuristic" or "mcts"
    
    // MCTS Configuration
    MCTSIterations int           `env:"VIRUSBOT_MCTS_ITERATIONS" default:"1000"`
    MCTSTimeLimit  time.Duration `env:"VIRUSBOT_MCTS_TIME_LIMIT" default:"1s"`
    MCTSUCTConst   float64       `env:"VIRUSBOT_MCTS_UCT_CONST" default:"1.41"`
    
    // Heuristic Weights
    WeightTerritory    float64 `env:"VIRUSBOT_WGT_TERRITORY" default:"1.0"`
    WeightStrategic    float64 `env:"VIRUSBOT_WGT_STRATEGIC" default:"0.5"`
    WeightThreat       float64 `env:"VIRUSBOT_WGT_THREAT" default:"1.5"`
    WeightConnectivity float64 `env:"VIRUSBOT_WGT_CONNECTIVITY" default:"0.3"`
    WeightExpansion    float64 `env:"VIRUSBOT_WGT_EXPANSION" default:"0.4"`
    WeightDefensive    float64 `env:"VIRUSBOT_WGT_DEFENSIVE" default:"0.2"`
}
```

## Threat Detection

```go
func (s *Strategy) assessThreats(gameState *GameState) []Threat {
    threats := []Threat{}
    
    // Check for opponent cells adjacent to our base
    // Check for cells that would cut off our connection
    // Identify opponent expansion patterns
    
    return threats
}
```

## Testing Strategy

1. **Unit Tests**
   - Board operations
   - Move validation
   - Base connection detection
   - Move scoring consistency
   - MCTS UCT calculations

2. **Integration Tests**
   - WebSocket connection
   - Message parsing
   - Game state updates

3. **Bot vs Bot Testing**
   - Run multiple bot instances
   - Verify no crashes
   - Validate reasonable move patterns
   - Compare heuristic vs MCTS performance

## Implementation Phases

### Phase 1: Core Infrastructure
- Project setup (go.mod, directory structure)
- Configuration loading
- WebSocket client basic connection
- Message types and parsing

### Phase 2: Game Logic
- Board representation and operations
- Player state tracking
- Move validation
- Base connection detection

### Phase 3: Strategy
- Move generator
- Heuristic evaluator
- Turn decision algorithm
- Neutral placement

### Phase 4: MCTS (Optional)
- MCTS core implementation
- UCT selection
- Simulation engine
- Backpropagation

### Phase 5: Integration
- Lobby management
- Game loop coordination
- Error handling and reconnection
- Logging and debugging

### Phase 6: Testing & Polish
- Unit tests
- Integration tests
- Performance optimization
- Documentation
