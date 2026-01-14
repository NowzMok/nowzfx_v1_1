package trader

import (
	"fmt"
	"nofx/logger"
	"nofx/market"
	"nofx/store"
	"sync"
	"time"
)

// PendingOrderConfig 待决策系统配置
type PendingOrderConfig struct {
	// 最大待决策订单数量（每个交易员）
	MaxPendingOrders int `json:"max_pending_orders"`

	// 订单最大存活时间（超过此时间自动取消）
	MaxOrderAge time.Duration `json:"max_order_age"`

	// 价格最大偏离百分比（超过此值自动取消订单）
	MaxPriceDeviation float64 `json:"max_price_deviation"`

	// 替换策略：新订单需要满足的条件才能替换旧订单
	// 0: 只看置信度
	// 1: 置信度 + 时间（旧订单超过一定时间后更容易被替换）
	// 2: 智能模式（综合置信度、时间、价格偏离）
	ReplacementStrategy int `json:"replacement_strategy"`

	// 清理间隔
	CleanupInterval time.Duration `json:"cleanup_interval"`

	// 执行失败后重试次数（超过后自动取消）
	MaxRetryCount int `json:"max_retry_count"`
}

// DefaultPendingOrderConfig 默认配置
func DefaultPendingOrderConfig() *PendingOrderConfig {
	return &PendingOrderConfig{
		MaxPendingOrders:    10,              // 最多 10 个待决策订单
		MaxOrderAge:         12 * time.Hour,  // 12 小时后自动取消
		MaxPriceDeviation:   0.15,            // 价格偏离超过 15% 自动取消
		ReplacementStrategy: 2,               // 智能替换模式
		CleanupInterval:     5 * time.Minute, // 每 5 分钟清理一次
		MaxRetryCount:       3,               // 执行失败 3 次后取消
	}
}

// PendingOrderManager 待决策订单管理器
type PendingOrderManager struct {
	config   *PendingOrderConfig
	store    *store.Store
	stopCh   chan struct{}
	wg       sync.WaitGroup
	mu       sync.RWMutex
	running  bool
	retryMap map[string]int // 订单 ID -> 重试次数
}

// NewPendingOrderManager 创建待决策订单管理器
func NewPendingOrderManager(st *store.Store, config *PendingOrderConfig) *PendingOrderManager {
	if config == nil {
		config = DefaultPendingOrderConfig()
	}
	return &PendingOrderManager{
		config:   config,
		store:    st,
		stopCh:   make(chan struct{}),
		retryMap: make(map[string]int),
	}
}

// Start 启动后台清理任务
func (m *PendingOrderManager) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.mu.Unlock()

	m.wg.Add(1)
	go m.cleanupLoop()

	logger.Info("🧹 PendingOrderManager started")
}

// Stop 停止后台任务
func (m *PendingOrderManager) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	m.mu.Unlock()

	close(m.stopCh)
	m.wg.Wait()

	logger.Info("🧹 PendingOrderManager stopped")
}

// cleanupLoop 定期清理任务
func (m *PendingOrderManager) cleanupLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.runCleanup()
		}
	}
}

// runCleanup 执行清理
func (m *PendingOrderManager) runCleanup() {
	// 获取所有交易员
	traders, err := m.store.Trader().List("")
	if err != nil {
		logger.Errorf("❌ Failed to get traders for cleanup: %v", err)
		return
	}

	for _, trader := range traders {
		m.cleanupTraderOrders(trader.ID)
	}
}

// cleanupTraderOrders 清理单个交易员的订单
func (m *PendingOrderManager) cleanupTraderOrders(traderID string) {
	// 1. 标记过期订单
	expired, err := m.store.Analysis().MarkExpiredOrdersAsExpired(traderID)
	if err != nil {
		logger.Warnf("⚠️ Failed to mark expired orders: %v", err)
	} else if expired > 0 {
		logger.Infof("🗑️ Marked %d expired orders for trader %s", expired, traderID[:8])
	}

	// 2. 清理超过最大存活时间的订单
	stale, err := m.store.Analysis().CleanupStaleOrders(traderID, m.config.MaxOrderAge)
	if err != nil {
		logger.Warnf("⚠️ Failed to cleanup stale orders: %v", err)
	} else if stale > 0 {
		logger.Infof("🗑️ Cancelled %d stale orders for trader %s", stale, traderID[:8])
	}

	// 3. 检查价格偏离过大的订单
	m.cancelDeviatedOrders(traderID)

	// 4. 检查超过最大数量限制
	cancelled, err := m.store.Analysis().CancelOldestPendingOrders(traderID, m.config.MaxPendingOrders)
	if err != nil {
		logger.Warnf("⚠️ Failed to cancel excess orders: %v", err)
	} else if cancelled > 0 {
		logger.Infof("🗑️ Cancelled %d excess orders for trader %s (limit: %d)",
			cancelled, traderID[:8], m.config.MaxPendingOrders)
	}
}

// cancelDeviatedOrders 取消价格偏离过大的订单
func (m *PendingOrderManager) cancelDeviatedOrders(traderID string) {
	orders, err := m.store.Analysis().GetOrdersWithPriceDeviation(traderID, m.config.MaxPriceDeviation)
	if err != nil {
		logger.Warnf("⚠️ Failed to get orders for deviation check: %v", err)
		return
	}

	for _, order := range orders {
		// 获取当前价格
		marketData, err := market.Get(order.Symbol)
		if err != nil {
			// 如果无法获取价格，可能是无效的交易对
			m.handleInvalidSymbol(order)
			continue
		}

		currentPrice := marketData.CurrentPrice
		if currentPrice <= 0 {
			continue
		}

		// 计算偏离百分比
		deviation := (currentPrice - order.TriggerPrice) / order.TriggerPrice
		if deviation < 0 {
			deviation = -deviation
		}

		// 如果偏离超过阈值，取消订单
		if deviation > m.config.MaxPriceDeviation {
			reason := fmt.Sprintf("Price deviation too high: %.2f%% (current: %.4f, trigger: %.4f, max: %.2f%%)",
				deviation*100, currentPrice, order.TriggerPrice, m.config.MaxPriceDeviation*100)

			if err := m.store.Analysis().CancelPendingOrder(order.ID, reason); err != nil {
				logger.Warnf("⚠️ Failed to cancel deviated order %s: %v", order.ID, err)
			} else {
				logger.Infof("🗑️ Cancelled order %s due to price deviation: %s %.2f%%",
					order.Symbol, order.ID[:8], deviation*100)
			}
		}
	}
}

// handleInvalidSymbol 处理无效交易对
func (m *PendingOrderManager) handleInvalidSymbol(order *store.PendingOrder) {
	m.mu.Lock()
	m.retryMap[order.ID]++
	retries := m.retryMap[order.ID]
	m.mu.Unlock()

	if retries >= m.config.MaxRetryCount {
		reason := fmt.Sprintf("Invalid symbol or market data unavailable after %d attempts", retries)
		if err := m.store.Analysis().CancelPendingOrder(order.ID, reason); err != nil {
			logger.Warnf("⚠️ Failed to cancel invalid order %s: %v", order.ID, err)
		} else {
			logger.Infof("🗑️ Cancelled order %s due to invalid symbol: %s", order.Symbol, order.ID[:8])
		}

		m.mu.Lock()
		delete(m.retryMap, order.ID)
		m.mu.Unlock()
	}
}

// RecordExecutionFailure 记录执行失败
func (m *PendingOrderManager) RecordExecutionFailure(orderID string, err error) {
	m.mu.Lock()
	m.retryMap[orderID]++
	retries := m.retryMap[orderID]
	m.mu.Unlock()

	if retries >= m.config.MaxRetryCount {
		reason := fmt.Sprintf("Execution failed %d times: %v", retries, err)
		if err := m.store.Analysis().CancelPendingOrder(orderID, reason); err != nil {
			logger.Warnf("⚠️ Failed to cancel failed order %s: %v", orderID, err)
		} else {
			logger.Infof("🗑️ Cancelled order %s after %d execution failures", orderID[:8], retries)
		}

		m.mu.Lock()
		delete(m.retryMap, orderID)
		m.mu.Unlock()
	}
}

// ShouldReplaceOrder 判断是否应该替换现有订单
// 返回 true 表示应该用新订单替换旧订单
func (m *PendingOrderManager) ShouldReplaceOrder(existingOrder *store.PendingOrder, newConfidence float64, newTriggerPrice float64) bool {
	switch m.config.ReplacementStrategy {
	case 0:
		// 只看置信度
		return newConfidence > existingOrder.Confidence

	case 1:
		// 置信度 + 时间
		orderAge := time.Since(existingOrder.CreatedAt)
		// 订单越旧，替换阈值越低
		ageBonus := float64(orderAge.Hours()) * 0.02 // 每小时降低 2%
		adjustedThreshold := existingOrder.Confidence - ageBonus
		return newConfidence > adjustedThreshold

	case 2:
		// 智能模式
		return m.smartReplacementCheck(existingOrder, newConfidence, newTriggerPrice)

	default:
		return newConfidence > existingOrder.Confidence
	}
}

// smartReplacementCheck 智能替换检查
func (m *PendingOrderManager) smartReplacementCheck(existingOrder *store.PendingOrder, newConfidence float64, newTriggerPrice float64) bool {
	// 1. 基础置信度比较
	confidenceScore := 0.0
	if newConfidence > existingOrder.Confidence {
		confidenceScore = (newConfidence - existingOrder.Confidence) * 100 // 置信度差异分数
	}

	// 2. 时间因素：旧订单越久，越容易被替换
	orderAge := time.Since(existingOrder.CreatedAt)
	ageScore := orderAge.Hours() * 5 // 每小时 5 分

	// 3. 价格偏离因素：检查旧订单的触发价格与当前价格的偏离
	priceDeviationScore := 0.0
	if marketData, err := market.Get(existingOrder.Symbol); err == nil && marketData.CurrentPrice > 0 {
		deviation := (marketData.CurrentPrice - existingOrder.TriggerPrice) / existingOrder.TriggerPrice
		if deviation < 0 {
			deviation = -deviation
		}
		// 偏离越大，越应该被替换
		priceDeviationScore = deviation * 100 // 每 1% 偏离 = 1 分
	}

	// 总分：置信度差异分数 + 时间分数 + 价格偏离分数
	totalScore := confidenceScore + ageScore + priceDeviationScore

	// 阈值：总分超过 20 分就替换
	// 这意味着：
	// - 置信度高 10% = 10 分
	// - 订单存在 2 小时 = 10 分
	// - 价格偏离 10% = 10 分
	// 以上任意组合超过 20 分就替换
	threshold := 20.0

	if totalScore >= threshold {
		logger.Debugf("🔄 Smart replacement: confidence=%.1f, age=%.1f, deviation=%.1f, total=%.1f >= %.1f",
			confidenceScore, ageScore, priceDeviationScore, totalScore, threshold)
		return true
	}

	return false
}

// GetConfig 获取配置
func (m *PendingOrderManager) GetConfig() *PendingOrderConfig {
	return m.config
}

// UpdateConfig 更新配置
func (m *PendingOrderManager) UpdateConfig(config *PendingOrderConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
}

// ForceCleanup 强制执行一次清理
func (m *PendingOrderManager) ForceCleanup(traderID string) {
	m.cleanupTraderOrders(traderID)
}

// GetStatistics 获取统计信息
func (m *PendingOrderManager) GetStatistics(traderID string) (map[string]interface{}, error) {
	// 分别统计各个状态的订单数量
	var pendingCount int64
	if err := m.store.GormDB().Model(&store.PendingOrder{}).
		Where("trader_id = ? AND status = 'PENDING' AND expires_at > ?", traderID, time.Now().UTC()).
		Count(&pendingCount).Error; err != nil {
		pendingCount = 0
	}

	var triggeredCount int64
	if err := m.store.GormDB().Model(&store.PendingOrder{}).
		Where("trader_id = ? AND status = 'TRIGGERED'", traderID).
		Count(&triggeredCount).Error; err != nil {
		triggeredCount = 0
	}

	var filledCount int64
	if err := m.store.GormDB().Model(&store.PendingOrder{}).
		Where("trader_id = ? AND status = 'FILLED'", traderID).
		Count(&filledCount).Error; err != nil {
		filledCount = 0
	}

	return map[string]interface{}{
		"pending_count":       pendingCount,
		"triggered_count":     triggeredCount,
		"filled_count":        filledCount,
		"max_pending":         m.config.MaxPendingOrders,
		"max_order_age":       m.config.MaxOrderAge.String(),
		"max_price_deviation": fmt.Sprintf("%.2f%%", m.config.MaxPriceDeviation*100),
		"cleanup_interval":    m.config.CleanupInterval.String(),
	}, nil
}
