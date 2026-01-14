package api

import (
	"net/http"
	"time"

	"nofx/logger"
	"nofx/store"

	"github.com/gin-gonic/gin"
)

// OrderStatistics 订单统计数据
type OrderStatistics struct {
	TotalOrders           int     `json:"total_orders"`             // 总订单数
	ExecutedOrders        int     `json:"executed_orders"`          // 已执行订单数
	CancelledOrders       int     `json:"cancelled_orders"`         // 已取消订单数
	PendingOrders         int     `json:"pending_orders"`           // 待执行订单数
	ExecutionRate         float64 `json:"execution_rate"`           // 执行率 %
	SuccessRate           float64 `json:"success_rate"`             // 成功率 %
	AverageWaitTime       float64 `json:"average_wait_time"`        // 平均等待时间 (秒)
	TotalProfit           float64 `json:"total_profit"`             // 总利润
	TotalLoss             float64 `json:"total_loss"`               // 总亏损
	ProfitFactor          float64 `json:"profit_factor"`            // 利润因子
	WinningTrades         int     `json:"winning_trades"`           // 赢利交易数
	LosingTrades          int     `json:"losing_trades"`            // 亏损交易数
	WinRate               float64 `json:"win_rate"`                 // 赢率 %
	AverageProfitPerTrade float64 `json:"average_profit_per_trade"` // 平均交易利润
	MaxDrawdown           float64 `json:"max_drawdown"`             // 最大回撤 %
	TimeRange             string  `json:"time_range"`               // 时间范围
}

// OrderTrend 日常订单趋势
type OrderTrend struct {
	Date          string  `json:"date"`
	OrderCount    int     `json:"order_count"`
	ExecutedCount int     `json:"executed_count"`
	SuccessRate   float64 `json:"success_rate"`
	DailyProfit   float64 `json:"daily_profit"`
}

// handleOrderStatistics 获取订单统计信息
func (s *Server) handleOrderStatistics(c *gin.Context) {
	traderID := c.Query("trader_id")
	if traderID == "" {
		SafeBadRequest(c, "trader_id is required")
		return
	}

	// 获取统计数据
	stats, err := getOrderStatistics(s.store, traderID)
	if err != nil {
		logger.Warnf("Failed to get order statistics: %v", err)
		// 返回空的默认统计
		stats = &OrderStatistics{
			TimeRange: "N/A",
		}
	}

	c.JSON(http.StatusOK, stats)
}

// getOrderStatistics 计算订单统计数据
func getOrderStatistics(st *store.Store, traderID string) (*OrderStatistics, error) {
	stats := &OrderStatistics{
		TimeRange: time.Now().Format("2006-01-02 15:04:05"),
	}

	// 获取订单列表
	orders, err := st.Order().GetTraderOrders(traderID, 10000)
	if err != nil {
		return stats, err
	}

	if len(orders) == 0 {
		return stats, nil
	}

	var executedCount, cancelledCount, pendingCount int
	var totalWaitTime int64
	var totalProfit, totalLoss float64
	var winningTrades, losingTrades int

	for _, order := range orders {
		stats.TotalOrders++

		// Count by status
		switch order.Status {
		case "FILLED":
			executedCount++
			pendingCount = 0
		case "CANCELLED":
			cancelledCount++
		case "NEW", "PARTIALLY_FILLED":
			pendingCount++
		}

		// 计算利润/亏损
		if order.Status == "FILLED" && order.FilledQuantity > 0 {
			profit := (order.AvgFillPrice - order.Price) * order.FilledQuantity
			if order.Side == "SELL" {
				profit = -profit
			}

			if profit > 0 {
				totalProfit += profit
				winningTrades++
			} else if profit < 0 {
				totalLoss += -profit
				losingTrades++
			}

			// 计算等待时间
			if order.FilledAt > 0 && order.CreatedAt > 0 {
				waitTime := (order.FilledAt - order.CreatedAt) / 1000 // 转为秒
				totalWaitTime += waitTime
			}
		}
	}

	stats.ExecutedOrders = executedCount
	stats.CancelledOrders = cancelledCount
	stats.PendingOrders = pendingCount

	// 计算各项指标
	if stats.TotalOrders > 0 {
		stats.ExecutionRate = float64(executedCount) / float64(stats.TotalOrders) * 100
	}

	if winningTrades+losingTrades > 0 {
		stats.SuccessRate = float64(winningTrades) / float64(winningTrades+losingTrades) * 100
		stats.WinRate = stats.SuccessRate
	}

	if executedCount > 0 {
		stats.AverageWaitTime = float64(totalWaitTime) / float64(executedCount)
	}

	stats.TotalProfit = totalProfit
	stats.TotalLoss = totalLoss
	stats.WinningTrades = winningTrades
	stats.LosingTrades = losingTrades

	// 利润因子
	if totalLoss > 0 {
		stats.ProfitFactor = totalProfit / totalLoss
	}

	// 平均交易利润
	if winningTrades+losingTrades > 0 {
		stats.AverageProfitPerTrade = (totalProfit - totalLoss) / float64(winningTrades+losingTrades)
	}

	// 计算最大回撤
	stats.MaxDrawdown = calculateMaxDrawdown(orders)

	return stats, nil
}

// calculateMaxDrawdown 计算最大回撤
func calculateMaxDrawdown(orders []*store.TraderOrder) float64 {
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

// handleOrderStatisticsTrend 获取订单统计趋势（按时间）
func (s *Server) handleOrderStatisticsTrend(c *gin.Context) {
	traderID := c.Query("trader_id")
	if traderID == "" {
		SafeBadRequest(c, "trader_id is required")
		return
	}

	// 获取按天的统计数据
	trends, err := getOrderStatisticsTrend(s.store, traderID)
	if err != nil {
		logger.Warnf("Failed to get order statistics trend: %v", err)
		trends = []OrderTrend{}
	}

	c.JSON(http.StatusOK, gin.H{
		"trends": trends,
	})
}

// getOrderStatisticsTrend 获取按日期的统计数据
func getOrderStatisticsTrend(st *store.Store, traderID string) ([]OrderTrend, error) {
	orders, err := st.Order().GetTraderOrders(traderID, 10000)
	if err != nil {
		return []OrderTrend{}, err
	}

	// 按日期分组
	trendMap := make(map[string]*OrderTrend)

	for _, order := range orders {
		dateKey := time.UnixMilli(order.CreatedAt).Format("2006-01-02")
		if _, exists := trendMap[dateKey]; !exists {
			trendMap[dateKey] = &OrderTrend{
				Date: dateKey,
			}
		}

		trend := trendMap[dateKey]
		trend.OrderCount++

		if order.Status == "FILLED" {
			trend.ExecutedCount++

			profit := (order.AvgFillPrice - order.Price) * order.FilledQuantity
			if order.Side == "SELL" {
				profit = -profit
			}
			trend.DailyProfit += profit
		}
	}

	// 计算成功率
	for _, trend := range trendMap {
		if trend.OrderCount > 0 {
			trend.SuccessRate = float64(trend.ExecutedCount) / float64(trend.OrderCount) * 100
		}
	}

	// 转换为slice
	result := make([]OrderTrend, 0, len(trendMap))
	for _, trend := range trendMap {
		result = append(result, *trend)
	}

	return result, nil
}

// handleOrdersBySymbol 获取按币种分类的订单统计
func (s *Server) handleOrdersBySymbol(c *gin.Context) {
	traderID := c.Query("trader_id")
	if traderID == "" {
		SafeBadRequest(c, "trader_id is required")
		return
	}

	orders, err := s.store.Order().GetTraderOrders(traderID, 10000)
	if err != nil {
		SafeInternalError(c, "Get orders", err)
		return
	}

	// 按币种分组统计
	symbolStats := make(map[string]map[string]interface{})
	for _, order := range orders {
		symbol := order.Symbol
		if _, exists := symbolStats[symbol]; !exists {
			symbolStats[symbol] = map[string]interface{}{
				"symbol":       symbol,
				"count":        0,
				"executed":     0,
				"cancelled":    0,
				"total_profit": 0.0,
				"win_rate":     0.0,
				"winning":      0,
				"losing":       0,
			}
		}

		stats := symbolStats[symbol]
		stats["count"] = stats["count"].(int) + 1

		if order.Status == "FILLED" {
			stats["executed"] = stats["executed"].(int) + 1

			profit := (order.AvgFillPrice - order.Price) * order.FilledQuantity
			if order.Side == "SELL" {
				profit = -profit
			}

			totalProfit := stats["total_profit"].(float64)
			stats["total_profit"] = totalProfit + profit

			if profit > 0 {
				stats["winning"] = stats["winning"].(int) + 1
			} else if profit < 0 {
				stats["losing"] = stats["losing"].(int) + 1
			}
		} else if order.Status == "CANCELLED" {
			stats["cancelled"] = stats["cancelled"].(int) + 1
		}
	}

	// 计算胜率
	for _, stats := range symbolStats {
		winning := stats["winning"].(int)
		losing := stats["losing"].(int)
		total := winning + losing
		if total > 0 {
			stats["win_rate"] = float64(winning) / float64(total) * 100
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"statistics": symbolStats,
	})
}
