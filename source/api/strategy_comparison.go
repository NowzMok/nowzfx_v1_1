package api

import (
	"fmt"
	"net/http"
	"time"

	"nofx/logger"
	"nofx/store"

	"github.com/gin-gonic/gin"
)

// StrategyComparisonMetrics 策略对比指标
type StrategyComparisonMetrics struct {
	StrategyName      string  `json:"strategy_name"`
	TraderID          string  `json:"trader_id"`
	TotalTrades       int     `json:"total_trades"`
	WinningTrades     int     `json:"winning_trades"`
	LosingTrades      int     `json:"losing_trades"`
	WinRate           float64 `json:"win_rate"`
	TotalProfit       float64 `json:"total_profit"`
	TotalLoss         float64 `json:"total_loss"`
	NetProfit         float64 `json:"net_profit"`
	ProfitFactor      float64 `json:"profit_factor"`
	MaxDrawdown       float64 `json:"max_drawdown"`
	SharpeRatio       float64 `json:"sharpe_ratio"`
	AverageTrade      float64 `json:"average_trade"`
	AverageWin        float64 `json:"average_win"`
	AverageLoss       float64 `json:"average_loss"`
	LargestWin        float64 `json:"largest_win"`
	LargestLoss       float64 `json:"largest_loss"`
	TradeDuration     float64 `json:"trade_duration"` // 平均持仓时间（小时）
	StartDate         string  `json:"start_date"`
	EndDate           string  `json:"end_date"`
	DaysActive        int     `json:"days_active"`
	ReturnRate        float64 `json:"return_rate"`       // 总收益率%
	AnnualizedReturn  float64 `json:"annualized_return"` // 年化收益率%
	ConsecutiveWins   int     `json:"consecutive_wins"`
	ConsecutiveLosses int     `json:"consecutive_losses"`
}

// StrategyPerformanceTrend 策略性能趋势
type StrategyPerformanceTrend struct {
	Date          string  `json:"date"`
	CumulativeROI float64 `json:"cumulative_roi"` // 累计收益率%
	DailyReturn   float64 `json:"daily_return"`   // 日收益
	TradeCount    int     `json:"trade_count"`
	WinRate       float64 `json:"win_rate"`
}

// handleStrategyComparison 处理策略对比请求
func (s *Server) handleStrategyComparison(c *gin.Context) {
	traderIDs := c.QueryArray("trader_ids")
	if len(traderIDs) == 0 {
		SafeBadRequest(c, "trader_ids is required")
		return
	}

	if len(traderIDs) > 5 {
		SafeBadRequest(c, "Maximum 5 strategies can be compared at once")
		return
	}

	var comparisons []StrategyComparisonMetrics
	for _, traderID := range traderIDs {
		metrics, err := s.calculateStrategyMetrics(traderID)
		if err != nil {
			logger.Warnf("Failed to calculate metrics for trader %s: %v", traderID, err)
			continue
		}
		comparisons = append(comparisons, *metrics)
	}

	if len(comparisons) == 0 {
		SafeInternalError(c, "Calculate metrics", fmt.Errorf("no valid strategies found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"comparisons": comparisons,
		"count":       len(comparisons),
	})
}

// handleStrategyPerformanceTrend 获取策略性能趋势对比
func (s *Server) handleStrategyPerformanceTrend(c *gin.Context) {
	traderIDs := c.QueryArray("trader_ids")
	if len(traderIDs) == 0 {
		SafeBadRequest(c, "trader_ids is required")
		return
	}

	result := make(map[string][]StrategyPerformanceTrend)

	for _, traderID := range traderIDs {
		trends, err := s.calculatePerformanceTrend(traderID)
		if err != nil {
			logger.Warnf("Failed to calculate trend for trader %s: %v", traderID, err)
			continue
		}
		result[traderID] = trends
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"trends":  result,
	})
}

// calculateStrategyMetrics 计算策略指标
func (s *Server) calculateStrategyMetrics(traderID string) (*StrategyComparisonMetrics, error) {
	// 获取订单数据
	orders, err := s.store.Order().GetTraderOrders(traderID, 10000)
	if err != nil {
		return nil, err
	}

	if len(orders) == 0 {
		return &StrategyComparisonMetrics{
			TraderID:     traderID,
			StrategyName: traderID,
		}, nil
	}

	metrics := &StrategyComparisonMetrics{
		TraderID:     traderID,
		StrategyName: traderID,
	}

	// 获取trader配置获取策略名称
	fullConfig, err := s.store.Trader().GetFullConfig("", traderID)
	if err == nil && fullConfig != nil && fullConfig.Trader != nil {
		metrics.StrategyName = fullConfig.Trader.Name
	}

	var totalProfit, totalLoss float64
	var winningTrades, losingTrades int
	var totalDuration float64
	var profits []float64
	var currentStreak int
	var currentStreakType string // "win" or "loss"
	var maxConsecutiveWins, maxConsecutiveLosses int
	var largestWin, largestLoss float64
	var startTime, endTime int64

	for i, order := range orders {
		if order.Status != "FILLED" {
			continue
		}

		metrics.TotalTrades++

		// 计算利润
		profit := (order.AvgFillPrice - order.Price) * order.FilledQuantity
		if order.Side == "SELL" {
			profit = -profit
		}

		profits = append(profits, profit)

		if profit > 0 {
			totalProfit += profit
			winningTrades++
			if profit > largestWin {
				largestWin = profit
			}

			// 连胜追踪
			if currentStreakType == "win" {
				currentStreak++
			} else {
				currentStreakType = "win"
				currentStreak = 1
			}
			if currentStreak > maxConsecutiveWins {
				maxConsecutiveWins = currentStreak
			}
		} else if profit < 0 {
			totalLoss += -profit
			losingTrades++
			if profit < largestLoss {
				largestLoss = profit
			}

			// 连亏追踪
			if currentStreakType == "loss" {
				currentStreak++
			} else {
				currentStreakType = "loss"
				currentStreak = 1
			}
			if currentStreak > maxConsecutiveLosses {
				maxConsecutiveLosses = currentStreak
			}
		}

		// 持仓时间
		if order.FilledAt > 0 && order.CreatedAt > 0 {
			duration := float64(order.FilledAt-order.CreatedAt) / (1000 * 3600) // 转为小时
			totalDuration += duration
		}

		// 时间范围
		if i == 0 || order.CreatedAt < startTime {
			startTime = order.CreatedAt
		}
		if i == 0 || order.CreatedAt > endTime {
			endTime = order.CreatedAt
		}
	}

	metrics.WinningTrades = winningTrades
	metrics.LosingTrades = losingTrades
	metrics.TotalProfit = totalProfit
	metrics.TotalLoss = totalLoss
	metrics.NetProfit = totalProfit - totalLoss
	metrics.LargestWin = largestWin
	metrics.LargestLoss = largestLoss
	metrics.ConsecutiveWins = maxConsecutiveWins
	metrics.ConsecutiveLosses = maxConsecutiveLosses

	// 计算各项指标
	if metrics.TotalTrades > 0 {
		metrics.WinRate = float64(winningTrades) / float64(metrics.TotalTrades) * 100
		metrics.AverageTrade = metrics.NetProfit / float64(metrics.TotalTrades)
		metrics.TradeDuration = totalDuration / float64(metrics.TotalTrades)
	}

	if winningTrades > 0 {
		metrics.AverageWin = totalProfit / float64(winningTrades)
	}

	if losingTrades > 0 {
		metrics.AverageLoss = totalLoss / float64(losingTrades)
	}

	if totalLoss > 0 {
		metrics.ProfitFactor = totalProfit / totalLoss
	}

	// 计算最大回撤
	metrics.MaxDrawdown = calculateMaxDrawdownFromOrders(orders)

	// 计算夏普比率
	if len(profits) > 0 {
		metrics.SharpeRatio = calculateSharpeRatio(profits)
	}

	// 时间范围
	if startTime > 0 {
		metrics.StartDate = time.UnixMilli(startTime).Format("2006-01-02")
		metrics.EndDate = time.UnixMilli(endTime).Format("2006-01-02")
		daysActive := (endTime - startTime) / (1000 * 86400)
		if daysActive < 1 {
			daysActive = 1
		}
		metrics.DaysActive = int(daysActive)

		// 假设初始资金1000 USDT
		initialCapital := 1000.0
		metrics.ReturnRate = (metrics.NetProfit / initialCapital) * 100
		metrics.AnnualizedReturn = (metrics.ReturnRate / float64(metrics.DaysActive)) * 365
	}

	return metrics, nil
}

// calculatePerformanceTrend 计算性能趋势
func (s *Server) calculatePerformanceTrend(traderID string) ([]StrategyPerformanceTrend, error) {
	orders, err := s.store.Order().GetTraderOrders(traderID, 10000)
	if err != nil {
		return nil, err
	}

	// 按日期分组
	dailyData := make(map[string]*StrategyPerformanceTrend)
	var cumulativeProfit float64

	for _, order := range orders {
		if order.Status != "FILLED" {
			continue
		}

		dateKey := time.UnixMilli(order.CreatedAt).Format("2006-01-02")
		if _, exists := dailyData[dateKey]; !exists {
			dailyData[dateKey] = &StrategyPerformanceTrend{
				Date: dateKey,
			}
		}

		trend := dailyData[dateKey]
		trend.TradeCount++

		// 计算利润
		profit := (order.AvgFillPrice - order.Price) * order.FilledQuantity
		if order.Side == "SELL" {
			profit = -profit
		}

		trend.DailyReturn += profit
		cumulativeProfit += profit

		// 假设初始资金1000
		trend.CumulativeROI = (cumulativeProfit / 1000.0) * 100

		// 计算当日胜率
		if profit > 0 {
			// 简化处理，实际需要追踪每日赢/输交易数
			trend.WinRate = 50.0 // 占位值
		}
	}

	// 转换为切片
	trends := make([]StrategyPerformanceTrend, 0, len(dailyData))
	for _, trend := range dailyData {
		trends = append(trends, *trend)
	}

	return trends, nil
}

// calculateMaxDrawdownFromOrders 从订单计算最大回撤
func calculateMaxDrawdownFromOrders(orders []*store.TraderOrder) float64 {
	if len(orders) == 0 {
		return 0
	}

	var peak, currentEquity, maxDD float64

	for _, order := range orders {
		if order.Status == "FILLED" {
			profit := (order.AvgFillPrice - order.Price) * order.FilledQuantity
			if order.Side == "SELL" {
				profit = -profit
			}
			currentEquity += profit

			if currentEquity > peak || peak == 0 {
				peak = currentEquity
			}

			if peak > 0 {
				dd := (peak - currentEquity) / peak * 100
				if dd > maxDD {
					maxDD = dd
				}
			}
		}
	}

	return maxDD
}

// calculateSharpeRatio 计算夏普比率
func calculateSharpeRatio(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// 计算平均收益
	var sum float64
	for _, r := range returns {
		sum += r
	}
	avgReturn := sum / float64(len(returns))

	// 计算标准差
	var variance float64
	for _, r := range returns {
		variance += (r - avgReturn) * (r - avgReturn)
	}
	variance /= float64(len(returns))
	stdDev := variance

	if stdDev == 0 {
		return 0
	}

	// 夏普比率 = (平均收益 - 无风险利率) / 标准差
	// 简化：假设无风险利率为0
	return avgReturn / stdDev
}
