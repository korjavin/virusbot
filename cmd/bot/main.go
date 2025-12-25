package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"virusbot/config"
	"virusbot/internal/client"
	"virusbot/internal/game"
	"virusbot/internal/strategy"
)

func main() {
	// Parse command line flags
	lobbyID := flag.String("lobby", "", "Lobby ID to join")
	autoCreate := flag.Bool("create", false, "Create a new lobby")
	debug := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override with command line flags
	if *lobbyID != "" {
		cfg.LobbyID = *lobbyID
	}
	if *autoCreate {
		cfg.AutoCreate = true
	}
	if *debug {
		cfg.Debug = true
	}

	log.Printf("Starting Virus Bot (%s strategy)", cfg.Strategy)

	// Create strategy
	strategy := strategy.NewStrategy(cfg)
	log.Printf("Using strategy: %s", strategy.Name())

	// Create game state
	gameState := &game.GameState{}

	// Create callback for handling game events
	callback := func(event string, data interface{}) {
		switch event {
		case "connected":
			log.Printf("Connected to game server!")
			if cfg.LobbyID != "" {
				log.Printf("Joining lobby: %s", cfg.LobbyID)
			} else if cfg.AutoCreate {
				log.Println("Creating new lobby...")
			}

		case "game_start":
			log.Println("Game started!")
			// Update local game state
			if msg, ok := data.(*client.GameState); ok {
				gameState.Board = game.NewBoardFromData(
					msg.Board,
					nil, // Base positions are in the board
				)
			}

		case "move_made":
			log.Println("Move made by opponent")

		case "game_end":
			log.Println("Game ended!")

		case "disconnected":
			log.Println("Disconnected from server")
		}
	}

	// Create WebSocket client
	wsClient := client.NewClient(cfg, callback)

	// Connect to server
	if err := wsClient.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the client in a goroutine
	go func() {
		if err := wsClient.Run(); err != nil {
			log.Printf("Client error: %v", err)
			cancel()
		}
	}()

	// Main loop - handle turns
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down...")
			wsClient.Disconnect()
			return

		case <-sigChan:
			log.Println("Received shutdown signal")
			cancel()
			wsClient.Disconnect()
			return

		case <-ticker.C:
			if wsClient.IsMyTurn() {
				state := wsClient.GetGameState()
				if state != nil {
					// Convert to game state
					gs := convertToGameState(state)

					// Get strategy moves
					moves := strategy.DecideMoves(gs, 3)

					// Execute moves
					for _, move := range moves {
						if err := wsClient.MakeMove(move.Position.Row, move.Position.Col); err != nil {
							log.Printf("Failed to make move: %v", err)
						} else {
							log.Printf("Made move: (%d, %d)", move.Position.Row, move.Position.Col)
						}
						time.Sleep(cfg.MoveDelay)
					}
				}
			}
		}
	}
}

// convertToGameState converts the client.GameState to game.GameState
func convertToGameState(cs *client.GameState) *game.GameState {
	if cs == nil {
		return nil
	}

	// Build base positions from players
	basePos := make(map[int]game.Position)
	for _, p := range cs.Players {
		basePos[p.ID] = game.Position{
			Row: p.Position.Row,
			Col: p.Position.Col,
		}
	}

	board := game.NewBoardFromData(cs.Board, basePos)

	players := make([]*game.Player, len(cs.Players))
	for i, p := range cs.Players {
		players[i] = &game.Player{
			ID:      p.ID,
			Name:    p.Name,
			Symbol:  p.Symbol,
			BasePos: game.Position{Row: p.Position.Row, Col: p.Position.Col},
			IsAlive: true,
		}
	}

	return &game.GameState{
		Board:         board,
		Players:       players,
		CurrentPlayer: cs.CurrentPlayer,
		YourPlayerID:  cs.YourPlayerID,
	}
}
