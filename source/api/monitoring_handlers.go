package api

import (
	"net/http"
	"nofx/backtest"
	"nofx/logger"
	"nofx/store"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MonitoringHandler 监控 API 处理器
type MonitoringHandler struct {
	store              *store.Store
	monitoringServices map[string]*backtest.MonitoringCoordinator
}

// NewMonitoringHandler 创建新的监控处理器
func NewMonitoringHandler(st *store.Store) *MonitoringHandler {
	return &MonitoringHandler{
		store:              st,
		monitoringServices: make(map[string]*backtest.MonitoringCoordinator),
	}
}

// RegisterMonitoringRoutes 注册监控路由
func RegisterMonitoringRoutes(router *gin.Engine, st *store.Store) {
	h := NewMonitoringHandler(st)

	// 性能指标路由
	router.GET("/api/monitoring/:traderID/metrics", h.GetMetrics)
	router.GET("/api/monitoring/:traderID/metrics/latest", h.GetLatestMetric)
	router.GET("/api/monitoring/:traderID/metrics/trend", h.GetMetricsTrend)
	router.POST("/api/monitoring/:traderID/metrics/collect", h.CollectMetrics)

	// 告警路由
	router.GET("/api/monitoring/:traderID/alerts", h.GetAlerts)
	router.GET("/api/monitoring/:traderID/alerts/active", h.GetActiveAlerts)
	router.GET("/api/monitoring/alerts/:alertID", h.GetAlertByID)
	router.POST("/api/monitoring/alerts/:alertID/acknowledge", h.AcknowledgeAlert)
	router.POST("/api/monitoring/alerts/:alertID/resolve", h.ResolveAlert)
	router.GET("/api/monitoring/:traderID/alert-rules", h.GetAlertRules)
	router.POST("/api/monitoring/:traderID/alert-rules", h.CreateAlertRule)
	router.PUT("/api/monitoring/alert-rules/:ruleID", h.UpdateAlertRule)
	router.DELETE("/api/monitoring/alert-rules/:ruleID", h.DeleteAlertRule)

	// 健康检查路由
	router.GET("/api/monitoring/:traderID/health", h.GetHealthStatus)
	router.GET("/api/monitoring/:traderID/health/history", h.GetHealthHistory)
	router.POST("/api/monitoring/:traderID/health/check", h.PerformHealthCheck)

	// 统计和汇总路由
	router.GET("/api/monitoring/:traderID/summary", h.GetMonitoringSummary)
	router.GET("/api/monitoring/:traderID/statistics", h.GetMonitoringStatistics)

	logger.Infof("Monitoring routes registered")
}

// ====== Performance Metrics Handlers ======

// GetMetrics 获取性能指标
func (h *MonitoringHandler) GetMetrics(c *gin.Context) {
	traderID := c.Param("traderID")
	limitStr := c.DefaultQuery("limit", "100")
	limit, _ := strconv.Atoi(limitStr)

	repo := store.NewMonitoringRepository(h.store)
	metrics, err := repo.GetMetricsByTraderID(traderID, limit)
	if err != nil {
		logger.Errorf("Failed to get metrics: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get metrics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": map[string]interface{}{
			"count":   len(metrics),
			"metrics": metrics,
		},
		"status": "success",
	})
}

// GetLatestMetric 获取最新的性能指标
func (h *MonitoringHandler) GetLatestMetric(c *gin.Context) {
	traderID := c.Param("traderID")

	repo := store.NewMonitoringRepository(h.store)
	metric, err := repo.GetLatestMetric(traderID)
	if err != nil {
		logger.Errorf("Failed to get latest metric: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "No metrics found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": metric, "status": "success"})
}

// GetMetricsTrend 获取性能趋势
func (h *MonitoringHandler) GetMetricsTrend(c *gin.Context) {
	traderID := c.Param("traderID")
	hoursStr := c.DefaultQuery("hours", "24")
	hours, _ := strconv.Atoi(hoursStr)

	// 获取或创建监控协调器
	monitor := h.getOrCreateMonitoringCoordinator(traderID)
	trend := monitor.GetPerformanceMonitor().AnalyzePerformanceTrend(hours)

	if trend == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No trend data available"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": trend, "status": "success"})
}

// CollectMetricsRequest 收集性能指标请求
type CollectMetricsRequest struct {
	WinRate              float64 `json:"win_rate"`
	ProfitFactor         float64 `json:"profit_factor"`
	TotalPnL             float64 `json:"total_pnl"`
	DailyPnL             float64 `json:"daily_pnl"`
	MaxDrawdown          float64 `json:"max_drawdown"`
	CurrentDrawdown      float64 `json:"current_drawdown"`
	SharpeRatio          float64 `json:"sharpe_ratio"`
	TotalTrades          int     `json:"total_trades"`
	WinningTrades        int     `json:"winning_trades"`
	LosingTrades         int     `json:"losing_trades"`
	OpenPositions        int     `json:"open_positions"`
	TotalEquity          float64 `json:"total_equity"`
	AvailableBalance     float64 `json:"available_balance"`
	VolatilityMultiplier float64 `json:"volatility_multiplier"`
	ConfidenceAdjustment float64 `json:"confidence_adjustment"`
}

// CollectMetrics 收集性能指标
func (h *MonitoringHandler) CollectMetrics(c *gin.Context) {
	traderID := c.Param("traderID")

	var req CollectMetricsRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	monitor := h.getOrCreateMonitoringCoordinator(traderID)
	metric := monitor.GetPerformanceMonitor().CollectMetrics(
		req.WinRate, req.ProfitFactor, req.TotalPnL, req.DailyPnL,
		req.MaxDrawdown, req.CurrentDrawdown, req.SharpeRatio,
		req.TotalTrades, req.WinningTrades, req.LosingTrades,
		req.OpenPositions, req.TotalEquity, req.AvailableBalance,
		req.VolatilityMultiplier, req.ConfidenceAdjustment,
	)

	c.JSON(http.StatusOK, gin.H{"data": metric, "status": "success"})
}

// ====== Alert Handlers ======

// GetAlerts 获取告警列表
func (h *MonitoringHandler) GetAlerts(c *gin.Context) {
	traderID := c.Param("traderID")
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	repo := store.NewMonitoringRepository(h.store)
	alerts, err := repo.GetAlertsByTraderID(traderID, limit)
	if err != nil {
		logger.Errorf("Failed to get alerts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get alerts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": map[string]interface{}{
			"count":  len(alerts),
			"alerts": alerts,
		},
		"status": "success",
	})
}

// GetActiveAlerts 获取活跃的告警
func (h *MonitoringHandler) GetActiveAlerts(c *gin.Context) {
	traderID := c.Param("traderID")

	repo := store.NewMonitoringRepository(h.store)
	alerts, err := repo.GetActiveAlerts(traderID)
	if err != nil {
		logger.Errorf("Failed to get active alerts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get active alerts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": map[string]interface{}{
			"count":  len(alerts),
			"alerts": alerts,
		},
		"status": "success",
	})
}

// GetAlertByID 根据 ID 获取告警
func (h *MonitoringHandler) GetAlertByID(c *gin.Context) {
	alertID := c.Param("alertID")

	repo := store.NewMonitoringRepository(h.store)
	alert, err := repo.GetAlertByID(alertID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": alert, "status": "success"})
}

// AcknowledgeAlert 确认告警
func (h *MonitoringHandler) AcknowledgeAlert(c *gin.Context) {
	alertID := c.Param("alertID")

	repo := store.NewMonitoringRepository(h.store)
	if err := repo.UpdateAlertStatus(alertID, "acknowledged"); err != nil {
		logger.Errorf("Failed to acknowledge alert: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to acknowledge alert"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Alert acknowledged"})
}

// ResolveAlert 解决告警
func (h *MonitoringHandler) ResolveAlert(c *gin.Context) {
	alertID := c.Param("alertID")

	repo := store.NewMonitoringRepository(h.store)
	if err := repo.UpdateAlertStatus(alertID, "resolved"); err != nil {
		logger.Errorf("Failed to resolve alert: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve alert"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Alert resolved"})
}

// GetAlertRules 获取告警规则列表
func (h *MonitoringHandler) GetAlertRules(c *gin.Context) {
	traderID := c.Param("traderID")

	repo := store.NewMonitoringRepository(h.store)
	rules, err := repo.GetAlertRulesByTraderID(traderID)
	if err != nil {
		logger.Errorf("Failed to get alert rules: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get alert rules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": map[string]interface{}{
			"count": len(rules),
			"rules": rules,
		},
		"status": "success",
	})
}

// CreateAlertRuleRequest 创建告警规则请求
type CreateAlertRuleRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	MetricType  string  `json:"metric_type" binding:"required"`
	Operator    string  `json:"operator" binding:"required"`
	Threshold   float64 `json:"threshold" binding:"required"`
	Duration    int     `json:"duration"`
	Severity    string  `json:"severity" binding:"required"`
}

// CreateAlertRule 创建告警规则
func (h *MonitoringHandler) CreateAlertRule(c *gin.Context) {
	traderID := c.Param("traderID")

	var req CreateAlertRuleRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	rule := &store.AlertRule{
		ID:          generateMonitoringID("rule"),
		TraderID:    traderID,
		Name:        req.Name,
		Description: req.Description,
		MetricType:  req.MetricType,
		Operator:    req.Operator,
		Threshold:   req.Threshold,
		Duration:    req.Duration,
		Severity:    req.Severity,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	repo := store.NewMonitoringRepository(h.store)
	if err := repo.CreateAlertRule(rule); err != nil {
		logger.Errorf("Failed to create alert rule: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create alert rule"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": rule, "status": "success"})
}

// UpdateAlertRule 更新告警规则
func (h *MonitoringHandler) UpdateAlertRule(c *gin.Context) {
	ruleID := c.Param("ruleID")

	var updates map[string]interface{}
	if err := c.BindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	repo := store.NewMonitoringRepository(h.store)
	if err := repo.UpdateAlertRule(ruleID, updates); err != nil {
		logger.Errorf("Failed to update alert rule: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update alert rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Alert rule updated"})
}

// DeleteAlertRule 删除告警规则
func (h *MonitoringHandler) DeleteAlertRule(c *gin.Context) {
	ruleID := c.Param("ruleID")

	repo := store.NewMonitoringRepository(h.store)
	if err := repo.DeleteAlertRule(ruleID); err != nil {
		logger.Errorf("Failed to delete alert rule: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete alert rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Alert rule deleted"})
}

// ====== Health Check Handlers ======

// GetHealthStatus 获取系统健康状态
func (h *MonitoringHandler) GetHealthStatus(c *gin.Context) {
	traderID := c.Param("traderID")

	repo := store.NewMonitoringRepository(h.store)
	health, err := repo.GetLatestHealthCheck(traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No health check data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": health, "status": "success"})
}

// GetHealthHistory 获取健康状态历史
func (h *MonitoringHandler) GetHealthHistory(c *gin.Context) {
	traderID := c.Param("traderID")
	hoursStr := c.DefaultQuery("hours", "24")
	hours, _ := strconv.Atoi(hoursStr)

	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)
	endTime := time.Now()

	repo := store.NewMonitoringRepository(h.store)
	checks, err := repo.GetHealthChecksByTimeRange(traderID, startTime, endTime)
	if err != nil {
		logger.Errorf("Failed to get health history: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get health history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": map[string]interface{}{
			"count":  len(checks),
			"checks": checks,
		},
		"status": "success",
	})
}

// HealthCheckRequest 健康检查请求
type HealthCheckRequest struct {
	ExchangeConnected bool    `json:"exchange_connected"`
	DatabaseConnected bool    `json:"database_connected"`
	APIHealthy        bool    `json:"api_healthy"`
	APILatency        float64 `json:"api_latency_ms"`
	DatabaseLatency   float64 `json:"database_latency_ms"`
	MemoryUsage       float64 `json:"memory_usage_mb"`
	CPUUsage          float64 `json:"cpu_usage_percent"`
}

// PerformHealthCheck 执行健康检查
func (h *MonitoringHandler) PerformHealthCheck(c *gin.Context) {
	traderID := c.Param("traderID")

	var req HealthCheckRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	monitor := h.getOrCreateMonitoringCoordinator(traderID)
	health := monitor.GetHealthChecker().Check(
		req.ExchangeConnected, req.DatabaseConnected, req.APIHealthy,
		req.APILatency, req.DatabaseLatency, req.MemoryUsage, req.CPUUsage,
	)

	c.JSON(http.StatusOK, gin.H{"data": health, "status": "success"})
}

// ====== Statistics Handlers ======

// GetMonitoringSummary 获取监控摘要
func (h *MonitoringHandler) GetMonitoringSummary(c *gin.Context) {
	traderID := c.Param("traderID")
	hoursStr := c.DefaultQuery("hours", "24")
	hours, _ := strconv.Atoi(hoursStr)

	repo := store.NewMonitoringRepository(h.store)
	summary, err := repo.GetMetricsSummary(traderID, hours)
	if err != nil {
		logger.Errorf("Failed to get metrics summary: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get summary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": summary, "status": "success"})
}

// GetMonitoringStatistics 获取监控统计信息
func (h *MonitoringHandler) GetMonitoringStatistics(c *gin.Context) {
	traderID := c.Param("traderID")

	repo := store.NewMonitoringRepository(h.store)

	// 获取告警统计
	alertStats, err := repo.GetAlertStats(traderID)
	if err != nil {
		logger.Errorf("Failed to get alert stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get statistics"})
		return
	}

	// 获取指标摘要
	summary, err := repo.GetMetricsSummary(traderID, 24)
	if err != nil {
		logger.Errorf("Failed to get metrics summary: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get statistics"})
		return
	}

	stats := map[string]interface{}{
		"alert_statistics": alertStats,
		"metrics_summary":  summary,
	}

	c.JSON(http.StatusOK, gin.H{"data": stats, "status": "success"})
}

// Helper Methods

// getOrCreateMonitoringCoordinator 获取或创建监控协调器
func (h *MonitoringHandler) getOrCreateMonitoringCoordinator(traderID string) *backtest.MonitoringCoordinator {
	if monitor, exists := h.monitoringServices[traderID]; exists {
		return monitor
	}

	monitor := backtest.NewMonitoringCoordinator(traderID, h.store)
	h.monitoringServices[traderID] = monitor

	return monitor
}

// generateMonitoringID 生成监控系统的唯一 ID
func generateMonitoringID(prefix string) string {
	return prefix + "_" + uuid.New().String()
}
