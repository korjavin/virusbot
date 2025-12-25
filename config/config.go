package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the bot
type Config struct {
	// Server connection
	ServerURL string `env:"VIRUSBOT_SERVER_URL" default:"ws://localhost:8080/ws"`

	// Bot identity
	BotName string `env:"VIRUSBOT_NAME" default:"VirusBot"`

	// Lobby settings
	LobbyID    string `env:"VIRUSBOT_LOBBY"`
	AutoJoin   bool   `env:"VIRUSBOT_AUTO_JOIN"`
	AutoCreate bool   `env:"VIRUSBOT_AUTO_CREATE"`

	// Game behavior
	MoveDelay          time.Duration `env:"VIRUSBOT_MOVE_DELAY" default:"500ms"`
	Debug              bool          `env:"VIRUSBOT_DEBUG"`
	AutoAcceptChallenge bool         `env:"VIRUSBOT_AUTO_ACCEPT_CHALLENGE" default:"true"`

	// Strategy selection
	Strategy string `env:"VIRUSBOT_STRATEGY" default:"mcts"` // "heuristic" or "mcts"

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

// StrategyType represents the strategy to use
type StrategyType string

const (
	StrategyHeuristic StrategyType = "heuristic"
	StrategyMCTS      StrategyType = "mcts"
)

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if present
	_ = godotenv.Load()

	cfg := &Config{
		ServerURL:           getEnv("VIRUSBOT_SERVER_URL", "ws://localhost:8080/ws"),
		BotName:             getEnv("VIRUSBOT_NAME", "VirusBot"),
		LobbyID:             getEnv("VIRUSBOT_LOBBY", ""),
		AutoJoin:            getEnvBool("VIRUSBOT_AUTO_JOIN"),
		AutoCreate:          getEnvBool("VIRUSBOT_AUTO_CREATE"),
		MoveDelay:           getEnvDuration("VIRUSBOT_MOVE_DELAY", 500*time.Millisecond),
		Debug:               getEnvBool("VIRUSBOT_DEBUG"),
		AutoAcceptChallenge: getEnvBool("VIRUSBOT_AUTO_ACCEPT_CHALLENGE"),
		Strategy:           getEnv("VIRUSBOT_STRATEGY", "heuristic"),
		MCTSIterations:     getEnvInt("VIRUSBOT_MCTS_ITERATIONS", 1000),
		MCTSTimeLimit:      getEnvDuration("VIRUSBOT_MCTS_TIME_LIMIT", 1*time.Second),
		MCTSUCTConst:       getEnvFloat("VIRUSBOT_MCTS_UCT_CONST", 1.41),
		WeightTerritory:    getEnvFloat("VIRUSBOT_WGT_TERRITORY", 1.0),
		WeightStrategic:    getEnvFloat("VIRUSBOT_WGT_STRATEGIC", 0.5),
		WeightThreat:       getEnvFloat("VIRUSBOT_WGT_THREAT", 1.5),
		WeightConnectivity: getEnvFloat("VIRUSBOT_WGT_CONNECTIVITY", 0.3),
		WeightExpansion:    getEnvFloat("VIRUSBOT_WGT_EXPANSION", 0.4),
		WeightDefensive:    getEnvFloat("VIRUSBOT_WGT_DEFENSIVE", 0.2),
	}

	return cfg, nil
}

// GetStrategyType returns the strategy as a typed enum
func (c *Config) GetStrategyType() StrategyType {
	switch c.Strategy {
	case "mcts", "MCTS":
		return StrategyMCTS
	default:
		return StrategyHeuristic
	}
}

// Helper functions for environment variables
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvBool(key string) bool {
	val := os.Getenv(key)
	return val == "true" || val == "1" || val == "yes"
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var result int
		if _, err := fmt.Sscanf(val, "%d", &result); err == nil {
			return result
		}
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		var result float64
		if _, err := fmt.Sscanf(val, "%f", &result); err == nil {
			return result
		}
	}
	return defaultVal
}
