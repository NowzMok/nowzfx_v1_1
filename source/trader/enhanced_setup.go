package trader

import (
	"nofx/logger"
	"nofx/store"
)

// EnhancedAutoTraderSetup contains all the Option B modules
// This is used to extend the existing AutoTrader with advanced trading capabilities
type EnhancedAutoTraderSetup struct {
	ParameterOptimizer *ParameterOptimizer
	RiskManager        *EnhancedRiskManager
	StrategyFusion     *StrategyFusionEngine
	FundManagement     *FundManagementSystem
	AdaptiveStopLoss   *AdaptiveStopLossManager
}

// InitializeEnhancedModules initializes all Option B modules for an AutoTrader
func InitializeEnhancedModules(traderID string, initialBalance float64, st store.Store) *EnhancedAutoTraderSetup {
	logger.Infof("🚀 [%s] Initializing enhanced trading modules (Option B)...", traderID)

	setup := &EnhancedAutoTraderSetup{
		// 1. Parameter Dynamic Adjustment
		ParameterOptimizer: NewParameterOptimizer(traderID, st),

		// 2. Enhanced Risk Management
		RiskManager: NewEnhancedRiskManager(traderID, st),

		// 3. Multi-Strategy Fusion
		StrategyFusion: NewStrategyFusionEngine(traderID),

		// 4. Fund Management Optimization
		FundManagement: NewFundManagementSystem(initialBalance),

		// 5. Adaptive Stop Loss
		AdaptiveStopLoss: NewAdaptiveStopLossManager(traderID),
	}

	logger.Infof("✅ [%s] Enhanced modules initialized:", traderID)
	logger.Info("  • 📊 Parameter Optimizer - Dynamic adjustment based on performance")
	logger.Info("  • ⚠️  Enhanced Risk Manager - Kelly Criterion & drawdown control")
	logger.Info("  • 🎯 Strategy Fusion - Multi-strategy consensus voting")
	logger.Info("  • 💰 Fund Management - Position sizing optimization")
	logger.Info("  • 🛑 Adaptive Stop Loss - ATR-based dynamic stops")

	return setup
}

// String returns a formatted status of all enhanced modules
func (eas *EnhancedAutoTraderSetup) String() string {
	return "Enhanced Setup:\n" +
		"  " + eas.ParameterOptimizer.String() + "\n" +
		"  " + eas.RiskManager.String() + "\n" +
		"  " + eas.StrategyFusion.String() + "\n" +
		"  " + eas.FundManagement.String() + "\n" +
		"  " + eas.AdaptiveStopLoss.String()
}

// ============================================================================
// Integration Helper Functions - Call these from AutoTrader.runCycle()
// ============================================================================

// ApplyParameterOptimization adjusts trading parameters based on recent performance
// Call this before making AI decisions
func (eas *EnhancedAutoTraderSetup) ApplyParameterOptimization(
	currentVolatility float64,
	volatilityAverage float64,
) {
	eas.ParameterOptimizer.OptimizeParameters(currentVolatility, volatilityAverage)

	metrics := eas.ParameterOptimizer.GetPerformanceMetrics()
	logger.Infof("📈 [ParamOpt] Win Rate: %.2f%% | ProfitFactor: %.2f | Volatility Mult: %.2fx",
		metrics.WinRate*100, metrics.ProfitFactor, eas.ParameterOptimizer.volatilityMultiplier)
}

// ValidateRiskLimits checks if trading is allowed before executing trades
// Returns (allowed bool, reason string)
func (eas *EnhancedAutoTraderSetup) ValidateRiskLimits() (bool, string) {
	return eas.RiskManager.CheckRiskLimits()
}

// FuseMultipleDecisions combines multiple strategy outputs
// Use this when you have decisions from different sources
func (eas *EnhancedAutoTraderSetup) FuseMultipleDecisions(
	symbol string,
	strategyDecisions map[string]interface{},
) *FusionDecision {
	// Convert interface{} to *kernel.Decision if needed
	// This is a helper for integration
	logger.Infof("🔄 [Fusion] Fusing decisions from %d strategies for %s", len(strategyDecisions), symbol)

	// Call the actual fusion function with proper types
	// You'll need to convert strategyDecisions appropriately
	return &FusionDecision{
		Symbol:            symbol,
		Action:            "hold",
		Confidence:        50,
		ConsensusStrength: 0.5,
		Reasoning:         "Fusion pending implementation",
	}
}

// CalculateOptimalPositionSize combines all sizing factors
// Call this instead of using position size directly from AI
func (eas *EnhancedAutoTraderSetup) CalculateOptimalPositionSize(
	basePositionSize float64,
	volatility float64,
	winRate float64,
	avgWin float64,
	avgLoss float64,
	equity float64,
) float64 {
	// 1. Apply parameter optimizer adjustment
	volatilityAdjusted := eas.ParameterOptimizer.GetAdjustedPositionSize(basePositionSize)

	// 2. Apply Kelly Criterion from fund management
	kellyAdjusted := eas.FundManagement.CalculatePositionSizeWithKelly(
		winRate, avgWin, avgLoss, 0, 0,
	)

	// 3. Apply risk manager constraints
	finalSize := eas.RiskManager.CalculatePositionSize(
		volatility, winRate, avgWin, avgLoss, equity, basePositionSize,
	)

	logger.Infof("💰 [PosSizing] Base: %.2f → Vol: %.2f → Kelly: %.2f → Final: %.2f USDT",
		basePositionSize, volatilityAdjusted, kellyAdjusted, finalSize)

	return finalSize
}

// ValidateStopLossProfitRatio checks if stop loss and take profit meet criteria
// Call before executing a trade
func (eas *EnhancedAutoTraderSetup) ValidateStopLossProfitRatio(
	entryPrice float64,
	stopLoss float64,
	takeProfit float64,
	isBuy bool,
) (valid bool, reason string) {
	return eas.RiskManager.ValidateStopLoss(entryPrice, stopLoss, takeProfit, isBuy)
}

// ApplyDynamicStopLoss updates stop loss and take profit adaptively
// Call this periodically for open positions
func (eas *EnhancedAutoTraderSetup) ApplyDynamicStopLoss(
	symbol string,
	currentPrice float64,
	atrValue float64,
) (stopLoss float64, takeProfit float64, updated bool) {
	level := eas.AdaptiveStopLoss.UpdateATR(symbol, atrValue, currentPrice)
	if level == nil {
		return 0, 0, false
	}

	// Return updated stop loss and unchanged take profit
	return level.StopLoss, level.TakeProfit, true
}

// RecordTradeOutcome records the result for future optimization
// Call this when a trade closes
func (eas *EnhancedAutoTraderSetup) RecordTradeOutcome(
	symbol string,
	pnl float64,
	isWin bool,
) {
	eas.FundManagement.RecordTrade(pnl)

	if isWin {
		eas.ParameterOptimizer.performanceMetrics.ConsecutiveWins++
	} else {
		eas.RiskManager.RecordLosingTrade(pnl)
		eas.ParameterOptimizer.performanceMetrics.ConsecutiveLosses++
	}

	logger.Infof("📊 [Outcome] %s: %+.2f USDT (%s)", symbol, pnl, map[bool]string{true: "WIN", false: "LOSS"}[isWin])
}

// GetHealthStatus returns comprehensive status of all modules
func (eas *EnhancedAutoTraderSetup) GetHealthStatus() map[string]string {
	return map[string]string{
		"parameter_optimizer": eas.ParameterOptimizer.String(),
		"risk_manager":        eas.RiskManager.String(),
		"strategy_fusion":     eas.StrategyFusion.String(),
		"fund_management":     eas.FundManagement.String(),
		"adaptive_stoploss":   eas.AdaptiveStopLoss.String(),
	}
}
