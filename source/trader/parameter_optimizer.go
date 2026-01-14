package trader

import (
	"fmt"
	"math"
	"nofx/logger"
	"nofx/store"
	"sync"
	"time"
)

// ParameterOptimizer handles dynamic parameter adjustment based on market conditions and trading performance
type ParameterOptimizer struct {
	mu                   sync.RWMutex
	traderID             string
	lastOptimizationTime time.Time
	optimizationInterval time.Duration
	performanceMetrics   *PerformanceMetrics
	marketConditionScore float64 // 0-100, higher = stronger trend, lower = choppy
	volatilityMultiplier float64 // 1.0-2.0, multiplies position size based on volatility
	confidenceAdjustment float64 // adjusts min confidence requirement based on market conditions
	leverageAdjustment   float64 // adjusts AI leverage recommendation
	store                store.Store
}

// PerformanceMetrics tracks trading performance for optimization
type PerformanceMetrics struct {
	WinRate              float64 // percentage of winning trades
	ProfitFactor         float64 // gross profit / gross loss
	ConsecutiveWins      int     // current consecutive win count
	ConsecutiveLosses    int     // current consecutive loss count
	MaxConsecutiveLosses int     // max consecutive losses in period
	MaxDrawdown          float64 // maximum drawdown percentage
	SharpRatio           float64 // Sharpe ratio approximation
	TotalPnL             float64 // total P&L in the period
	TradesInPeriod       int     // total trades executed
	LastUpdateTime       time.Time
}

// NewParameterOptimizer creates a new parameter optimizer
func NewParameterOptimizer(traderID string, st store.Store) *ParameterOptimizer {
	return &ParameterOptimizer{
		traderID:             traderID,
		optimizationInterval: 1 * time.Hour, // re-optimize every hour
		performanceMetrics:   &PerformanceMetrics{},
		volatilityMultiplier: 1.0,
		confidenceAdjustment: 0,
		leverageAdjustment:   1.0,
		marketConditionScore: 50, // neutral
		store:                st,
	}
}

// UpdateMetrics updates performance metrics based on recent trading results
// Using store.TraderFill to represent trade results
func (po *ParameterOptimizer) UpdateMetrics(trades []store.TraderFill) {
	po.mu.Lock()
	defer po.mu.Unlock()

	if len(trades) == 0 {
		return
	}

	// Calculate win rate and profit factor
	var winCount, lossCount int
	var grossProfit, grossLoss float64
	var pnlSequence []float64

	for _, fill := range trades {
		pnlSequence = append(pnlSequence, fill.RealizedPnL)
		if fill.RealizedPnL > 0 {
			winCount++
			grossProfit += fill.RealizedPnL
		} else if fill.RealizedPnL < 0 {
			lossCount++
			grossLoss += math.Abs(fill.RealizedPnL)
		}
	}

	po.performanceMetrics.TradesInPeriod = len(trades)
	po.performanceMetrics.WinRate = float64(winCount) / float64(len(trades))
	po.performanceMetrics.TotalPnL = grossProfit - grossLoss

	if grossLoss > 0 {
		po.performanceMetrics.ProfitFactor = grossProfit / grossLoss
	} else if grossProfit > 0 {
		po.performanceMetrics.ProfitFactor = math.Inf(1)
	} else {
		po.performanceMetrics.ProfitFactor = 1.0
	}

	// Calculate consecutive wins/losses
	po.performanceMetrics.ConsecutiveWins = 0
	po.performanceMetrics.ConsecutiveLosses = 0
	po.performanceMetrics.MaxConsecutiveLosses = 0

	for _, pnl := range pnlSequence {
		if pnl > 0 {
			po.performanceMetrics.ConsecutiveWins++
			po.performanceMetrics.ConsecutiveLosses = 0
		} else if pnl < 0 {
			po.performanceMetrics.ConsecutiveLosses++
			if po.performanceMetrics.ConsecutiveLosses > po.performanceMetrics.MaxConsecutiveLosses {
				po.performanceMetrics.MaxConsecutiveLosses = po.performanceMetrics.ConsecutiveLosses
			}
			po.performanceMetrics.ConsecutiveWins = 0
		}
	}

	po.performanceMetrics.LastUpdateTime = time.Now()

	logger.Infof("[Parameter Optimizer] Updated metrics - WinRate: %.2f%%, ProfitFactor: %.2f, Consecutive losses: %d/%d",
		po.performanceMetrics.WinRate*100, po.performanceMetrics.ProfitFactor,
		po.performanceMetrics.ConsecutiveLosses, po.performanceMetrics.MaxConsecutiveLosses)
}

// OptimizeParameters recalculates optimization parameters based on current conditions
func (po *ParameterOptimizer) OptimizeParameters(currentVolatility float64, volatilityAverage float64) {
	po.mu.Lock()
	defer po.mu.Unlock()

	// Check if optimization interval has passed
	if time.Since(po.lastOptimizationTime) < po.optimizationInterval {
		return
	}

	po.lastOptimizationTime = time.Now()

	// ========================================
	// 1. Volatility Multiplier Adjustment
	// ========================================
	// Increase position size when volatility is lower than average (more predictable)
	// Decrease position size when volatility is higher than average (more risky)
	if volatilityAverage > 0 {
		volatilityRatio := currentVolatility / volatilityAverage
		// Apply non-linear scaling: 0.5-2.0 range
		po.volatilityMultiplier = 1.0 / (1.0 + (volatilityRatio-1.0)*0.5)
		po.volatilityMultiplier = math.Max(0.5, math.Min(2.0, po.volatilityMultiplier))
	}

	// ========================================
	// 2. Confidence Adjustment Based on Win Rate
	// ========================================
	// If win rate is below 40%, require higher confidence to trade (be more selective)
	// If win rate is above 60%, allow lower confidence (take more opportunities)
	metrics := po.performanceMetrics
	if metrics.WinRate > 0 {
		if metrics.WinRate < 0.40 {
			// Low win rate: increase min confidence requirement
			po.confidenceAdjustment = 15 // Require 15% higher confidence
		} else if metrics.WinRate > 0.60 {
			// High win rate: relax confidence requirement
			po.confidenceAdjustment = -10 // Allow 10% lower confidence
		} else {
			po.confidenceAdjustment = 0 // Neutral
		}
	}

	// ========================================
	// 3. Leverage Adjustment Based on Drawdown
	// ========================================
	// If consecutive losses >= 3, reduce leverage
	// If consecutive wins >= 3, can increase leverage slightly
	if metrics.ConsecutiveLosses >= 3 {
		// Reduce leverage during drawdown
		po.leverageAdjustment = 0.7 + float64(metrics.MaxConsecutiveLosses-3)*(-0.1) // min 0.5
		po.leverageAdjustment = math.Max(0.5, po.leverageAdjustment)
		logger.Warnf("[Parameter Optimizer] 📉 Drawdown detected (%d consecutive losses), reducing leverage to %.1fx",
			metrics.ConsecutiveLosses, po.leverageAdjustment)
	} else if metrics.ConsecutiveWins >= 3 && metrics.WinRate > 0.55 {
		// Increase leverage during winning streak with high win rate
		po.leverageAdjustment = math.Min(1.3, 1.0+float64(metrics.ConsecutiveWins-3)*0.1)
		logger.Infof("[Parameter Optimizer] 📈 Winning streak detected (%d wins), increasing leverage to %.1fx",
			metrics.ConsecutiveWins, po.leverageAdjustment)
	} else {
		po.leverageAdjustment = 1.0 // Reset to normal
	}

	// ========================================
	// 4. Market Condition Score (0-100)
	// ========================================
	// Based on win rate and profit factor
	if metrics.TradesInPeriod >= 5 {
		// Formula: 50 (base) + (WinRate - 0.5) * 50 + min(ProfitFactor - 1, 0.5) * 10
		po.marketConditionScore = 50 +
			(metrics.WinRate-0.5)*50 +
			math.Min(metrics.ProfitFactor-1, 0.5)*10

		po.marketConditionScore = math.Max(0, math.Min(100, po.marketConditionScore))

		var condition string
		if po.marketConditionScore >= 70 {
			condition = "Strong trend"
		} else if po.marketConditionScore >= 50 {
			condition = "Normal"
		} else {
			condition = "Choppy"
		}
		logger.Infof("[Parameter Optimizer] 📊 Market condition: %s (score: %.0f/100)", condition, po.marketConditionScore)
	}

	logger.Infof("[Parameter Optimizer] 🔧 Parameters optimized - VolMult: %.2f, ConfAdj: %.0f%%, LevMult: %.2f",
		po.volatilityMultiplier, po.confidenceAdjustment, po.leverageAdjustment)
}

// GetAdjustedPositionSize returns position size adjusted by volatility
func (po *ParameterOptimizer) GetAdjustedPositionSize(baseSize float64) float64 {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return baseSize * po.volatilityMultiplier
}

// GetAdjustedConfidenceThreshold returns adjusted confidence threshold
func (po *ParameterOptimizer) GetAdjustedConfidenceThreshold(baseThreshold int) int {
	po.mu.RLock()
	defer po.mu.RUnlock()
	adjusted := float64(baseThreshold) + po.confidenceAdjustment
	adjusted = math.Max(30, math.Min(90, adjusted)) // Keep in valid range [30, 90]
	return int(adjusted)
}

// GetAdjustedLeverage returns adjusted leverage multiplier
func (po *ParameterOptimizer) GetAdjustedLeverage(baseLeverage int) int {
	po.mu.RLock()
	defer po.mu.RUnlock()
	adjusted := float64(baseLeverage) * po.leverageAdjustment
	// Keep leverage within reasonable bounds
	adjusted = math.Max(1, math.Min(20, adjusted))
	return int(adjusted)
}

// GetMarketConditionScore returns the current market condition score (0-100)
func (po *ParameterOptimizer) GetMarketConditionScore() float64 {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return po.marketConditionScore
}

// GetPerformanceMetrics returns a copy of current performance metrics
func (po *ParameterOptimizer) GetPerformanceMetrics() PerformanceMetrics {
	po.mu.RLock()
	defer po.mu.RUnlock()
	return *po.performanceMetrics
}

// String returns a formatted string of optimizer state
func (po *ParameterOptimizer) String() string {
	po.mu.RLock()
	defer po.mu.RUnlock()

	return fmt.Sprintf(
		"[ParameterOptimizer] Volatility: %.2fx | Confidence: %+.0f%% | Leverage: %.2fx | Market: %.0f/100",
		po.volatilityMultiplier,
		po.confidenceAdjustment,
		po.leverageAdjustment,
		po.marketConditionScore,
	)
}
