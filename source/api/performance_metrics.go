package api

import (
	"sync"
	"time"
)

// TrailingStopPerformanceMetric 动态止损单次性能指标
type TrailingStopPerformanceMetric struct {
	Timestamp      time.Time `json:"timestamp"`        // 时间戳
	Symbol         string    `json:"symbol"`           // 交易对
	TraderID       string    `json:"trader_id"`        // 交易者ID
	EntryPrice     float64   `json:"entry_price"`      // 入场价格
	PeakPrice      float64   `json:"peak_price"`       // 最高价格
	LowestPrice    float64   `json:"lowest_price"`     // 最低价格
	InitialSL      float64   `json:"initial_sl"`       // 初始止损
	FinalSL        float64   `json:"final_sl"`         // 最终止损
	SLMoveCount    int       `json:"sl_move_count"`    // 止损移动次数
	GainedPips     float64   `json:"gained_pips"`      // 通过跟踪获得的点数
	PipsBeforeTrig float64   `json:"pips_before_trig"` // 触发前的点数
	PipsAfterTrig  float64   `json:"pips_after_trig"`  // 触发后的点数
	AvoidedLoss    float64   `json:"avoided_loss"`     // 避免的损失 (%)
	TrailingRate   float64   `json:"trailing_rate"`    // 跟踪比率
	Duration       string    `json:"duration"`         // 持仓时长
	Status         string    `json:"status"`           // 状态: triggered/closed/active
}

// DailyPerformanceSummary 日度性能汇总
type DailyPerformanceSummary struct {
	Date                 string  `json:"date"`                   // 日期 (YYYY-MM-DD)
	TotalTradeCount      int     `json:"total_trade_count"`      // 总交易数
	TrailingTriggered    int     `json:"trailing_triggered"`     // 止损触发次数
	AverageSLMoves       float64 `json:"average_sl_moves"`       // 平均止损移动次数
	TotalGainedPips      float64 `json:"total_gained_pips"`      // 总获得点数
	TotalAvoidedLoss     float64 `json:"total_avoided_loss"`     // 总避免损失 (%)
	BestSLPerformance    float64 `json:"best_sl_performance"`    // 最佳止损效果
	WorstSLPerformance   float64 `json:"worst_sl_performance"`   // 最差止损效果
	EffectiveTradeCount  int     `json:"effective_trade_count"`  // 有效交易数 (被止损触发的)
	EffectiveRate        float64 `json:"effective_rate"`         // 有效率 (%)
	AveragePipsProtected float64 `json:"average_pips_protected"` // 平均保护点数
}

// HourlyPerformanceTrend 小时级性能趋势
type HourlyPerformanceTrend struct {
	Hour              int     `json:"hour"`              // 小时 (0-23)
	TradeCount        int     `json:"trade_count"`       // 交易数
	TotalGainedPips   float64 `json:"total_gained_pips"` // 总获得点数
	AvgSLMovePerTrade float64 `json:"avg_sl_move_count"` // 平均SL移动次数
	SuccessRate       float64 `json:"success_rate"`      // 成功率 (%)
}

// PerformanceAlertEvent 性能告警事件
type PerformanceAlertEvent struct {
	Timestamp   time.Time `json:"timestamp"`    // 告警时间
	AlertType   string    `json:"alert_type"`   // 告警类型
	Severity    string    `json:"severity"`     // 严重程度: low/medium/high/critical
	TraderID    string    `json:"trader_id"`    // 交易者ID
	Symbol      string    `json:"symbol"`       // 交易对
	Message     string    `json:"message"`      // 告警消息
	MetricValue float64   `json:"metric_value"` // 指标值
	Threshold   float64   `json:"threshold"`    // 阈值
	Action      string    `json:"action"`       // 建议行动
	Resolved    bool      `json:"resolved"`     // 是否已解决
}

// PerformanceMetricsCollector 性能指标收集器
type PerformanceMetricsCollector struct {
	mu                    sync.RWMutex
	metrics               []TrailingStopPerformanceMetric
	dailySummaries        map[string]*DailyPerformanceSummary
	hourlyTrends          []HourlyPerformanceTrend
	alerts                []PerformanceAlertEvent
	maxMetricsCount       int
	performanceThresholds PerformanceThresholds
	lastHourUpdate        time.Time
}

// PerformanceThresholds 性能告警阈值
type PerformanceThresholds struct {
	MinSLMoveCountPerDay    int     // 最小止损移动次数 (每天)
	MinAvoidedLossPercent   float64 // 最小避免损失百分比
	MaxConsecutiveFailures  int     // 最大连续失败次数
	MinEffectiveRate        float64 // 最小有效率
	AbnormalSLMovementCount int     // 异常止损移动次数阈值
}

var performanceCollector = &PerformanceMetricsCollector{
	metrics:         make([]TrailingStopPerformanceMetric, 0, 10000),
	dailySummaries:  make(map[string]*DailyPerformanceSummary),
	hourlyTrends:    make([]HourlyPerformanceTrend, 24),
	alerts:          make([]PerformanceAlertEvent, 0, 1000),
	maxMetricsCount: 10000,
	lastHourUpdate:  time.Now(),
	performanceThresholds: PerformanceThresholds{
		MinSLMoveCountPerDay:    5,    // 每天至少5次止损移动
		MinAvoidedLossPercent:   0.5,  // 至少避免0.5%的损失
		MaxConsecutiveFailures:  3,    // 最多3次连续失败
		MinEffectiveRate:        60.0, // 最小有效率60%
		AbnormalSLMovementCount: 20,   // 异常：单笔20次以上止损移动
	},
}

// RecordTrailingStopMetric 记录止损性能指标
func RecordTrailingStopMetric(metric TrailingStopPerformanceMetric) {
	performanceCollector.mu.Lock()
	defer performanceCollector.mu.Unlock()

	metric.Timestamp = time.Now()

	// 添加到指标列表
	performanceCollector.metrics = append(performanceCollector.metrics, metric)

	// 限制历史记录大小
	if len(performanceCollector.metrics) > performanceCollector.maxMetricsCount {
		performanceCollector.metrics = performanceCollector.metrics[len(performanceCollector.metrics)-performanceCollector.maxMetricsCount:]
	}

	// 更新日度汇总
	updateDailySummary(metric)

	// 更新小时趋势
	updateHourlyTrend(metric)

	// 检测异常并生成告警
	checkAndCreateAlerts(metric)
}

// updateDailySummary 更新日度汇总
func updateDailySummary(metric TrailingStopPerformanceMetric) {
	dateKey := metric.Timestamp.Format("2006-01-02")

	if _, exists := performanceCollector.dailySummaries[dateKey]; !exists {
		performanceCollector.dailySummaries[dateKey] = &DailyPerformanceSummary{
			Date: dateKey,
		}
	}

	summary := performanceCollector.dailySummaries[dateKey]
	summary.TotalTradeCount++
	summary.TotalGainedPips += metric.GainedPips
	summary.TotalAvoidedLoss += metric.AvoidedLoss
	summary.AverageSLMoves = (summary.AverageSLMoves + float64(metric.SLMoveCount)) / 2

	if metric.Status == "triggered" {
		summary.TrailingTriggered++
		summary.EffectiveTradeCount++
	}

	if metric.SLMoveCount > 0 {
		summary.AveragePipsProtected = (summary.AveragePipsProtected + metric.GainedPips) / 2
	}

	// 计算有效率
	if summary.TotalTradeCount > 0 {
		summary.EffectiveRate = float64(summary.EffectiveTradeCount) / float64(summary.TotalTradeCount) * 100
	}
}

// updateHourlyTrend 更新小时趋势
func updateHourlyTrend(metric TrailingStopPerformanceMetric) {
	hour := metric.Timestamp.Hour()
	if hour < 0 || hour > 23 {
		return
	}

	trend := &performanceCollector.hourlyTrends[hour]
	trend.Hour = hour
	trend.TradeCount++
	trend.TotalGainedPips += metric.GainedPips
	trend.AvgSLMovePerTrade = (trend.AvgSLMovePerTrade + float64(metric.SLMoveCount)) / 2

	if metric.Status == "triggered" {
		trend.SuccessRate = (float64(trend.TradeCount) - 1) / float64(trend.TradeCount) * 100
	}
}

// checkAndCreateAlerts 检查并创建告警
func checkAndCreateAlerts(metric TrailingStopPerformanceMetric) {
	thresholds := performanceCollector.performanceThresholds

	// 检查异常的止损移动次数
	if metric.SLMoveCount > thresholds.AbnormalSLMovementCount {
		createAlert(PerformanceAlertEvent{
			AlertType:   "abnormal_sl_movements",
			Severity:    "medium",
			TraderID:    metric.TraderID,
			Symbol:      metric.Symbol,
			Message:     "Abnormal number of SL movements detected",
			MetricValue: float64(metric.SLMoveCount),
			Threshold:   float64(thresholds.AbnormalSLMovementCount),
			Action:      "Review stop loss settings",
		})
	}

	// 检查避免损失的百分比
	if metric.AvoidedLoss < thresholds.MinAvoidedLossPercent && metric.Status == "triggered" {
		createAlert(PerformanceAlertEvent{
			AlertType:   "low_avoided_loss",
			Severity:    "low",
			TraderID:    metric.TraderID,
			Symbol:      metric.Symbol,
			Message:     "Low avoided loss percentage",
			MetricValue: metric.AvoidedLoss,
			Threshold:   thresholds.MinAvoidedLossPercent,
			Action:      "Consider adjusting trailing stop parameters",
		})
	}
}

// createAlert 创建告警事件
func createAlert(alert PerformanceAlertEvent) {
	alert.Timestamp = time.Now()
	performanceCollector.alerts = append(performanceCollector.alerts, alert)

	// 限制告警历史
	if len(performanceCollector.alerts) > 5000 {
		performanceCollector.alerts = performanceCollector.alerts[len(performanceCollector.alerts)-5000:]
	}
}

// GetPerformanceMetrics 获取所有性能指标
func GetPerformanceMetrics(limit int) []TrailingStopPerformanceMetric {
	performanceCollector.mu.RLock()
	defer performanceCollector.mu.RUnlock()

	if limit <= 0 || limit > len(performanceCollector.metrics) {
		limit = len(performanceCollector.metrics)
	}

	result := make([]TrailingStopPerformanceMetric, limit)
	copy(result, performanceCollector.metrics[len(performanceCollector.metrics)-limit:])
	return result
}

// GetDailySummaries 获取日度汇总
func GetDailySummaries(days int) []DailyPerformanceSummary {
	performanceCollector.mu.RLock()
	defer performanceCollector.mu.RUnlock()

	result := make([]DailyPerformanceSummary, 0)
	now := time.Now()

	for i := 0; i < days; i++ {
		dateKey := now.AddDate(0, 0, -i).Format("2006-01-02")
		if summary, exists := performanceCollector.dailySummaries[dateKey]; exists {
			result = append(result, *summary)
		}
	}

	return result
}

// GetHourlyTrends 获取小时趋势
func GetHourlyTrends() []HourlyPerformanceTrend {
	performanceCollector.mu.RLock()
	defer performanceCollector.mu.RUnlock()

	result := make([]HourlyPerformanceTrend, len(performanceCollector.hourlyTrends))
	copy(result, performanceCollector.hourlyTrends)
	return result
}

// GetActiveAlerts 获取活跃告警
func GetActiveAlerts() []PerformanceAlertEvent {
	performanceCollector.mu.RLock()
	defer performanceCollector.mu.RUnlock()

	var active []PerformanceAlertEvent
	for _, alert := range performanceCollector.alerts {
		if !alert.Resolved {
			active = append(active, alert)
		}
	}
	return active
}

// GetRecentAlerts 获取最近的告警
func GetRecentAlerts(limit int) []PerformanceAlertEvent {
	performanceCollector.mu.RLock()
	defer performanceCollector.mu.RUnlock()

	if limit <= 0 || limit > len(performanceCollector.alerts) {
		limit = len(performanceCollector.alerts)
	}

	result := make([]PerformanceAlertEvent, limit)
	copy(result, performanceCollector.alerts[len(performanceCollector.alerts)-limit:])
	return result
}

// GetPerformanceStats 获取聚合的性能统计
func GetPerformanceStats() map[string]interface{} {
	performanceCollector.mu.RLock()
	defer performanceCollector.mu.RUnlock()

	totalMetrics := len(performanceCollector.metrics)
	var totalGainedPips, totalAvoidedLoss float64
	var totalSLMoves int
	var triggeredCount int

	for _, metric := range performanceCollector.metrics {
		totalGainedPips += metric.GainedPips
		totalAvoidedLoss += metric.AvoidedLoss
		totalSLMoves += metric.SLMoveCount
		if metric.Status == "triggered" {
			triggeredCount++
		}
	}

	avgGainedPips := 0.0
	avgSLMoves := 0.0
	avgAvoidedLoss := 0.0

	if totalMetrics > 0 {
		avgGainedPips = totalGainedPips / float64(totalMetrics)
		avgSLMoves = float64(totalSLMoves) / float64(totalMetrics)
		avgAvoidedLoss = totalAvoidedLoss / float64(totalMetrics)
	}

	return map[string]interface{}{
		"total_metrics":      totalMetrics,
		"total_gained_pips":  totalGainedPips,
		"total_avoided_loss": totalAvoidedLoss,
		"total_sl_moves":     totalSLMoves,
		"triggered_count":    triggeredCount,
		"avg_gained_pips":    avgGainedPips,
		"avg_sl_moves":       avgSLMoves,
		"avg_avoided_loss":   avgAvoidedLoss,
		"triggered_rate":     float64(triggeredCount) / float64(totalMetrics) * 100,
		"active_alerts":      len(GetActiveAlerts()),
		"threshold_settings": performanceCollector.performanceThresholds,
	}
}
