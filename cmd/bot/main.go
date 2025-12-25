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
	"virusbot/internal/protocol"
	"virusbot/internal/strategy"
)

func main() {
	// Parse command line flags
	serverURL := flag.String("server", "", "WebSocket server URL (e.g., wss://vs.wandergeek.org/ws)")
	lobbyID := flag.String("lobby", "", "Lobby ID to join")
	autoCreate := flag.Bool("create", false, "Create a new lobby")
	autoAccept := flag.Bool("accept", false, "Auto-accept challenges")
	debug := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override with command line flags
	if *serverURL != "" {
		cfg.ServerURL = *serverURL
	}
	if *lobbyID != "" {
		cfg.LobbyID = *lobbyID
	}
	if *autoCreate {
		cfg.AutoCreate = true
	}
	if *autoAccept {
		cfg.AutoAcceptChallenge = true
	}
	if *debug {
		cfg.Debug = true
	}

	log.Printf("Starting Virus Bot (%s strategy)", cfg.Strategy)
	log.Printf("Connecting to: %s", cfg.ServerURL)

	// Create strategy
	strategy := strategy.NewStrategy(cfg)
	log.Printf("Using strategy: %s", strategy.Name())

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

		case "challenge":
			log.Printf("Challenge received! Auto-accepting...")

		case "game_start":
			log.Println("Game started!")
			// Debug: log the game state
			if msg, ok := data.(*client.GameState); ok {
				log.Printf("GameState from callback: Board=%v, Players=%v, CurrentPlayer=%d, YourPlayerID=%d",
					msg.Board != nil, msg.Players, msg.CurrentPlayer, msg.YourPlayerID)
			}

		case "move_made":
			if msg, ok := data.(*protocol.MoveMadeMessage); ok {
				log.Printf("Player %d moved to (%d, %d), movesLeft=%d", msg.Player, msg.Row, msg.Col, msg.MovesLeft)
			} else {
				log.Println("Move made")
			}

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
			// Refresh game state and check if it's our turn
			state := wsClient.GetGameState()
			if state == nil || !wsClient.IsMyTurn() {
				continue
			}

			log.Printf("It's my turn!")

			// Execute moves - keep making moves until no more valid moves or turn ends
			for i := 0; i < 3; i++ {
				// Refresh game state from server
				state := wsClient.GetGameState()
				if state == nil || state.Board == nil {
					log.Printf("Board is nil, stopping")
					break
				}

				// Check if it's still our turn
				if !wsClient.IsMyTurn() {
					log.Printf("Turn ended")
					break
				}

				// Convert to game state with fresh board
				gs := convertToGameState(state)
				if gs == nil || gs.Board == nil {
					log.Printf("Failed to convert game state")
					break
				}

				// Check if the previous move position is now occupied
				log.Printf("Board state check - cell (8,9) = %d", state.Board[8][9])

				// Get fresh strategy moves (1 at a time)
				moves := strategy.DecideMoves(gs, 1)
				if len(moves) == 0 {
					log.Printf("No more valid moves")
					break
				}

				move := moves[0]
				log.Printf("Strategy suggests: (%d, %d)", move.Position.Row, move.Position.Col)

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

// convertToGameState converts the client.GameState to game.GameState
func convertToGameState(cs *client.GameState) *game.GameState {
	if cs == nil {
		return nil
	}

	// Handle nil Players (new protocol format)
	var players []*game.Player
	if cs.Players != nil {
		players = make([]*game.Player, len(cs.Players))
		for i, p := range cs.Players {
			players[i] = &game.Player{
				ID:      p.ID,
				Name:    p.Name,
				Symbol:  p.Symbol,
				BasePos: game.Position{Row: p.Position.Row, Col: p.Position.Col},
				IsAlive: true,
			}
		}
	}

	// Build base positions from players if available
	basePos := make(map[int]game.Position)
	if cs.Players != nil {
		for _, p := range cs.Players {
			basePos[p.ID] = game.Position{
				Row: p.Position.Row,
				Col: p.Position.Col,
			}
		}
	}

	board := game.NewBoardFromData(cs.Board, basePos)

	return &game.GameState{
		Board:         board,
		Players:       players,
		CurrentPlayer: cs.CurrentPlayer,
		YourPlayerID:  cs.YourPlayerID,
	}
}
