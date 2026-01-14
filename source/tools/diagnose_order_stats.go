package main

import (
	"fmt"
	"nofx/logger"
	"nofx/store"
	"time"
)

// DiagnoseOrderStats 诊断订单统计问题
func DiagnoseOrderStats(traderID string, db *store.Store) {
	logger.Infof("🔍 开始诊断订单统计问题 (TraderID: %s)", traderID)

	// 1. 获取所有订单
	allOrders, err := db.Analysis().GetPendingOrdersByTrader(traderID)
	if err != nil {
		logger.Errorf("❌ 获取订单失败: %v", err)
		return
	}

	logger.Infof("📊 总订单数: %d", len(allOrders))

	// 2. 按状态分组统计
	statusCounts := make(map[string]int)
	expiredCount := 0
	activePendingCount := 0
	now := time.Now().UTC()

	for _, order := range allOrders {
		statusCounts[order.Status]++

		if order.Status == "PENDING" {
			if order.ExpiresAt.After(now) {
				activePendingCount++
			} else {
				expiredCount++
			}
		}
	}

	logger.Infof("📋 状态分布:")
	for status, count := range statusCounts {
		logger.Infof("   - %s: %d", status, count)
	}

	logger.Infof("⏰ PENDING订单详情:")
	logger.Infof("   - 未过期 (活跃): %d", activePendingCount)
	logger.Infof("   - 已过期: %d", expiredCount)

	// 3. 检查最近7天的订单
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour)
	recentCount := 0
	for _, order := range allOrders {
		if order.CreatedAt.After(sevenDaysAgo) {
			recentCount++
		}
	}
	logger.Infof("📅 最近7天创建的订单: %d", recentCount)

	// 4. 检查是否有大量重复订单
	groupedOrders := make(map[string][]*store.PendingOrder)
	for _, order := range allOrders {
		key := order.Symbol
		groupedOrders[key] = append(groupedOrders[key], order)
	}

	duplicateGroups := 0
	totalDuplicates := 0
	for symbol, orders := range groupedOrders {
		if len(orders) > 1 {
			duplicateGroups++
			totalDuplicates += len(orders) - 1
			logger.Infof("🔄 交易对 %s 有 %d 个订单", symbol, len(orders))
		}
	}

	if duplicateGroups > 0 {
		logger.Infof("⚠️ 发现 %d 个交易对有重复订单，共多余 %d 个订单", duplicateGroups, totalDuplicates)
	}

	// 5. 检查TRIGGERED订单的详细信息
	triggeredOrders := []*store.PendingOrder{}
	for _, order := range allOrders {
		if order.Status == "TRIGGERED" {
			triggeredOrders = append(triggeredOrders, order)
		}
	}

	if len(triggeredOrders) > 0 {
		logger.Infof("⚡ TRIGGERED订单详情 (共%d个):", len(triggeredOrders))
		for i, order := range triggeredOrders {
			if i < 5 { // 只显示前5个
				age := now.Sub(order.CreatedAt)
				logger.Infof("   %d. %s - 创建于%.1f小时前, 触发价: %.4f",
					i+1, order.Symbol, age.Hours(), order.TriggerPrice)
			}
		}
		if len(triggeredOrders) > 5 {
			logger.Infof("   ... 还有 %d 个", len(triggeredOrders)-5)
		}
	}

	// 6. 检查清理机制状态
	logger.Infof("🧹 清理机制检查:")

	// 检查是否有过期但未标记的订单
	var expiredPendingCount int64
	db.GormDB().Model(&store.PendingOrder{}).
		Where("trader_id = ? AND status = 'PENDING' AND expires_at < ?", traderID, now).
		Count(&expiredPendingCount)

	if expiredPendingCount > 0 {
		logger.Infof("   ⚠️ 发现 %d 个PENDING订单已过期但未标记", expiredPendingCount)
	} else {
		logger.Infof("   ✅ 没有过期未标记的PENDING订单")
	}

	// 7. 建议
	logger.Infof("💡 诊断建议:")
	if len(allOrders) > 50 {
		logger.Infof("   - 订单总数过多 (%d)，建议执行清理", len(allOrders))
	}
	if duplicateGroups > 0 {
		logger.Infof("   - 存在重复订单，建议检查去重逻辑")
	}
	if expiredPendingCount > 0 {
		logger.Infof("   - 有 %d 个过期订单需要清理", expiredPendingCount)
	}
	if len(triggeredOrders) > 10 {
		logger.Infof("   - TRIGGERED订单过多 (%d)，可能执行有问题", len(triggeredOrders))
	}

	// 8. 统计修正建议
	logger.Infof("📊 修正后的统计应该是:")
	activeCount := activePendingCount + len(triggeredOrders)
	logger.Infof("   - 活跃订单 (PENDING未过期 + TRIGGERED): %d", activeCount)
	logger.Infof("   - 已成交订单 (FILLED): %d", statusCounts["FILLED"])
	logger.Infof("   - 已取消/过期: %d", statusCounts["CANCELLED"]+statusCounts["EXPIRED"]+expiredCount)
	logger.Infof("   - 总订单数: %d", len(allOrders))

	logger.Infof("✅ 诊断完成")
}

func main() {
	// 这是一个诊断工具，需要配合具体 traderID 使用
	fmt.Println("请在代码中调用 DiagnoseOrderStats(traderID, store) 进行诊断")
}
