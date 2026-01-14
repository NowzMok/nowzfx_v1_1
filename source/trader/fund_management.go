package trader

import (
	"fmt"
	"math"
	"nofx/logger"
	"sync"
)

// FundManagementSystem implements advanced capital allocation and position sizing strategies
// Including Kelly Criterion, Fixed Fraction, and Dynamic Allocation
type FundManagementSystem struct {
	mu sync.RWMutex

	// Account configuration
	initialBalance float64
	currentBalance float64
	accountEquity  float64
	allocatedFunds map[string]float64 // strategy/position -> allocated amount

	// Risk per trade configuration
	riskPercentage float64 // percentage of equity to risk per trade (e.g., 0.02 = 2%)
	maxAllocation  float64 // maximum percentage of equity for single position
	minAllocation  float64 // minimum percentage of equity per allocation

	// Allocation method
	allocationMethod string  // "kelly", "fixed_fraction", "dynamic"
	kellyFraction    float64 // fractional kelly (typically 0.25 for safety)

	// Performance tracking
	totalTrades   int
	winningTrades int
	losingTrades  int
	largestWin    float64
	largestLoss   float64
	averageWin    float64
	averageLoss   float64

	// Rebalance configuration
	rebalanceThreshold float64 // rebalance when allocation drifts by this percentage
	lastRebalanceTime  int64   // unix timestamp
}

// NewFundManagementSystem creates a new fund management system
func NewFundManagementSystem(initialBalance float64) *FundManagementSystem {
	return &FundManagementSystem{
		initialBalance:     initialBalance,
		currentBalance:     initialBalance,
		accountEquity:      initialBalance,
		allocatedFunds:     make(map[string]float64),
		riskPercentage:     0.02,    // 2% risk per trade
		maxAllocation:      0.30,    // max 30% in single position
		minAllocation:      0.01,    // min 1% allocation
		allocationMethod:   "kelly", // default to Kelly Criterion
		kellyFraction:      0.25,    // use 25% of full Kelly
		rebalanceThreshold: 0.20,    // rebalance if 20%+ drift
	}
}

// CalculateRiskAmount calculates the amount to risk on the next trade
// stopLossPips: distance to stop loss in pips/points
// lotSize: lot/position size in units
func (fms *FundManagementSystem) CalculateRiskAmount(
	stopLossPips float64,
	currentPrice float64,
) float64 {
	fms.mu.RLock()
	defer fms.mu.RUnlock()

	if currentPrice <= 0 || stopLossPips <= 0 {
		return fms.accountEquity * fms.riskPercentage
	}

	// Risk = Equity × Risk% / (Stop Loss % / Current Price)
	stopLossPercentage := (stopLossPips / currentPrice)
	_ = stopLossPercentage // Mark as used for future expansion
	riskAmount := fms.accountEquity * fms.riskPercentage

	// Ensure we don't exceed max allocation
	maxRisk := fms.accountEquity * fms.maxAllocation
	if riskAmount > maxRisk {
		return maxRisk
	}

	return riskAmount
}

// CalculatePositionSizeWithKelly calculates optimal position size using Kelly Criterion
// winRate: historical win rate (0-1)
// avgWin: average winning trade profit
// avgLoss: average losing trade loss
// entryPrice: current entry price
// stopLoss: stop loss price
func (fms *FundManagementSystem) CalculatePositionSizeWithKelly(
	winRate float64,
	avgWin float64,
	avgLoss float64,
	entryPrice float64,
	stopLoss float64,
) float64 {
	fms.mu.RLock()
	defer fms.mu.RUnlock()

	// Validate inputs
	if winRate <= 0 || winRate >= 1 || avgWin <= 0 || avgLoss <= 0 || entryPrice <= 0 {
		// Fallback to fixed percentage
		return fms.accountEquity * fms.riskPercentage
	}

	// Kelly Formula: f* = (bp - q) / b
	// where: b = odds ratio, p = win probability, q = loss probability
	// Simplified: f* = (win% × avg_win - loss% × avg_loss) / avg_win

	lossPercent := 1 - winRate

	// Calculate raw kelly fraction
	kellySuggested := ((winRate * avgWin) - (lossPercent * avgLoss)) / avgWin

	// Handle negative Kelly (indicates unfavorable odds - should not trade)
	if kellySuggested <= 0 {
		logger.Warnf("[FundManagement] ⚠️ Negative Kelly criterion (%.4f) - unfavorable odds, using minimum position", kellySuggested)
		return fms.accountEquity * fms.minAllocation
	}

	// Apply fractional kelly for safety (e.g., 25% of full kelly)
	kellyAdjusted := kellySuggested * fms.kellyFraction

	// Ensure it's within bounds (0.5x to 2x of base position)
	kellyAdjusted = math.Max(0.5, math.Min(2.0, kellyAdjusted))

	// Calculate position size
	basePositionSize := fms.accountEquity * fms.riskPercentage
	positionSize := basePositionSize * kellyAdjusted

	// Calculate risk distance
	riskDistance := math.Abs(entryPrice - stopLoss)

	// If risk distance is too small, scale down position
	if riskDistance > 0 {
		maxPositionFromRisk := (fms.accountEquity * fms.riskPercentage) / riskDistance
		positionSize = math.Min(positionSize, maxPositionFromRisk)
	}

	// Apply allocation limits
	maxPosition := fms.accountEquity * fms.maxAllocation
	positionSize = math.Min(positionSize, maxPosition)

	minPosition := fms.accountEquity * fms.minAllocation
	positionSize = math.Max(positionSize, minPosition)

	return positionSize
}

// CalculatePositionSizeWithFixedFraction calculates position using fixed fraction method
// riskFraction: fraction of account to risk (e.g., 0.02 for 2%)
// entryPrice: entry price
// stopLoss: stop loss price
func (fms *FundManagementSystem) CalculatePositionSizeWithFixedFraction(
	riskFraction float64,
	entryPrice float64,
	stopLoss float64,
) float64 {
	fms.mu.RLock()
	defer fms.mu.RUnlock()

	if entryPrice <= 0 || stopLoss <= 0 {
		return fms.accountEquity * riskFraction
	}

	riskAmount := fms.accountEquity * riskFraction
	riskDistance := math.Abs(entryPrice - stopLoss)

	if riskDistance <= 0 {
		return riskAmount / entryPrice
	}

	positionSize := riskAmount / riskDistance

	// Apply limits
	maxPosition := fms.accountEquity * fms.maxAllocation
	positionSize = math.Min(positionSize, maxPosition)

	return positionSize
}

// CalculateDynamicAllocation allocates capital based on market opportunities
// confidence: confidence score (0-100)
// volatility: current market volatility
// currentExposure: current total position exposure
func (fms *FundManagementSystem) CalculateDynamicAllocation(
	confidence int,
	volatility float64,
	currentExposure float64,
) float64 {
	fms.mu.RLock()
	defer fms.mu.RUnlock()

	// Base allocation from risk percentage
	baseAllocation := fms.accountEquity * fms.riskPercentage

	// Confidence multiplier: higher confidence = larger position
	confidenceMultiplier := float64(confidence) / 100.0

	// Volatility adjustment: lower volatility = larger position
	volatilityAdjustment := 1.0
	if volatility > 0 {
		volatilityAdjustment = 1.0 / (1.0 + volatility*0.5)
		volatilityAdjustment = math.Max(0.5, math.Min(1.5, volatilityAdjustment))
	}

	// Calculate available allocation (remaining equity not exposed)
	maxExposure := fms.accountEquity * (1 - fms.maxAllocation)
	availableAllocation := maxExposure - currentExposure
	if availableAllocation < 0 {
		return 0
	}

	// Combine all factors
	allocation := baseAllocation * confidenceMultiplier * volatilityAdjustment

	// Cap to available space
	allocation = math.Min(allocation, availableAllocation)

	logger.Debugf("[FundMgmt] Allocation - Base: %.2f | Conf: %.1fx | Vol: %.2fx | Available: %.2f | Final: %.2f",
		baseAllocation, confidenceMultiplier, volatilityAdjustment, availableAllocation, allocation)

	return allocation
}

// AllocateFunds allocates funds to a specific position/strategy
func (fms *FundManagementSystem) AllocateFunds(identifier string, amount float64) error {
	fms.mu.Lock()
	defer fms.mu.Unlock()

	// Check if we have enough available funds
	totalAllocated := 0.0
	for _, allocated := range fms.allocatedFunds {
		totalAllocated += allocated
	}

	available := fms.accountEquity - totalAllocated
	if amount > available {
		return fmt.Errorf("insufficient funds: requested %.2f, available %.2f", amount, available)
	}

	fms.allocatedFunds[identifier] = amount
	logger.Infof("[FundMgmt] Allocated %.2f USDT to %s", amount, identifier)
	return nil
}

// DeallocateFunds deallocates funds from a position/strategy
func (fms *FundManagementSystem) DeallocateFunds(identifier string) float64 {
	fms.mu.Lock()
	defer fms.mu.Unlock()

	amount := fms.allocatedFunds[identifier]
	delete(fms.allocatedFunds, identifier)
	logger.Infof("[FundMgmt] Deallocated %.2f USDT from %s", amount, identifier)
	return amount
}

// UpdateAccountEquity updates the account equity after P&L
func (fms *FundManagementSystem) UpdateAccountEquity(newEquity float64) {
	fms.mu.Lock()
	defer fms.mu.Unlock()

	fms.accountEquity = newEquity
	fms.currentBalance = newEquity
}

// RecordTrade records a trade result for performance metrics
func (fms *FundManagementSystem) RecordTrade(pnl float64) {
	fms.mu.Lock()
	defer fms.mu.Unlock()

	fms.totalTrades++

	if pnl > 0 {
		fms.winningTrades++
		fms.largestWin = math.Max(fms.largestWin, pnl)
		fms.averageWin = (fms.averageWin*float64(fms.winningTrades-1) + pnl) / float64(fms.winningTrades)
	} else if pnl < 0 {
		fms.losingTrades++
		fms.largestLoss = math.Max(fms.largestLoss, math.Abs(pnl))
		fms.averageLoss = (fms.averageLoss*float64(fms.losingTrades-1) + math.Abs(pnl)) / float64(fms.losingTrades)
	}

	logger.Debugf("[FundMgmt] Trade recorded - Win Rate: %.2f%%, Avg Win: %.2f, Avg Loss: %.2f",
		float64(fms.winningTrades)/float64(fms.totalTrades)*100, fms.averageWin, fms.averageLoss)
}

// GetWinRate returns the current win rate
func (fms *FundManagementSystem) GetWinRate() float64 {
	fms.mu.RLock()
	defer fms.mu.RUnlock()

	if fms.totalTrades == 0 {
		return 0
	}
	return float64(fms.winningTrades) / float64(fms.totalTrades)
}

// GetAverageWin returns the average winning trade
func (fms *FundManagementSystem) GetAverageWin() float64 {
	fms.mu.RLock()
	defer fms.mu.RUnlock()
	return fms.averageWin
}

// GetAverageLoss returns the average losing trade
func (fms *FundManagementSystem) GetAverageLoss() float64 {
	fms.mu.RLock()
	defer fms.mu.RUnlock()
	return fms.averageLoss
}

// GetAllocationReport returns a formatted report of current allocations
func (fms *FundManagementSystem) GetAllocationReport() string {
	fms.mu.RLock()
	defer fms.mu.RUnlock()

	totalAllocated := 0.0
	for _, amount := range fms.allocatedFunds {
		totalAllocated += amount
	}

	utilizationPercent := (totalAllocated / fms.accountEquity) * 100
	return fmt.Sprintf("[FundMgmt] Allocation: %.2f/%.2f USDT (%.1f%%) | Equity: %.2f | Win Rate: %.2f%%",
		totalAllocated, fms.accountEquity, utilizationPercent, fms.accountEquity,
		fms.GetWinRate()*100)
}

// String returns formatted fund management system state
func (fms *FundManagementSystem) String() string {
	fms.mu.RLock()
	defer fms.mu.RUnlock()

	return fmt.Sprintf(
		"[FundMgmt] Method: %s | Risk: %.2f%% | MaxAlloc: %.2f%% | Equity: %.2f | Trades: %d",
		fms.allocationMethod,
		fms.riskPercentage*100,
		fms.maxAllocation*100,
		fms.accountEquity,
		fms.totalTrades,
	)
}
