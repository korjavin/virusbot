package strategy

import (
	"virusbot/config"
)

// NewStrategy creates a strategy based on configuration
func NewStrategy(cfg *config.Config) Strategy {
	switch cfg.GetStrategyType() {
	case config.StrategyMCTS:
		return NewMCTSStrategy(cfg)
	default:
		return NewHeuristicStrategy(cfg)
	}
}
