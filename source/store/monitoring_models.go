package store

import "time"

// ====== Performance Monitoring Models ======

// PerformanceMetric 交易性能指标
type PerformanceMetric struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	TraderID  string    `json:"trader_id" gorm:"index"`
	Timestamp time.Time `json:"timestamp" gorm:"index"`

	// 交易指标
	WinRate      float64 `json:"win_rate"`      // 胜率 (0-1)
	ProfitFactor float64 `json:"profit_factor"` // 盈利因子
	TotalPnL     float64 `json:"total_pnl"`     // 总损益
	DailyPnL     float64 `json:"daily_pnl"`     // 日损益

	// 风险指标
	MaxDrawdown     float64 `json:"max_drawdown"`     // 最大回撤 (0-1)
	CurrentDrawdown float64 `json:"current_drawdown"` // 当前回撤
	SharpeRatio     float64 `json:"sharpe_ratio"`     // Sharpe 比率

	// 交易统计
	TotalTrades   int `json:"total_trades"`   // 总交易数
	WinningTrades int `json:"winning_trades"` // 胜利交易数
	LosingTrades  int `json:"losing_trades"`  // 亏损交易数

	// 持仓情况
	OpenPositions    int     `json:"open_positions"`    // 开放头寸数
	TotalEquity      float64 `json:"total_equity"`      // 总权益
	AvailableBalance float64 `json:"available_balance"` // 可用余额

	// 参数优化指标
	VolatilityMultiplier float64 `json:"volatility_multiplier"` // 波动率倍数
	ConfidenceAdjustment float64 `json:"confidence_adjustment"` // 置信度调整

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AlertRule 告警规则
type AlertRule struct {
	ID          string `json:"id" gorm:"primaryKey"`
	TraderID    string `json:"trader_id" gorm:"index"`
	Name        string `json:"name"`
	Description string `json:"description"`

	// 告警条件
	MetricType string  `json:"metric_type"` // 指标类型（win_rate, drawdown, pnl 等）
	Operator   string  `json:"operator"`    // 比较操作符 (>, <, >=, <=, ==)
	Threshold  float64 `json:"threshold"`   // 阈值
	Duration   int     `json:"duration"`    // 持续时间（秒）

	// 告警配置
	Enabled     bool   `json:"enabled"`
	Severity    string `json:"severity"`     // 严重程度 (info, warning, critical)
	NotifyEmail string `json:"notify_email"` // 通知邮箱
	WebhookURL  string `json:"webhook_url"`  // Webhook 地址

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Alert 告警实例
type Alert struct {
	ID          string `json:"id" gorm:"primaryKey"`
	AlertRuleID string `json:"alert_rule_id" gorm:"index"`
	TraderID    string `json:"trader_id" gorm:"index"`

	MetricType  string  `json:"metric_type"`
	MetricValue float64 `json:"metric_value"`
	Threshold   float64 `json:"threshold"`

	Status   string `json:"status"` // triggered, acknowledged, resolved
	Severity string `json:"severity"`
	Message  string `json:"message"`

	TriggeredAt    time.Time  `json:"triggered_at" gorm:"index"`
	AcknowledgedAt *time.Time `json:"acknowledged_at"`
	ResolvedAt     *time.Time `json:"resolved_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SystemHealth 系统健康状态
type SystemHealth struct {
	ID       string `json:"id" gorm:"primaryKey"`
	TraderID string `json:"trader_id" gorm:"index"`

	// 连接状态
	ExchangeConnected bool `json:"exchange_connected"`
	DatabaseConnected bool `json:"database_connected"`
	APIHealthy        bool `json:"api_healthy"`

	// 性能指标
	APILatency      float64 `json:"api_latency_ms"`      // API 延迟（毫秒）
	DatabaseLatency float64 `json:"database_latency_ms"` // 数据库延迟
	MemoryUsage     float64 `json:"memory_usage_mb"`     // 内存使用（MB）
	CPUUsage        float64 `json:"cpu_usage_percent"`   // CPU 使用率

	// 最后检查时间
	LastAPICheck time.Time `json:"last_api_check"`
	LastDBCheck  time.Time `json:"last_database_check"`

	Status       string `json:"status"` // healthy, degraded, unhealthy
	StatusReason string `json:"status_reason"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MetricsAggregation 指标聚合
type MetricsAggregation struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	TraderID    string    `json:"trader_id" gorm:"index"`
	PeriodType  string    `json:"period_type"` // hourly, daily, weekly, monthly
	PeriodStart time.Time `json:"period_start" gorm:"index"`
	PeriodEnd   time.Time `json:"period_end"`

	// 聚合的统计数据
	AvgWinRate      float64 `json:"avg_win_rate"`
	AvgDrawdown     float64 `json:"avg_drawdown"`
	AvgProfitFactor float64 `json:"avg_profit_factor"`

	TotalPnL     float64 `json:"total_pnl"`
	TotalTrades  int     `json:"total_trades"`
	PeakEquity   float64 `json:"peak_equity"`
	LowestEquity float64 `json:"lowest_equity"`

	// 趋势指标
	WinRateTrend  string `json:"win_rate_trend"` // up, down, stable
	DrawdownTrend string `json:"drawdown_trend"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MonitoringSession 监控会话
type MonitoringSession struct {
	ID       string `json:"id" gorm:"primaryKey"`
	TraderID string `json:"trader_id" gorm:"index"`

	SessionStartTime time.Time  `json:"session_start_time"`
	SessionEndTime   *time.Time `json:"session_end_time"`

	Status string `json:"status"` // active, paused, stopped

	TotalMetricsCollected int `json:"total_metrics_collected"`
	TotalAlertsTriggered  int `json:"total_alerts_triggered"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
