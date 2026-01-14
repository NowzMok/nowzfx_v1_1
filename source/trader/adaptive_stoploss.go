package trader

import (
	"fmt"
	"math"
	"nofx/logger"
	"sync"
	"time"
)

// AdaptiveStopLossManager handles dynamic stop loss and take profit adjustment
// based on ATR, volatility, and price action
type AdaptiveStopLossManager struct {
	mu sync.RWMutex

	traderID           string
	lookbackPeriod     int     // periods for ATR calculation
	baseATRMultiplier  float64 // base ATR multiplier for stop loss (typically 1.5-2.0)
	profitTargetATR    float64 // ATR multiplier for take profit (typically 3.0-4.0)
	minStopLossPercent float64 // minimum stop loss distance (%)
	maxStopLossPercent float64 // maximum stop loss distance (%)
	trailingEnabled    bool    // enable trailing stop loss
	trailingPercent    float64 // trailing stop loss percentage
	breakEvenLevel     float64 // move stop to breakeven after this profit %

	// Position tracking
	positions map[string]*PositionStopLevel // symbol -> position stop levels
}

// PositionStopLevel tracks stop loss and take profit for a position
type PositionStopLevel struct {
	Symbol             string
	EntryPrice         float64
	StopLoss           float64
	TakeProfit         float64
	TrailingStop       float64
	HighestPrice       float64 // for trailing stop
	EnteredAt          time.Time
	IsBreakeven        bool // has stop been moved to breakeven?
	LastATR            float64
	HighestATR         float64
	SLMoveCount        int     // 止损移动次数
	PeakPrice          float64 // 最高价格
	LowestPrice        float64 // 最低价格
	InitialStopLoss    float64 // 初始止损价格
	FinalStopLoss      float64 // 最终止损价格 (用于关闭时)
	GainedPips         float64 // 获得的点数
	AvoidedLossPercent float64 // 避免的损失百分比
}

// NewAdaptiveStopLossManager creates a new adaptive stop loss manager
func NewAdaptiveStopLossManager(traderID string) *AdaptiveStopLossManager {
	return &AdaptiveStopLossManager{
		traderID:           traderID,
		lookbackPeriod:     14,
		baseATRMultiplier:  1.5,
		profitTargetATR:    3.0,
		minStopLossPercent: 0.5, // minimum 0.5%
		maxStopLossPercent: 5.0, // maximum 5%
		trailingEnabled:    true,
		trailingPercent:    2.0,
		breakEvenLevel:     2.0, // move to breakeven after 2% profit
		positions:          make(map[string]*PositionStopLevel),
	}
}

// SetStopLevelForPosition sets initial stop loss and take profit for a position
func (aslm *AdaptiveStopLossManager) SetStopLevelForPosition(
	symbol string,
	entryPrice float64,
	stopLoss float64,
	takeProfit float64,
	atrValue float64,
) *PositionStopLevel {
	aslm.mu.Lock()
	defer aslm.mu.Unlock()

	level := &PositionStopLevel{
		Symbol:             symbol,
		EntryPrice:         entryPrice,
		StopLoss:           stopLoss,
		TakeProfit:         takeProfit,
		TrailingStop:       stopLoss,
		HighestPrice:       entryPrice,
		LowestPrice:        entryPrice,
		EnteredAt:          time.Now(),
		IsBreakeven:        false,
		LastATR:            atrValue,
		HighestATR:         atrValue,
		SLMoveCount:        0,
		PeakPrice:          entryPrice,
		InitialStopLoss:    stopLoss,
		FinalStopLoss:      stopLoss,
		GainedPips:         0,
		AvoidedLossPercent: 0,
	}

	aslm.positions[symbol] = level

	logger.Infof("[AdaptiveStop] Position %s - Entry: %.6f | SL: %.6f | TP: %.6f | ATR: %.6f",
		symbol, entryPrice, stopLoss, takeProfit, atrValue)

	return level
}

// UpdateATR updates ATR for existing position and recalculates dynamic stops
func (aslm *AdaptiveStopLossManager) UpdateATR(symbol string, atrValue float64, currentPrice float64) *PositionStopLevel {
	aslm.mu.Lock()
	defer aslm.mu.Unlock()

	level, exists := aslm.positions[symbol]
	if !exists {
		return nil
	}

	// Determine if position is long or short based on current price vs entry
	isLong := currentPrice > level.EntryPrice ||
		(currentPrice == level.EntryPrice && level.StopLoss < level.EntryPrice)
	isShort := currentPrice < level.EntryPrice ||
		(currentPrice == level.EntryPrice && level.StopLoss > level.EntryPrice)

	level.LastATR = atrValue
	if atrValue > level.HighestATR {
		level.HighestATR = atrValue
	}

	// Update highest price for trailing stop
	if currentPrice > level.HighestPrice {
		level.HighestPrice = currentPrice
	}

	// ========================================
	// 1. 动态追踪止盈（新增功能）
	// ========================================
	// 当盈利超过阈值后，TP跟随价格移动
	profitPct := 0.0
	if isLong {
		profitPct = (currentPrice - level.EntryPrice) / level.EntryPrice
	} else {
		profitPct = (level.EntryPrice - currentPrice) / level.EntryPrice
	}

	// 盈利超过2%后启用TP追踪
	const profitThreshold = 0.02 // 2%盈利阈值
	const tpTrailingRatio = 0.5  // TP跟随50%的价格移动

	if profitPct > profitThreshold {
		// 计算TP应该跟随的距离
		if isLong {
			// 做多：TP向上跟随
			if currentPrice > level.HighestPrice {
				priceMove := currentPrice - level.HighestPrice
				newTP := level.TakeProfit + (priceMove * tpTrailingRatio)
				if newTP > level.TakeProfit {
					oldTP := level.TakeProfit
					level.TakeProfit = newTP
					logger.Infof("[AdaptiveStop] 🎯 TP trailing up for %s: %.6f -> %.6f (profit: %.2f%%)",
						symbol, oldTP, newTP, profitPct*100)
				}
			}
		} else {
			// 做空：TP向下跟随
			if currentPrice < level.HighestPrice {
				priceMove := level.HighestPrice - currentPrice
				newTP := level.TakeProfit - (priceMove * tpTrailingRatio)
				if newTP < level.TakeProfit {
					oldTP := level.TakeProfit
					level.TakeProfit = newTP
					logger.Infof("[AdaptiveStop] 🎯 TP trailing down for %s: %.6f -> %.6f (profit: %.2f%%)",
						symbol, oldTP, newTP, profitPct*100)
				}
			}
		}
	}

	// ========================================
	// 2. Keep original dynamic stop loss logic
	// ========================================
	// Calculate elapsed time since position entered
	elapsedTime := time.Since(level.EnteredAt)
	elapsedSeconds := int(elapsedTime.Seconds())

	// 🔥 新增：根据波动性动态调整时间窗口
	// 计算当前波动率（ATR / Price）
	volatility := 0.0
	if currentPrice > 0 {
		volatility = atrValue / currentPrice
	}

	// 根据波动性确定目标时间（秒）
	var targetSeconds float64
	if volatility > 0.03 {
		// 高波动 >3%: 延长至10分钟，避免在正常波动中被止损
		targetSeconds = 600.0
		logger.Debugf("[AdaptiveStop] High volatility (%.2f%%), using 10min window for %s",
			volatility*100, symbol)
	} else if volatility < 0.01 {
		// 低波动 <1%: 缩短至3分钟，快速保护利润
		targetSeconds = 180.0
		logger.Debugf("[AdaptiveStop] Low volatility (%.2f%%), using 3min window for %s",
			volatility*100, symbol)
	} else {
		// 中等波动 1%-3%: 默认5分钟
		targetSeconds = 300.0
	}

	// Calculate time-based progression (0 to 1 over target time window)
	timeProgression := float64(elapsedSeconds) / targetSeconds
	if timeProgression > 1.0 {
		timeProgression = 1.0
	}

	// Calculate the target stop loss (which will eventually reach entry price)
	// For long positions: stop loss moves up towards entry price
	// For short positions: stop loss moves down towards entry price

	// isLong and isShort are already determined at the beginning of the function

	// Calculate new stop loss based on time progression
	var newStopLoss float64
	if isLong {
		// For long: SL moves from original (higher) value towards entry price
		distanceToEntry := level.StopLoss - level.EntryPrice
		newStopLoss = level.StopLoss - (distanceToEntry * timeProgression)
	} else if isShort {
		// For short: SL moves from original (lower) value towards entry price
		distanceToEntry := level.EntryPrice - level.StopLoss
		newStopLoss = level.StopLoss + (distanceToEntry * timeProgression)
	} else {
		// Unknown position direction, don't update SL
		newStopLoss = level.StopLoss
	}

	// Only update if stop loss would decrease (more conservative)
	if newStopLoss < level.StopLoss {
		level.StopLoss = newStopLoss
		level.SLMoveCount++ // 增加止损移动计数
	}

	// Log the update every 10 seconds (when elapsedSeconds % 10 == 0)
	if elapsedSeconds > 0 && elapsedSeconds%10 == 0 {
		logger.Infof("[AdaptiveStop] %s SL updated after %ds: %.6f (progress: %.1f%%, volatility: %.2f%%)",
			symbol, elapsedSeconds, level.StopLoss, timeProgression*100, volatility*100)
	}

	// After target time window, ensure stop loss equals entry price (if not already there)
	if float64(elapsedSeconds) >= targetSeconds {
		if isLong && level.StopLoss < level.EntryPrice {
			level.StopLoss = level.EntryPrice
			logger.Infof("[AdaptiveStop] %s SL reached entry price after %.1fmin: %.6f",
				symbol, targetSeconds/60, level.EntryPrice)
		} else if isShort && level.StopLoss > level.EntryPrice {
			level.StopLoss = level.EntryPrice
			logger.Infof("[AdaptiveStop] %s SL reached entry price after %.1fmin: %.6f",
				symbol, targetSeconds/60, level.EntryPrice)
		}
	}

	return level
}

// calculateTakeProfitATR calculates take profit based on ATR
func (aslm *AdaptiveStopLossManager) calculateTakeProfitATR(level *PositionStopLevel, atrValue float64) float64 {
	// Determine if long or short based on entry vs current direction
	// Simplified: assume long position for now
	return level.EntryPrice + (atrValue * aslm.profitTargetATR)
}

// calculateTrailingStop calculates trailing stop loss
func (aslm *AdaptiveStopLossManager) calculateTrailingStop(level *PositionStopLevel, currentPrice float64) float64 {
	// Trailing stop = current price - (current price × trailing %)
	trailingDistance := currentPrice * (aslm.trailingPercent / 100.0)
	return currentPrice - trailingDistance
}

// GetCurrentStopLoss returns the current stop loss for a position
func (aslm *AdaptiveStopLossManager) GetCurrentStopLoss(symbol string) (float64, bool) {
	aslm.mu.RLock()
	defer aslm.mu.RUnlock()

	level, exists := aslm.positions[symbol]
	if !exists {
		return 0, false
	}

	// Return the more conservative (higher) stop loss
	if level.TrailingStop > level.StopLoss {
		return level.TrailingStop, true
	}
	return level.StopLoss, true
}

// GetCurrentTakeProfit returns the current take profit for a position
func (aslm *AdaptiveStopLossManager) GetCurrentTakeProfit(symbol string) (float64, bool) {
	aslm.mu.RLock()
	defer aslm.mu.RUnlock()

	level, exists := aslm.positions[symbol]
	if !exists {
		return 0, false
	}

	return level.TakeProfit, true
}

// ClosePosition removes a position from tracking
func (aslm *AdaptiveStopLossManager) ClosePosition(symbol string) *PositionStopLevel {
	aslm.mu.Lock()
	defer aslm.mu.Unlock()

	level := aslm.positions[symbol]
	delete(aslm.positions, symbol)

	if level != nil {
		duration := time.Since(level.EnteredAt)
		logger.Infof("[AdaptiveStop] Closed %s after %.0f minutes | Final SL: %.6f | Final TP: %.6f",
			symbol, duration.Minutes(), level.StopLoss, level.TakeProfit)

		// 记录性能指标到动态止损监控系统
		// （需要导入 api 包并调用 RecordTrailingStopMetric）
		recordTrailingStopMetric(aslm.traderID, symbol, level, duration)
	}

	return level
}

// recordTrailingStopMetric records the trailing stop performance metrics
func recordTrailingStopMetric(traderID, symbol string, level *PositionStopLevel, duration time.Duration) {
	// 避免循环导入，这里通过延迟执行来在主 goroutine 中调用
	// 该函数由专门的性能收集包提供
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("[AdaptiveStop] Failed to record performance metric: %v", r)
			}
		}()

		// 导入时需要在 adaptive_stoploss.go 顶部添加 import "nofx/api"
		// api.RecordTrailingStopMetric(api.TrailingStopPerformanceMetric{
		// 	Symbol:    symbol,
		// 	TraderID:  traderID,
		// 	EntryPrice: level.EntryPrice,
		// 	PeakPrice: level.HighestPrice,
		// 	LowestPrice: level.LowestPrice,
		// 	InitialSL: level.InitialStopLoss,
		// 	FinalSL:   level.StopLoss,
		// 	SLMoveCount: level.SLMoveCount,
		// 	Duration:  int(duration.Seconds()),
		// })
	}()
}

// AdjustATRMultipliers adjusts the ATR multipliers based on market conditions
func (aslm *AdaptiveStopLossManager) AdjustATRMultipliers(volatilityScore float64) {
	aslm.mu.Lock()
	defer aslm.mu.Unlock()

	// volatilityScore: 0-1 where 1 is highest volatility
	// In high volatility: increase multipliers for wider stops
	// In low volatility: decrease multipliers for tighter stops

	if volatilityScore > 0.7 {
		// High volatility: use wider stops
		aslm.baseATRMultiplier = 2.0
		aslm.profitTargetATR = 4.0
		logger.Infof("[AdaptiveStop] High volatility detected, widening stops - Base: 2.0x, TP: 4.0x")
	} else if volatilityScore < 0.3 {
		// Low volatility: use tighter stops
		aslm.baseATRMultiplier = 1.0
		aslm.profitTargetATR = 2.0
		logger.Infof("[AdaptiveStop] Low volatility detected, tightening stops - Base: 1.0x, TP: 2.0x")
	} else {
		// Normal volatility
		aslm.baseATRMultiplier = 1.5
		aslm.profitTargetATR = 3.0
		logger.Infof("[AdaptiveStop] Normal volatility, using standard stops - Base: 1.5x, TP: 3.0x")
	}
}

// ValidateStopLossDistance validates that stop loss meets minimum distance requirements
func (aslm *AdaptiveStopLossManager) ValidateStopLossDistance(
	entryPrice float64,
	stopLoss float64,
	isBuy bool,
) (valid bool, reason string) {
	var distance float64
	if isBuy {
		distance = ((entryPrice - stopLoss) / entryPrice) * 100
	} else {
		distance = ((stopLoss - entryPrice) / entryPrice) * 100
	}

	distance = math.Abs(distance)

	if distance < aslm.minStopLossPercent {
		return false, fmt.Sprintf("Stop loss too close (%.2f%% < %.2f%% min)",
			distance, aslm.minStopLossPercent)
	}

	if distance > aslm.maxStopLossPercent {
		return false, fmt.Sprintf("Stop loss too wide (%.2f%% > %.2f%% max)",
			distance, aslm.maxStopLossPercent)
	}

	return true, ""
}

// ScaleOutPartialProfit implements partial take profit at multiple levels
// Returns updated take profit level
func (aslm *AdaptiveStopLossManager) ScaleOutPartialProfit(
	symbol string,
	profitTarget float64, // profit % to take partial
	profitPercent float64, // percentage of position to close
) (float64, bool) {
	aslm.mu.RLock()
	defer aslm.mu.RUnlock()

	level, exists := aslm.positions[symbol]
	if !exists {
		return 0, false
	}

	// Calculate profit % at current level
	currentProfit := ((level.HighestPrice - level.EntryPrice) / level.EntryPrice) * 100

	if currentProfit >= profitTarget {
		// Scale out - move take profit higher
		newTP := level.TakeProfit + (level.TakeProfit * profitPercent)
		logger.Infof("[AdaptiveStop] %s scaling out %.0f%% of position at %.2f%% profit, new TP: %.6f",
			symbol, profitPercent*100, currentProfit, newTP)
		return newTP, true
	}

	return level.TakeProfit, false
}

// GetPositionStatus returns formatted status of a position
func (aslm *AdaptiveStopLossManager) GetPositionStatus(symbol string) string {
	aslm.mu.RLock()
	defer aslm.mu.RUnlock()

	level, exists := aslm.positions[symbol]
	if !exists {
		return fmt.Sprintf("%s: No position tracked", symbol)
	}

	age := time.Since(level.EnteredAt)
	return fmt.Sprintf("%s: Entry=%.6f | SL=%.6f | TP=%.6f | Highest=%.6f | Age=%.0fm",
		symbol, level.EntryPrice, level.StopLoss, level.TakeProfit, level.HighestPrice, age.Minutes())
}

// String returns formatted adaptive stop loss manager state
func (aslm *AdaptiveStopLossManager) String() string {
	aslm.mu.RLock()
	defer aslm.mu.RUnlock()

	return fmt.Sprintf(
		"[AdaptiveStop] Tracking: %d positions | Base ATR: %.1fx | TP ATR: %.1fx | Trailing: %v (%.1f%%)",
		len(aslm.positions),
		aslm.baseATRMultiplier,
		aslm.profitTargetATR,
		aslm.trailingEnabled,
		aslm.trailingPercent,
	)
}
