package trader

import (
	"fmt"
	"nofx/kernel"
	"nofx/logger"
	"sync"
)

// StrategyFusionEngine combines multiple strategies with weighted voting
// This allows trading with consensus from multiple approaches
type StrategyFusionEngine struct {
	mu sync.RWMutex

	strategies        map[string]*StrategyProfile
	decisionWeights   map[string]float64 // weight for each strategy (0-1)
	consensusRequired float64            // minimum consensus strength required (0-1)
	traderID          string
}

// StrategyProfile stores a strategy's configuration and recent performance
type StrategyProfile struct {
	Name         string
	Weight       float64 // voting weight
	WinRate      float64 // recent win rate
	ProfitFactor float64 // profit factor
	IsActive     bool
}

// FusionDecision represents a decision combining multiple strategies
type FusionDecision struct {
	Symbol            string
	Action            string  // open_long, open_short, close_long, close_short, hold
	Confidence        int     // 0-100
	ConsensusStrength float64 // how much agreement between strategies (0-1)
	Reasoning         string
	StrategyVotes     map[string]string  // strategy name -> decision
	StrategyScores    map[string]float64 // strategy name -> confidence score
}

// NewStrategyFusionEngine creates a new fusion engine
func NewStrategyFusionEngine(traderID string) *StrategyFusionEngine {
	return &StrategyFusionEngine{
		traderID:          traderID,
		strategies:        make(map[string]*StrategyProfile),
		decisionWeights:   make(map[string]float64),
		consensusRequired: 0.65, // 65% consensus by default
	}
}

// RegisterStrategy registers a new strategy in the fusion engine
func (sfe *StrategyFusionEngine) RegisterStrategy(name string, weight float64, active bool) {
	sfe.mu.Lock()
	defer sfe.mu.Unlock()

	// Normalize weight
	if weight <= 0 {
		weight = 1.0 / float64(len(sfe.strategies)+1)
	}

	sfe.strategies[name] = &StrategyProfile{
		Name:         name,
		Weight:       weight,
		WinRate:      0.5, // assume neutral until proven otherwise
		ProfitFactor: 1.0,
		IsActive:     active,
	}

	// Recalculate weights
	sfe.normalizeWeights()
}

// normalizeWeights normalizes strategy weights so they sum to 1.0
func (sfe *StrategyFusionEngine) normalizeWeights() {
	var totalWeight float64
	for _, profile := range sfe.strategies {
		if profile.IsActive {
			totalWeight += profile.Weight
		}
	}

	if totalWeight > 0 {
		for name, profile := range sfe.strategies {
			if profile.IsActive {
				sfe.decisionWeights[name] = profile.Weight / totalWeight
			}
		}
	}
}

// FuseDecisions combines multiple strategy decisions into one
// strategyDecisions: map of strategy name -> decision (FullDecision with Decisions array)
func (sfe *StrategyFusionEngine) FuseDecisions(
	symbol string,
	strategyDecisions map[string]*kernel.FullDecision,
) *FusionDecision {
	sfe.mu.Lock()
	defer sfe.mu.Unlock()

	if len(strategyDecisions) == 0 {
		return &FusionDecision{
			Symbol:            symbol,
			Action:            "hold",
			Confidence:        0,
			ConsensusStrength: 0,
			Reasoning:         "No strategy decisions provided",
		}
	}

	// ========================================
	// 1. Vote for Each Action
	// ========================================
	actionVotes := make(map[string]float64)      // action -> weighted vote count
	confidenceScores := make(map[string]float64) // strategy -> confidence

	var maxConfidence int
	bestAction := "hold"

	for strategyName, decision := range strategyDecisions {
		strategy, exists := sfe.strategies[strategyName]
		if !exists || !strategy.IsActive {
			continue
		}

		weight := sfe.decisionWeights[strategyName]
		if weight <= 0 || decision == nil || len(decision.Decisions) == 0 {
			continue
		}

		// Weight the vote by strategy weight and confidence from first decision
		firstDecision := decision.Decisions[0]
		confidenceMultiplier := float64(firstDecision.Confidence) / 100.0
		weightedVote := weight * confidenceMultiplier

		for _, d := range decision.Decisions {
			actionVotes[d.Action] += weightedVote
			confidenceScores[strategyName] = float64(d.Confidence)
			if d.Confidence > maxConfidence {
				maxConfidence = d.Confidence
			}
		}
	}

	// ========================================
	// 2. Determine Best Action
	// ========================================
	var maxVotes float64
	for action, votes := range actionVotes {
		if votes > maxVotes {
			maxVotes = votes
			bestAction = action
		}
	}

	consensusStrength := maxVotes // already normalized to 0-1 range

	// ========================================
	// 3. Blend Confidence from All Strategies
	// ========================================
	var totalConfidence float64
	activeCount := 0
	for _, fullDec := range strategyDecisions {
		if fullDec != nil && len(fullDec.Decisions) > 0 {
			for _, decision := range fullDec.Decisions {
				totalConfidence += float64(decision.Confidence)
				activeCount++
			}
		}
	}

	finalConfidence := int(totalConfidence / float64(activeCount))
	if finalConfidence < 0 {
		finalConfidence = 0
	}
	if finalConfidence > 100 {
		finalConfidence = 100
	}

	// ========================================
	// 4. Apply Consensus Requirement
	// ========================================
	// If consensus is weak, reduce confidence
	if consensusStrength < sfe.consensusRequired {
		// Reduce confidence based on how far below consensus requirement
		penalty := (sfe.consensusRequired - consensusStrength) * 50
		finalConfidence = int(float64(finalConfidence) - penalty)
		if finalConfidence < 0 {
			finalConfidence = 0
		}
		logger.Infof("[StrategyFusion] ⚠️ Low consensus (%.2f%% < %.2f%%), reducing confidence to %d",
			consensusStrength*100, sfe.consensusRequired*100, finalConfidence)
	}

	// ========================================
	// 5. Build Reasoning
	// ========================================
	reasoning := fmt.Sprintf("Consensus: %d strategies voted on %s with %.2f%% agreement. Final confidence: %d",
		activeCount, bestAction, consensusStrength*100, finalConfidence)

	return &FusionDecision{
		Symbol:            symbol,
		Action:            bestAction,
		Confidence:        finalConfidence,
		ConsensusStrength: consensusStrength,
		Reasoning:         reasoning,
		StrategyVotes:     extractActionVotes(strategyDecisions),
		StrategyScores:    confidenceScores,
	}
}

// UpdateStrategyPerformance updates a strategy's historical performance
func (sfe *StrategyFusionEngine) UpdateStrategyPerformance(
	strategyName string,
	winRate float64,
	profitFactor float64,
) {
	sfe.mu.Lock()
	defer sfe.mu.Unlock()

	if strategy, exists := sfe.strategies[strategyName]; exists {
		strategy.WinRate = winRate
		strategy.ProfitFactor = profitFactor

		// Adjust weight based on performance (optional: better performing strategies get more weight)
		if profitFactor > 1.5 {
			strategy.Weight = strategy.Weight * 1.1 // boost weight for good performers
		} else if profitFactor < 0.8 {
			strategy.Weight = strategy.Weight * 0.9 // reduce weight for poor performers
		}

		logger.Infof("[StrategyFusion] Updated %s: WinRate=%.2f%%, ProfitFactor=%.2f, Weight=%.2f",
			strategyName, winRate*100, profitFactor, strategy.Weight)

		sfe.normalizeWeights()
	}
}

// EnableStrategy enables a strategy for voting
func (sfe *StrategyFusionEngine) EnableStrategy(name string) {
	sfe.mu.Lock()
	defer sfe.mu.Unlock()

	if strategy, exists := sfe.strategies[name]; exists {
		strategy.IsActive = true
		sfe.normalizeWeights()
		logger.Infof("[StrategyFusion] ✅ Enabled strategy: %s", name)
	}
}

// DisableStrategy disables a strategy from voting
func (sfe *StrategyFusionEngine) DisableStrategy(name string) {
	sfe.mu.Lock()
	defer sfe.mu.Unlock()

	if strategy, exists := sfe.strategies[name]; exists {
		strategy.IsActive = false
		sfe.normalizeWeights()
		logger.Infof("[StrategyFusion] ❌ Disabled strategy: %s", name)
	}
}

// SetConsensusRequired sets the minimum consensus strength required (0-1)
func (sfe *StrategyFusionEngine) SetConsensusRequired(threshold float64) {
	sfe.mu.Lock()
	defer sfe.mu.Unlock()

	if threshold < 0 {
		threshold = 0
	}
	if threshold > 1 {
		threshold = 1
	}

	sfe.consensusRequired = threshold
	logger.Infof("[StrategyFusion] 🎯 Consensus requirement set to %.2f%%", threshold*100)
}

// GetStrategyStats returns statistics for all strategies
func (sfe *StrategyFusionEngine) GetStrategyStats() map[string]map[string]interface{} {
	sfe.mu.RLock()
	defer sfe.mu.RUnlock()

	stats := make(map[string]map[string]interface{})
	for name, profile := range sfe.strategies {
		stats[name] = map[string]interface{}{
			"weight":        profile.Weight,
			"win_rate":      profile.WinRate,
			"profit_factor": profile.ProfitFactor,
			"active":        profile.IsActive,
		}
	}
	return stats
}

// extractActionVotes extracts the action from each strategy decision
func extractActionVotes(decisions map[string]*kernel.FullDecision) map[string]string {
	votes := make(map[string]string)
	for strategyName, fullDec := range decisions {
		if fullDec != nil && len(fullDec.Decisions) > 0 {
			votes[strategyName] = fullDec.Decisions[0].Action
		}
	}
	return votes
}

// String returns formatted fusion engine state
func (sfe *StrategyFusionEngine) String() string {
	sfe.mu.RLock()
	defer sfe.mu.RUnlock()

	var activeCount int
	for _, strategy := range sfe.strategies {
		if strategy.IsActive {
			activeCount++
		}
	}

	return fmt.Sprintf("[StrategyFusion] %d/%d strategies active, consensus required: %.2f%%",
		activeCount, len(sfe.strategies), sfe.consensusRequired*100)
}
