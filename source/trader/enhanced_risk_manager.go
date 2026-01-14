package trader

import (
	"fmt"
	"math"
	"nofx/logger"
	"nofx/store"
	"sync"
	"time"
)

// EnhancedRiskManager handles advanced risk management including drawdown control, Kelly criterion, and position sizing
type EnhancedRiskManager struct {
	mu sync.RWMutex

	// Account-level constraints
	traderID             string
	accountEquity        float64
	maxDailyLoss         float64       // maximum loss allowed per day (percentage)
	maxDrawdownAllowed   float64       // maximum drawdown allowed (percentage)
	maxConsecutiveLosses int           // max consecutive losses before stopping
	drawdownStopDuration time.Duration // how long to pause trading after hitting drawdown
	stopUntil            time.Time

	// Per-position constraints
	minRiskRewardRatio float64 // minimum risk/reward ratio for opening positions
	atrLookbackPeriod  int     // periods to calculate ATR for position sizing

	// Tracking
	dailyStartEquity    float64
	dailyMaxEquity      float64
	dailyLowestEquity   float64
	currentDrawdown     float64
	consecutiveStopLoss int
	dailyLosses         []float64
	lastResetDate       time.Time // Track the last reset date for daily reset logic

	// Store for persistence
	store store.Store
}

// NewEnhancedRiskManager creates a new enhanced risk manager
func NewEnhancedRiskManager(traderID string, st store.Store) *EnhancedRiskManager {
	now := time.Now()
	return &EnhancedRiskManager{
		traderID:             traderID,
		maxDailyLoss:         0.05,          // 5% max daily loss
		maxDrawdownAllowed:   0.20,          // 20% max drawdown
		maxConsecutiveLosses: 5,             // stop after 5 consecutive losses
		drawdownStopDuration: 4 * time.Hour, // pause 4 hours after drawdown hit
		minRiskRewardRatio:   1.5,           // require 1.5:1 risk/reward
		atrLookbackPeriod:    14,            // 14 periods for ATR
		dailyStartEquity:     0,
		dailyMaxEquity:       0,
		dailyLowestEquity:    math.MaxFloat64,
		stopUntil:            now,
		dailyLosses:          make([]float64, 0),
	}
}

// UpdateEquity updates the current account equity and checks risk limits
func (rm *EnhancedRiskManager) UpdateEquity(currentEquity float64) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Get today's date in UTC (truncated to day) - ensures consistent reset across timezones
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Initialize daily tracking on first update or new day
	if rm.dailyStartEquity == 0 || !rm.lastResetDate.Equal(today) {
		rm.dailyStartEquity = currentEquity
		rm.dailyMaxEquity = currentEquity
		rm.dailyLowestEquity = currentEquity
		rm.dailyLosses = make([]float64, 0)
		rm.lastResetDate = today
		logger.Infof("[EnhancedRiskManager] 📅 Daily metrics reset for %s (UTC), starting equity: %.2f", today.Format("2006-01-02"), currentEquity)
	}

	// Update daily tracking
	if currentEquity > rm.dailyMaxEquity {
		rm.dailyMaxEquity = currentEquity
	}
	if currentEquity < rm.dailyLowestEquity {
		rm.dailyLowestEquity = currentEquity
	}

	// Calculate drawdown
	if rm.dailyMaxEquity > 0 {
		rm.currentDrawdown = (rm.dailyMaxEquity - currentEquity) / rm.dailyMaxEquity
	}

	// Check if drawdown limit exceeded
	if rm.currentDrawdown > rm.maxDrawdownAllowed && time.Now().Before(rm.stopUntil) {
		rm.stopUntil = time.Now().Add(rm.drawdownStopDuration)
		logger.Warnf("[EnhancedRiskManager] ⚠️ Maximum drawdown (%.2f%%) exceeded limit (%.2f%%), trading paused until %s",
			rm.currentDrawdown*100, rm.maxDrawdownAllowed*100, rm.stopUntil.Format("15:04:05"))
	}

	// Check daily loss limit
	dailyLoss := (rm.dailyStartEquity - currentEquity) / rm.dailyStartEquity
	if dailyLoss > rm.maxDailyLoss {
		rm.stopUntil = time.Now().Add(rm.drawdownStopDuration)
		logger.Warnf("[EnhancedRiskManager] ⚠️ Daily loss limit (%.2f%%) exceeded (%.2f%%), trading paused",
			dailyLoss*100, rm.maxDailyLoss*100)
	}
}

// CheckRiskLimits checks if trading should be allowed based on current risk state
func (rm *EnhancedRiskManager) CheckRiskLimits() (allowed bool, reason string) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if time.Now().Before(rm.stopUntil) {
		remaining := rm.stopUntil.Sub(time.Now())
		return false, fmt.Sprintf("Trading paused due to risk control, %.0f minutes remaining", remaining.Minutes())
	}

	return true, ""
}

// CalculatePositionSize calculates position size using Kelly Criterion with ATR-based stops
// volatility: current ATR or volatility measure
// winRate: historical win rate (0-1)
// avgWin: average winning trade size
// avgLoss: average losing trade size
// accountEquity: current account equity
func (rm *EnhancedRiskManager) CalculatePositionSize(
	volatility float64,
	winRate float64,
	avgWin float64,
	avgLoss float64,
	accountEquity float64,
	basePositionSize float64,
) float64 {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// ========================================
	// Method 1: Kelly Criterion Adjustment
	// ========================================
	// Kelly % = (winRate * avgWin - (1-winRate) * avgLoss) / avgWin
	// Apply fractional Kelly (f = 0.25) for safety
	fractionalKelly := 0.25

	var kellyRatio float64
	if avgWin > 0 && winRate > 0 && winRate < 1 {
		kellyCalc := (winRate*avgWin - (1-winRate)*avgLoss) / avgWin
		kellyRatio = kellyCalc * fractionalKelly              // Use fractional Kelly
		kellyRatio = math.Max(0.5, math.Min(2.0, kellyRatio)) // Keep between 0.5x and 2x
	} else {
		kellyRatio = 1.0
	}

	// ========================================
	// Method 2: Volatility-Based Adjustment
	// ========================================
	// Lower volatility = larger position, higher volatility = smaller position
	volatilityAdjustment := 1.0
	if volatility > 0 {
		// Inverse relationship: as volatility increases, position size decreases
		volatilityAdjustment = 1.0 / (1.0 + volatility*0.5)
		volatilityAdjustment = math.Max(0.5, math.Min(1.5, volatilityAdjustment))
	}

	// ========================================
	// Method 3: Equity-Based Position Sizing
	// ========================================
	// Risk only a fixed percentage of equity per trade (typically 1-2%)
	riskPercentagePerTrade := 0.02 // 2% of equity
	maxPositionFromEquity := accountEquity * riskPercentagePerTrade

	// ========================================
	// Combine all methods
	// ========================================
	adjustedSize := basePositionSize *
		kellyRatio *
		volatilityAdjustment *
		(maxPositionFromEquity / basePositionSize) // Ensure we don't exceed equity-based max

	// Apply final safety constraints
	adjustedSize = math.Max(basePositionSize*0.5, math.Min(maxPositionFromEquity, adjustedSize))

	logger.Debugf("[EnhancedRiskManager] Position sizing - Base: %.2f | Kelly: %.2fx | Vol: %.2fx | Final: %.2f USDT",
		basePositionSize, kellyRatio, volatilityAdjustment, adjustedSize)

	return adjustedSize
}

// ValidateStopLoss validates that stop loss is appropriate based on risk/reward ratio
func (rm *EnhancedRiskManager) ValidateStopLoss(
	entryPrice float64,
	stopLoss float64,
	takeProfit float64,
	isBuy bool,
) (valid bool, reason string) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if entryPrice <= 0 || stopLoss <= 0 || takeProfit <= 0 {
		return false, "Invalid price parameters"
	}

	var risk, reward float64
	if isBuy {
		risk = (entryPrice - stopLoss) / entryPrice
		reward = (takeProfit - entryPrice) / entryPrice
	} else {
		risk = (stopLoss - entryPrice) / entryPrice
		reward = (entryPrice - takeProfit) / entryPrice
	}

	if risk <= 0 || reward <= 0 {
		return false, fmt.Sprintf("Invalid stop loss or take profit (entry: %.6f, SL: %.6f, TP: %.6f)",
			entryPrice, stopLoss, takeProfit)
	}

	riskRewardRatio := reward / risk
	if riskRewardRatio < rm.minRiskRewardRatio {
		return false, fmt.Sprintf("Risk/Reward ratio (%.2f) below minimum (%.2f)",
			riskRewardRatio, rm.minRiskRewardRatio)
	}

	return true, ""
}

// RecordLosingTrade records a losing trade for drawdown tracking
func (rm *EnhancedRiskManager) RecordLosingTrade(loss float64) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.consecutiveStopLoss++
	rm.dailyLosses = append(rm.dailyLosses, loss)

	if rm.consecutiveStopLoss >= rm.maxConsecutiveLosses {
		rm.stopUntil = time.Now().Add(rm.drawdownStopDuration)
		logger.Warnf("[EnhancedRiskManager] ⚠️ Max consecutive losses (%d) reached, trading paused",
			rm.maxConsecutiveLosses)
	}
}

// RecordWinningTrade resets consecutive stop loss counter
func (rm *EnhancedRiskManager) RecordWinningTrade() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.consecutiveStopLoss = 0
}

// GetCurrentDrawdown returns the current drawdown percentage
func (rm *EnhancedRiskManager) GetCurrentDrawdown() float64 {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.currentDrawdown
}

// GetDrawdownStatus returns formatted drawdown status
func (rm *EnhancedRiskManager) GetDrawdownStatus() string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return fmt.Sprintf("Drawdown: %.2f%%/%.2f%% | Daily Losses: %d | Consecutive SL: %d/%d",
		rm.currentDrawdown*100,
		rm.maxDrawdownAllowed*100,
		len(rm.dailyLosses),
		rm.consecutiveStopLoss,
		rm.maxConsecutiveLosses)
}

// CalculateATRBasedStopLoss calculates stop loss based on ATR
func (rm *EnhancedRiskManager) CalculateATRBasedStopLoss(
	currentPrice float64,
	atrValue float64,
	stopLossMultiplier float64, // typically 1.5-2.0
	isBuy bool,
) float64 {
	if atrValue <= 0 {
		// Fallback: use fixed percentage
		stopLossPercent := 0.02 // 2%
		if isBuy {
			return currentPrice * (1 - stopLossPercent)
		} else {
			return currentPrice * (1 + stopLossPercent)
		}
	}

	adjustedATR := atrValue * stopLossMultiplier
	if isBuy {
		return currentPrice - adjustedATR
	} else {
		return currentPrice + adjustedATR
	}
}

// SetDailyLossLimit sets the maximum daily loss percentage
func (rm *EnhancedRiskManager) SetDailyLossLimit(percentage float64) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.maxDailyLoss = percentage
}

// SetDrawdownLimit sets the maximum drawdown percentage
func (rm *EnhancedRiskManager) SetDrawdownLimit(percentage float64) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.maxDrawdownAllowed = percentage
}

// String returns a formatted string of risk manager state
func (rm *EnhancedRiskManager) String() string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return fmt.Sprintf(
		"[EnhancedRiskManager] %s | Daily: %.2f%%/%.2f%% | Consecutive SL: %d/%d",
		rm.GetDrawdownStatus(),
		(rm.dailyStartEquity-rm.dailyLowestEquity)/rm.dailyStartEquity*100,
		rm.maxDailyLoss*100,
		rm.consecutiveStopLoss,
		rm.maxConsecutiveLosses,
	)
}
