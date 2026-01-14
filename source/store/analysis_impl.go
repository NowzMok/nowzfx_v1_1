package store

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// AnalysisImpl AI 分析数据访问实现
type AnalysisImpl struct {
	db *gorm.DB
}

// NewAnalysisImpl 创建分析数据访问实例
func NewAnalysisImpl(db *gorm.DB) AnalysisStore {
	return &AnalysisImpl{db: db}
}

// SaveAnalysis 保存分析记录
func (a *AnalysisImpl) SaveAnalysis(analysis *AnalysisRecord) error {
	if analysis.ID == "" {
		analysis.ID = generateUUID()
	}
	if analysis.CreatedAt.IsZero() {
		analysis.CreatedAt = time.Now().UTC()
	}
	if analysis.ExpiresAt.IsZero() {
		analysis.ExpiresAt = time.Now().UTC().Add(4 * time.Hour) // 4小时过期
	}
	if analysis.Status == "" {
		analysis.Status = "ACTIVE"
	}
	return a.db.Save(analysis).Error
}

// GetAnalysisByID 获取分析记录
func (a *AnalysisImpl) GetAnalysisByID(id string) (*AnalysisRecord, error) {
	var analysis AnalysisRecord
	if err := a.db.Where("id = ?", id).First(&analysis).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &analysis, nil
}

// GetActiveAnalyses 获取有效的分析记录
func (a *AnalysisImpl) GetActiveAnalyses(traderID string) ([]*AnalysisRecord, error) {
	var analyses []*AnalysisRecord
	err := a.db.Where("trader_id = ? AND status = ? AND expires_at > ?", traderID, "ACTIVE", time.Now().UTC()).
		Order("created_at DESC").
		Find(&analyses).Error
	return analyses, err
}

// GetAnalysesBySymbol 获取某交易对的最新分析
func (a *AnalysisImpl) GetAnalysesBySymbol(traderID, symbol string) (*AnalysisRecord, error) {
	var analysis AnalysisRecord
	err := a.db.Where("trader_id = ? AND symbol = ? AND status = ? AND expires_at > ?",
		traderID, symbol, "ACTIVE", time.Now().UTC()).
		Order("created_at DESC").
		First(&analysis).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &analysis, nil
}

// UpdateAnalysisStatus 更新分析状态
func (a *AnalysisImpl) UpdateAnalysisStatus(id, status string) error {
	return a.db.Model(&AnalysisRecord{}).Where("id = ?", id).Update("status", status).Error
}

// DeleteExpiredAnalyses 删除过期分析
func (a *AnalysisImpl) DeleteExpiredAnalyses(traderID string) error {
	return a.db.Where("trader_id = ? AND expires_at < ?", traderID, time.Now().UTC()).
		Delete(&AnalysisRecord{}).Error
}

// SavePendingOrder 保存待执行订单
func (a *AnalysisImpl) SavePendingOrder(order *PendingOrder) error {
	if order.ID == "" {
		order.ID = generateUUID()
	}
	if order.CreatedAt.IsZero() {
		order.CreatedAt = time.Now().UTC()
	}
	if order.ExpiresAt.IsZero() {
		order.ExpiresAt = time.Now().UTC().Add(24 * time.Hour) // 1天过期
	}
	if order.Status == "" {
		order.Status = "PENDING"
	}
	return a.db.Save(order).Error
}

// GetPendingOrderByID 获取待执行订单
func (a *AnalysisImpl) GetPendingOrderByID(id string) (*PendingOrder, error) {
	var order PendingOrder
	if err := a.db.Where("id = ?", id).First(&order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// GetPendingOrdersByTrader 获取交易员的所有待执行订单（包括已成交）
// 注意：PENDING订单需要检查过期时间，但TRIGGERED和FILLED订单即使过期也显示（它们是历史记录）
func (a *AnalysisImpl) GetPendingOrdersByTrader(traderID string) ([]*PendingOrder, error) {
	var orders []*PendingOrder
	// PENDING订单需要未过期，TRIGGERED和FILLED订单始终显示（最近7天内）
	err := a.db.Where(
		"trader_id = ? AND ("+
			"(status = 'PENDING' AND expires_at > ?) OR "+
			"(status IN ? AND created_at > ?))",
		traderID,
		time.Now().UTC(),
		[]string{"TRIGGERED", "FILLED"},
		time.Now().UTC().Add(-7*24*time.Hour), // 最近7天的已触发/已成交订单
	).
		Order("created_at DESC").
		Find(&orders).Error
	return orders, err
}

// GetPendingOrdersByStatus 获取特定状态的待执行订单
func (a *AnalysisImpl) GetPendingOrdersByStatus(traderID, status string) ([]*PendingOrder, error) {
	var orders []*PendingOrder
	err := a.db.Where("trader_id = ? AND status = ? AND expires_at > ?",
		traderID, status, time.Now().UTC()).
		Order("created_at DESC").
		Find(&orders).Error
	return orders, err
}

// UpdatePendingOrderStatus 更新待执行订单状态
func (a *AnalysisImpl) UpdatePendingOrderStatus(id, status string, triggeredPrice float64, triggeredAt time.Time) error {
	return a.db.Model(&PendingOrder{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":          status,
			"triggered_price": triggeredPrice,
			"triggered_at":    triggeredAt,
		}).Error
}

// UpdatePendingOrderFilled 标记订单已成交（同时记录触发价格和时间）
func (a *AnalysisImpl) UpdatePendingOrderFilled(id string, filledAt time.Time, orderID int64) error {
	return a.db.Model(&PendingOrder{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       "FILLED",
			"triggered_at": filledAt, // 触发时间与成交时间相同
			"filled_at":    filledAt,
			"order_id":     orderID,
		}).Error
}

// UpdatePendingOrderFilledWithPrice 标记订单已成交（包含触发价格）
func (a *AnalysisImpl) UpdatePendingOrderFilledWithPrice(id string, triggeredPrice float64, filledAt time.Time, orderID int64) error {
	return a.db.Model(&PendingOrder{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":          "FILLED",
			"triggered_price": triggeredPrice,
			"triggered_at":    filledAt,
			"filled_at":       filledAt,
			"order_id":        orderID,
		}).Error
}

// CancelPendingOrder 取消待执行订单
func (a *AnalysisImpl) CancelPendingOrder(id, reason string) error {
	return a.db.Model(&PendingOrder{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        "CANCELLED",
			"cancel_reason": reason,
		}).Error
}

// DeleteExpiredPendingOrders 删除过期待执行订单
func (a *AnalysisImpl) DeleteExpiredPendingOrders(traderID string) error {
	return a.db.Where("trader_id = ? AND expires_at < ?", traderID, time.Now().UTC()).
		Delete(&PendingOrder{}).Error
}

// MarkExpiredOrdersAsExpired 标记过期订单为 EXPIRED 状态（不删除，保留历史记录）
func (a *AnalysisImpl) MarkExpiredOrdersAsExpired(traderID string) (int64, error) {
	result := a.db.Model(&PendingOrder{}).
		Where("trader_id = ? AND status = 'PENDING' AND expires_at < ?", traderID, time.Now().UTC()).
		Updates(map[string]interface{}{
			"status":        "EXPIRED",
			"cancel_reason": "Order expired without being triggered",
		})
	return result.RowsAffected, result.Error
}

// CleanupStaleOrders 清理过时订单（创建时间超过 maxAge 且价格偏离过大的订单）
func (a *AnalysisImpl) CleanupStaleOrders(traderID string, maxAge time.Duration) (int64, error) {
	cutoffTime := time.Now().UTC().Add(-maxAge)
	result := a.db.Model(&PendingOrder{}).
		Where("trader_id = ? AND status = 'PENDING' AND created_at < ?", traderID, cutoffTime).
		Updates(map[string]interface{}{
			"status":        "CANCELLED",
			"cancel_reason": fmt.Sprintf("Order too old (>%v)", maxAge),
		})
	return result.RowsAffected, result.Error
}

// GetPendingOrderCount 获取待决策订单数量（包括 PENDING 和 TRIGGERED 状态）
func (a *AnalysisImpl) GetPendingOrderCount(traderID string) (int64, error) {
	var count int64
	err := a.db.Model(&PendingOrder{}).
		Where("trader_id = ? AND status IN ('PENDING', 'TRIGGERED') AND expires_at > ?", traderID, time.Now().UTC()).
		Count(&count).Error
	return count, err
}

// CancelOldestPendingOrders 取消最旧的订单，保留最新的 keepCount 个
func (a *AnalysisImpl) CancelOldestPendingOrders(traderID string, keepCount int) (int64, error) {
	// 获取所有 PENDING 订单，按创建时间倒序
	var orders []*PendingOrder
	if err := a.db.Where("trader_id = ? AND status = 'PENDING' AND expires_at > ?", traderID, time.Now().UTC()).
		Order("created_at DESC").
		Find(&orders).Error; err != nil {
		return 0, err
	}

	// 如果订单数不超过限制，不需要取消
	if len(orders) <= keepCount {
		return 0, nil
	}

	// 取消多余的订单（保留最新的 keepCount 个）
	var cancelCount int64
	for i := keepCount; i < len(orders); i++ {
		if err := a.db.Model(&PendingOrder{}).
			Where("id = ?", orders[i].ID).
			Updates(map[string]interface{}{
				"status":        "CANCELLED",
				"cancel_reason": fmt.Sprintf("Exceeded max pending orders limit (%d)", keepCount),
			}).Error; err != nil {
			continue
		}
		cancelCount++
	}

	return cancelCount, nil
}

// GetOrdersWithPriceDeviation 获取价格偏离过大的订单
// maxDeviation 是最大允许偏离百分比（如 0.15 表示 15%）
func (a *AnalysisImpl) GetOrdersWithPriceDeviation(traderID string, maxDeviation float64) ([]*PendingOrder, error) {
	var orders []*PendingOrder
	// 注意：这只返回订单，实际价格偏离检查需要在业务层获取当前价格后计算
	err := a.db.Where("trader_id = ? AND status = 'PENDING' AND expires_at > ?", traderID, time.Now().UTC()).
		Order("created_at DESC").
		Find(&orders).Error
	return orders, err
}

// SaveTradeHistory 保存交易历史
func (a *AnalysisImpl) SaveTradeHistory(trade *TradeHistoryRecord) error {
	if trade.ID == "" {
		trade.ID = generateUUID()
	}
	if trade.CreatedAt.IsZero() {
		trade.CreatedAt = time.Now().UTC()
	}
	return a.db.Save(trade).Error
}

// GetTradeHistoryByID 获取交易历史
func (a *AnalysisImpl) GetTradeHistoryByID(id string) (*TradeHistoryRecord, error) {
	var trade TradeHistoryRecord
	if err := a.db.Where("id = ?", id).First(&trade).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &trade, nil
}

// GetTradeHistoriesByTrader 获取交易员的交易历史
func (a *AnalysisImpl) GetTradeHistoriesByTrader(traderID string, limit int) ([]*TradeHistoryRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	var trades []*TradeHistoryRecord
	err := a.db.Where("trader_id = ?", traderID).
		Order("exit_time DESC").
		Limit(limit).
		Find(&trades).Error
	return trades, err
}

// GetTradeHistoriesBySymbol 获取某交易对的交易历史
func (a *AnalysisImpl) GetTradeHistoriesBySymbol(traderID, symbol string, limit int) ([]*TradeHistoryRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	var trades []*TradeHistoryRecord
	err := a.db.Where("trader_id = ? AND symbol = ?", traderID, symbol).
		Order("exit_time DESC").
		Limit(limit).
		Find(&trades).Error
	return trades, err
}

// GetTradeHistoryInPeriod 获取时间段内的交易历史
func (a *AnalysisImpl) GetTradeHistoryInPeriod(traderID string, startTime, endTime time.Time) ([]*TradeHistoryRecord, error) {
	var trades []*TradeHistoryRecord
	err := a.db.Where("trader_id = ? AND entry_time >= ? AND entry_time <= ?", traderID, startTime, endTime).
		Order("entry_time ASC").
		Find(&trades).Error
	return trades, err
}

// InitSchema 初始化表结构
func (a *AnalysisImpl) InitSchema() error {
	if err := a.db.AutoMigrate(
		&AnalysisRecord{},
		&PendingOrder{},
		&TradeHistoryRecord{},
	); err != nil {
		return fmt.Errorf("failed to migrate tables: %w", err)
	}

	// 创建索引（如果不存在）
	// 这些索引是在 struct tags 中定义的，但如果已存在则忽略
	_ = a.db.Exec(`CREATE INDEX IF NOT EXISTS idx_trader_symbol_time ON ai_analysis(trader_id, symbol, created_at DESC)`)
	_ = a.db.Exec(`CREATE INDEX IF NOT EXISTS idx_trader_status ON ai_analysis(trader_id, status)`)
	_ = a.db.Exec(`CREATE INDEX IF NOT EXISTS idx_symbol_time ON ai_analysis(symbol, created_at DESC)`)

	return nil
}

// TryMarkAsExecuting 尝试原子地标记订单为正在执行
// 返回 true 如果成功标记，false 如果已被其他进程标记
func (a *AnalysisImpl) TryMarkAsExecuting(orderID string) bool {
	// 使用数据库级别的原子操作
	result := a.db.Model(&PendingOrder{}).
		Where("id = ? AND is_executing = ? AND status = ?", orderID, false, "PENDING").
		Updates(map[string]interface{}{
			"is_executing":      true,
			"execution_version": gorm.Expr("execution_version + 1"),
		})

	return result.RowsAffected > 0
}

// MarkAsExecuted 标记订单为已执行
func (a *AnalysisImpl) MarkAsExecuted(orderID string) error {
	now := time.Now().UTC()
	return a.db.Model(&PendingOrder{}).
		Where("id = ?", orderID).
		Updates(map[string]interface{}{
			"status":       "FILLED",
			"executed_at":  &now,
			"is_executing": false,
		}).Error
}

// CancelExecution 取消执行标记（用于错误恢复）
func (a *AnalysisImpl) CancelExecution(orderID string) error {
	return a.db.Model(&PendingOrder{}).
		Where("id = ?", orderID).
		Updates(map[string]interface{}{
			"is_executing": false,
		}).Error
}

func generateUUID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
