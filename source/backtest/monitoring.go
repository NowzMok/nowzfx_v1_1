package backtest

import (
	"fmt"
	"math"
	"nofx/logger"
	"nofx/store"
	"sync"
	"time"
)

// PerformanceMonitor 交易性能监控器
type PerformanceMonitor struct {
	mu                 sync.RWMutex
	traderID           string
	store              *store.Store
	lastMetricTime     time.Time
	collectionInterval time.Duration // 数据收集间隔
}

// NewPerformanceMonitor 创建新的性能监控器
func NewPerformanceMonitor(traderID string, st *store.Store) *PerformanceMonitor {
	return &PerformanceMonitor{
		traderID:           traderID,
		store:              st,
		collectionInterval: 5 * time.Minute, // 默认每 5 分钟收集一次
		lastMetricTime:     time.Now(),
	}
}

// CollectMetrics 收集当前性能指标
func (pm *PerformanceMonitor) CollectMetrics(
	winRate, profitFactor, totalPnL, dailyPnL float64,
	maxDrawdown, currentDrawdown, sharpeRatio float64,
	totalTrades, winningTrades, losingTrades int,
	openPositions int,
	totalEquity, availableBalance float64,
	volatilityMult, confidenceAdj float64,
) *store.PerformanceMetric {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	metric := &store.PerformanceMetric{
		ID:                   generateID("perf"),
		TraderID:             pm.traderID,
		Timestamp:            time.Now().UTC(),
		WinRate:              winRate,
		ProfitFactor:         profitFactor,
		TotalPnL:             totalPnL,
		DailyPnL:             dailyPnL,
		MaxDrawdown:          maxDrawdown,
		CurrentDrawdown:      currentDrawdown,
		SharpeRatio:          sharpeRatio,
		TotalTrades:          totalTrades,
		WinningTrades:        winningTrades,
		LosingTrades:         losingTrades,
		OpenPositions:        openPositions,
		TotalEquity:          totalEquity,
		AvailableBalance:     availableBalance,
		VolatilityMultiplier: volatilityMult,
		ConfidenceAdjustment: confidenceAdj,
		CreatedAt:            time.Now().UTC(),
		UpdatedAt:            time.Now().UTC(),
	}

	// 保存到数据库
	if pm.store != nil {
		if err := pm.store.GormDB().Create(metric).Error; err != nil {
			logger.Errorf("[%s] Failed to save performance metric: %v", pm.traderID, err)
		}
	}

	logger.Debugf("[%s] Performance metric collected: WR=%.2f%%, PF=%.2f, DD=%.2f%%",
		pm.traderID, winRate*100, profitFactor, maxDrawdown*100)

	return metric
}

// GetRecentMetrics 获取最近的性能指标
func (pm *PerformanceMonitor) GetRecentMetrics(limit int) []*store.PerformanceMetric {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.store == nil {
		return []*store.PerformanceMetric{}
	}

	var metrics []*store.PerformanceMetric
	if err := pm.store.GormDB().
		Where("trader_id = ?", pm.traderID).
		Order("timestamp DESC").
		Limit(limit).
		Find(&metrics).Error; err != nil {
		logger.Errorf("[%s] Failed to fetch recent metrics: %v", pm.traderID, err)
		return []*store.PerformanceMetric{}
	}

	return metrics
}

// AnalyzePerformanceTrend 分析性能趋势
func (pm *PerformanceMonitor) AnalyzePerformanceTrend(periodHours int) *PerformanceTrend {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.store == nil {
		return nil
	}

	startTime := time.Now().Add(-time.Duration(periodHours) * time.Hour)

	var metrics []*store.PerformanceMetric
	if err := pm.store.GormDB().
		Where("trader_id = ? AND timestamp >= ?", pm.traderID, startTime).
		Order("timestamp ASC").
		Find(&metrics).Error; err != nil {
		logger.Errorf("[%s] Failed to fetch metrics for trend analysis: %v", pm.traderID, err)
		return nil
	}

	if len(metrics) == 0 {
		return nil
	}

	// 计算趋势
	trend := &PerformanceTrend{
		PeriodStart: metrics[0].Timestamp,
		PeriodEnd:   metrics[len(metrics)-1].Timestamp,
		MetricCount: len(metrics),
	}

	// 计算平均值
	var totalWR, totalPF, totalDD, totalSharpe float64
	minWR, maxWR := metrics[0].WinRate, metrics[0].WinRate
	minDD, maxDD := metrics[0].MaxDrawdown, metrics[0].MaxDrawdown

	for _, m := range metrics {
		totalWR += m.WinRate
		totalPF += m.ProfitFactor
		totalDD += m.MaxDrawdown
		totalSharpe += m.SharpeRatio

		if m.WinRate < minWR {
			minWR = m.WinRate
		}
		if m.WinRate > maxWR {
			maxWR = m.WinRate
		}
		if m.MaxDrawdown < minDD {
			minDD = m.MaxDrawdown
		}
		if m.MaxDrawdown > maxDD {
			maxDD = m.MaxDrawdown
		}
	}

	count := float64(len(metrics))
	trend.AvgWinRate = totalWR / count
	trend.AvgProfitFactor = totalPF / count
	trend.AvgDrawdown = totalDD / count
	trend.AvgSharpeRatio = totalSharpe / count
	trend.MinWinRate = minWR
	trend.MaxWinRate = maxWR
	trend.MinDrawdown = minDD
	trend.MaxDrawdown = maxDD

	// 判断趋势方向
	if len(metrics) >= 2 {
		// 比较前后的胜率
		firstHalf := metrics[0 : len(metrics)/2]
		secondHalf := metrics[len(metrics)/2:]

		firstAvgWR := 0.0
		secondAvgWR := 0.0
		for _, m := range firstHalf {
			firstAvgWR += m.WinRate
		}
		for _, m := range secondHalf {
			secondAvgWR += m.WinRate
		}
		firstAvgWR /= float64(len(firstHalf))
		secondAvgWR /= float64(len(secondHalf))

		if secondAvgWR > firstAvgWR {
			trend.WinRateTrend = "up"
		} else if secondAvgWR < firstAvgWR {
			trend.WinRateTrend = "down"
		} else {
			trend.WinRateTrend = "stable"
		}

		// 判断回撤趋势
		firstAvgDD := 0.0
		secondAvgDD := 0.0
		for _, m := range firstHalf {
			firstAvgDD += m.MaxDrawdown
		}
		for _, m := range secondHalf {
			secondAvgDD += m.MaxDrawdown
		}
		firstAvgDD /= float64(len(firstHalf))
		secondAvgDD /= float64(len(secondHalf))

		if secondAvgDD < firstAvgDD {
			trend.DrawdownTrend = "down" // 回撤减少是好的
		} else if secondAvgDD > firstAvgDD {
			trend.DrawdownTrend = "up"
		} else {
			trend.DrawdownTrend = "stable"
		}
	}

	return trend
}

// PerformanceTrend 性能趋势分析结果
type PerformanceTrend struct {
	PeriodStart     time.Time
	PeriodEnd       time.Time
	MetricCount     int
	AvgWinRate      float64
	AvgProfitFactor float64
	AvgDrawdown     float64
	AvgSharpeRatio  float64
	MinWinRate      float64
	MaxWinRate      float64
	MinDrawdown     float64
	MaxDrawdown     float64
	WinRateTrend    string // up, down, stable
	DrawdownTrend   string // up, down, stable
}

// ====== Alert Manager ======

// AlertManager 告警管理器
type AlertManager struct {
	mu       sync.RWMutex
	traderID string
	store    *store.Store
	rules    map[string]*store.AlertRule
	alerts   []*store.Alert
}

// NewAlertManager 创建新的告警管理器
func NewAlertManager(traderID string, st *store.Store) *AlertManager {
	am := &AlertManager{
		traderID: traderID,
		store:    st,
		rules:    make(map[string]*store.AlertRule),
		alerts:   make([]*store.Alert, 0),
	}

	// 加载已有的规则
	if st != nil {
		am.loadRules()
	}

	return am
}

// loadRules 从数据库加载告警规则
func (am *AlertManager) loadRules() {
	var rules []*store.AlertRule
	if err := am.store.GormDB().
		Where("trader_id = ? AND enabled = ?", am.traderID, true).
		Find(&rules).Error; err != nil {
		logger.Errorf("[%s] Failed to load alert rules: %v", am.traderID, err)
		return
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	for _, rule := range rules {
		am.rules[rule.ID] = rule
	}

	logger.Infof("[%s] Loaded %d alert rules", am.traderID, len(am.rules))
}

// CreateRule 创建新的告警规则
func (am *AlertManager) CreateRule(
	name, description string,
	metricType, operator string,
	threshold float64,
	duration int,
	severity string,
) *store.AlertRule {
	am.mu.Lock()
	defer am.mu.Unlock()

	rule := &store.AlertRule{
		ID:          generateID("rule"),
		TraderID:    am.traderID,
		Name:        name,
		Description: description,
		MetricType:  metricType,
		Operator:    operator,
		Threshold:   threshold,
		Duration:    duration,
		Enabled:     true,
		Severity:    severity,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if am.store != nil {
		if err := am.store.GormDB().Create(rule).Error; err != nil {
			logger.Errorf("[%s] Failed to create alert rule: %v", am.traderID, err)
			return nil
		}
	}

	am.rules[rule.ID] = rule
	logger.Infof("[%s] Alert rule created: %s", am.traderID, name)

	return rule
}

// CheckAlert 检查是否需要触发告警
func (am *AlertManager) CheckAlert(metricType string, value float64) *store.Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	for _, rule := range am.rules {
		if rule.MetricType != metricType {
			continue
		}

		// 检查条件
		triggered := false
		switch rule.Operator {
		case ">":
			triggered = value > rule.Threshold
		case "<":
			triggered = value < rule.Threshold
		case ">=":
			triggered = value >= rule.Threshold
		case "<=":
			triggered = value <= rule.Threshold
		case "==":
			triggered = math.Abs(value-rule.Threshold) < 0.0001
		}

		if !triggered {
			continue
		}

		// 创建告警实例
		alert := &store.Alert{
			ID:          generateID("alert"),
			AlertRuleID: rule.ID,
			TraderID:    am.traderID,
			MetricType:  metricType,
			MetricValue: value,
			Threshold:   rule.Threshold,
			Status:      "triggered",
			Severity:    rule.Severity,
			Message:     fmt.Sprintf("%s: %s %.4f (threshold: %.4f)", rule.Name, rule.Operator, value, rule.Threshold),
			TriggeredAt: time.Now().UTC(),
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		// 保存告警
		if am.store != nil {
			if err := am.store.GormDB().Create(alert).Error; err != nil {
				logger.Errorf("[%s] Failed to save alert: %v", am.traderID, err)
			}
		}

		logger.Warnf("[%s] ALERT TRIGGERED: %s (severity: %s)", am.traderID, alert.Message, rule.Severity)

		return alert
	}

	return nil
}

// GetActiveAlerts 获取活跃的告警
func (am *AlertManager) GetActiveAlerts() []*store.Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if am.store == nil {
		return []*store.Alert{}
	}

	var alerts []*store.Alert
	if err := am.store.GormDB().
		Where("trader_id = ? AND status IN ?", am.traderID, []string{"triggered", "acknowledged"}).
		Order("triggered_at DESC").
		Find(&alerts).Error; err != nil {
		logger.Errorf("[%s] Failed to fetch active alerts: %v", am.traderID, err)
		return []*store.Alert{}
	}

	return alerts
}

// AcknowledgeAlert 确认告警
func (am *AlertManager) AcknowledgeAlert(alertID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now().UTC()
	if err := am.store.GormDB().
		Model(&store.Alert{}).
		Where("id = ?", alertID).
		Updates(map[string]interface{}{
			"status":          "acknowledged",
			"acknowledged_at": now,
			"updated_at":      now,
		}).Error; err != nil {
		return err
	}

	logger.Infof("[%s] Alert acknowledged: %s", am.traderID, alertID)
	return nil
}

// ResolveAlert 解决告警
func (am *AlertManager) ResolveAlert(alertID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now().UTC()
	if err := am.store.GormDB().
		Model(&store.Alert{}).
		Where("id = ?", alertID).
		Updates(map[string]interface{}{
			"status":      "resolved",
			"resolved_at": now,
			"updated_at":  now,
		}).Error; err != nil {
		return err
	}

	logger.Infof("[%s] Alert resolved: %s", am.traderID, alertID)
	return nil
}

// ====== Health Checker ======

// HealthChecker 系统健康检查器
type HealthChecker struct {
	mu       sync.RWMutex
	traderID string
	store    *store.Store

	lastHealth *store.SystemHealth
	startTime  time.Time
}

// NewHealthChecker 创建新的健康检查器
func NewHealthChecker(traderID string, st *store.Store) *HealthChecker {
	return &HealthChecker{
		traderID:  traderID,
		store:     st,
		startTime: time.Now(),
	}
}

// Check 执行健康检查
func (hc *HealthChecker) Check(
	exchangeConnected, databaseConnected, apiHealthy bool,
	apiLatency, dbLatency, memUsage, cpuUsage float64,
) *store.SystemHealth {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	status := "healthy"
	reason := "All systems operational"

	// 判断健康状态
	issueCount := 0
	if !exchangeConnected {
		issueCount++
		reason += "; Exchange disconnected"
	}
	if !databaseConnected {
		issueCount++
		reason += "; Database disconnected"
	}
	if !apiHealthy {
		issueCount++
		reason += "; API unhealthy"
	}
	if apiLatency > 5000 { // > 5 秒
		issueCount++
		reason += fmt.Sprintf("; High API latency (%.0fms)", apiLatency)
	}
	if memUsage > 80 {
		issueCount++
		reason += fmt.Sprintf("; High memory usage (%.1f%%)", memUsage)
	}
	if cpuUsage > 80 {
		issueCount++
		reason += fmt.Sprintf("; High CPU usage (%.1f%%)", cpuUsage)
	}

	if issueCount >= 3 {
		status = "unhealthy"
	} else if issueCount > 0 {
		status = "degraded"
	}

	health := &store.SystemHealth{
		ID:                generateID("health"),
		TraderID:          hc.traderID,
		ExchangeConnected: exchangeConnected,
		DatabaseConnected: databaseConnected,
		APIHealthy:        apiHealthy,
		APILatency:        apiLatency,
		DatabaseLatency:   dbLatency,
		MemoryUsage:       memUsage,
		CPUUsage:          cpuUsage,
		LastAPICheck:      time.Now().UTC(),
		LastDBCheck:       time.Now().UTC(),
		Status:            status,
		StatusReason:      reason,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}

	// 保存健康状态
	if hc.store != nil {
		if err := hc.store.GormDB().Create(health).Error; err != nil {
			logger.Errorf("[%s] Failed to save health check: %v", hc.traderID, err)
		}
	}

	hc.lastHealth = health

	if status != "healthy" {
		logger.Warnf("[%s] System health: %s - %s", hc.traderID, status, reason)
	}

	return health
}

// GetLatestHealth 获取最新的健康状态
func (hc *HealthChecker) GetLatestHealth() *store.SystemHealth {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	if hc.lastHealth != nil {
		return hc.lastHealth
	}

	if hc.store == nil {
		return nil
	}

	var health *store.SystemHealth
	if err := hc.store.GormDB().
		Where("trader_id = ?", hc.traderID).
		Order("created_at DESC").
		First(&health).Error; err != nil {
		return nil
	}

	return health
}

// ====== Monitoring Coordinator ======

// MonitoringCoordinator 监控协调器
type MonitoringCoordinator struct {
	mu                 sync.RWMutex
	traderID           string
	store              *store.Store
	performanceMonitor *PerformanceMonitor
	alertManager       *AlertManager
	healthChecker      *HealthChecker
	isRunning          bool
	stopChan           chan struct{}
}

// NewMonitoringCoordinator 创建新的监控协调器
func NewMonitoringCoordinator(traderID string, st *store.Store) *MonitoringCoordinator {
	return &MonitoringCoordinator{
		traderID:           traderID,
		store:              st,
		performanceMonitor: NewPerformanceMonitor(traderID, st),
		alertManager:       NewAlertManager(traderID, st),
		healthChecker:      NewHealthChecker(traderID, st),
		stopChan:           make(chan struct{}),
	}
}

// Start 启动监控
func (mc *MonitoringCoordinator) Start(collectionInterval time.Duration) error {
	mc.mu.Lock()
	if mc.isRunning {
		mc.mu.Unlock()
		return fmt.Errorf("monitoring already running")
	}
	mc.isRunning = true
	mc.mu.Unlock()

	logger.Infof("[%s] Performance monitoring started (interval: %v)", mc.traderID, collectionInterval)

	return nil
}

// Stop 停止监控
func (mc *MonitoringCoordinator) Stop() {
	mc.mu.Lock()
	if !mc.isRunning {
		mc.mu.Unlock()
		return
	}
	mc.isRunning = false
	mc.mu.Unlock()

	close(mc.stopChan)
	logger.Infof("[%s] Performance monitoring stopped", mc.traderID)
}

// IsRunning 检查监控是否运行
func (mc *MonitoringCoordinator) IsRunning() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.isRunning
}

// GetPerformanceMonitor 获取性能监控器
func (mc *MonitoringCoordinator) GetPerformanceMonitor() *PerformanceMonitor {
	return mc.performanceMonitor
}

// GetAlertManager 获取告警管理器
func (mc *MonitoringCoordinator) GetAlertManager() *AlertManager {
	return mc.alertManager
}

// GetHealthChecker 获取健康检查器
func (mc *MonitoringCoordinator) GetHealthChecker() *HealthChecker {
	return mc.healthChecker
}

// generateID 生成唯一 ID
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixNano(), time.Now().Nanosecond())
}
