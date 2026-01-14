package store

import (
	"fmt"
	"time"
)

// MonitoringRepository 监控数据访问层
type MonitoringRepository struct {
	db *Store
}

// NewMonitoringRepository 创建新的监控仓储
func NewMonitoringRepository(st *Store) *MonitoringRepository {
	return &MonitoringRepository{db: st}
}

// ====== Performance Metric Operations ======

// SaveMetric 保存性能指标
func (mr *MonitoringRepository) SaveMetric(metric *PerformanceMetric) error {
	return mr.db.GormDB().Create(metric).Error
}

// GetMetricsByTraderID 获取指定交易员的性能指标
func (mr *MonitoringRepository) GetMetricsByTraderID(traderID string, limit int) ([]*PerformanceMetric, error) {
	var metrics []*PerformanceMetric
	err := mr.db.GormDB().
		Where("trader_id = ?", traderID).
		Order("timestamp DESC").
		Limit(limit).
		Find(&metrics).Error
	return metrics, err
}

// GetMetricsByTimeRange 获取时间范围内的性能指标
func (mr *MonitoringRepository) GetMetricsByTimeRange(traderID string, startTime, endTime time.Time) ([]*PerformanceMetric, error) {
	var metrics []*PerformanceMetric
	err := mr.db.GormDB().
		Where("trader_id = ? AND timestamp BETWEEN ? AND ?", traderID, startTime, endTime).
		Order("timestamp ASC").
		Find(&metrics).Error
	return metrics, err
}

// GetLatestMetric 获取最新的性能指标
func (mr *MonitoringRepository) GetLatestMetric(traderID string) (*PerformanceMetric, error) {
	var metric *PerformanceMetric
	err := mr.db.GormDB().
		Where("trader_id = ?", traderID).
		Order("timestamp DESC").
		First(&metric).Error
	return metric, err
}

// ====== Alert Rule Operations ======

// CreateAlertRule 创建告警规则
func (mr *MonitoringRepository) CreateAlertRule(rule *AlertRule) error {
	return mr.db.GormDB().Create(rule).Error
}

// GetAlertRulesByTraderID 获取交易员的所有告警规则
func (mr *MonitoringRepository) GetAlertRulesByTraderID(traderID string) ([]*AlertRule, error) {
	var rules []*AlertRule
	err := mr.db.GormDB().
		Where("trader_id = ?", traderID).
		Find(&rules).Error
	return rules, err
}

// GetEnabledAlertRules 获取启用的告警规则
func (mr *MonitoringRepository) GetEnabledAlertRules(traderID string) ([]*AlertRule, error) {
	var rules []*AlertRule
	err := mr.db.GormDB().
		Where("trader_id = ? AND enabled = ?", traderID, true).
		Find(&rules).Error
	return rules, err
}

// UpdateAlertRule 更新告警规则
func (mr *MonitoringRepository) UpdateAlertRule(ruleID string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	return mr.db.GormDB().
		Model(&AlertRule{}).
		Where("id = ?", ruleID).
		Updates(updates).Error
}

// DeleteAlertRule 删除告警规则
func (mr *MonitoringRepository) DeleteAlertRule(ruleID string) error {
	return mr.db.GormDB().
		Where("id = ?", ruleID).
		Delete(&AlertRule{}).Error
}

// ====== Alert Instance Operations ======

// SaveAlert 保存告警实例
func (mr *MonitoringRepository) SaveAlert(alert *Alert) error {
	return mr.db.GormDB().Create(alert).Error
}

// GetAlertsByTraderID 获取交易员的告警
func (mr *MonitoringRepository) GetAlertsByTraderID(traderID string, limit int) ([]*Alert, error) {
	var alerts []*Alert
	err := mr.db.GormDB().
		Where("trader_id = ?", traderID).
		Order("triggered_at DESC").
		Limit(limit).
		Find(&alerts).Error
	return alerts, err
}

// GetActiveAlerts 获取活跃的告警
func (mr *MonitoringRepository) GetActiveAlerts(traderID string) ([]*Alert, error) {
	var alerts []*Alert
	err := mr.db.GormDB().
		Where("trader_id = ? AND status IN ?", traderID, []string{"triggered", "acknowledged"}).
		Order("triggered_at DESC").
		Find(&alerts).Error
	return alerts, err
}

// GetAlertByID 根据 ID 获取告警
func (mr *MonitoringRepository) GetAlertByID(alertID string) (*Alert, error) {
	var alert *Alert
	err := mr.db.GormDB().
		Where("id = ?", alertID).
		First(&alert).Error
	return alert, err
}

// UpdateAlertStatus 更新告警状态
func (mr *MonitoringRepository) UpdateAlertStatus(alertID string, status string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": now,
	}

	// 根据状态设置对应的时间戳
	switch status {
	case "acknowledged":
		updates["acknowledged_at"] = now
	case "resolved":
		updates["resolved_at"] = now
	}

	return mr.db.GormDB().
		Model(&Alert{}).
		Where("id = ?", alertID).
		Updates(updates).Error
}

// GetAlertsByStatus 按状态获取告警
func (mr *MonitoringRepository) GetAlertsByStatus(traderID string, status string) ([]*Alert, error) {
	var alerts []*Alert
	err := mr.db.GormDB().
		Where("trader_id = ? AND status = ?", traderID, status).
		Order("triggered_at DESC").
		Find(&alerts).Error
	return alerts, err
}

// ====== System Health Operations ======

// SaveHealthCheck 保存健康检查结果
func (mr *MonitoringRepository) SaveHealthCheck(health *SystemHealth) error {
	return mr.db.GormDB().Create(health).Error
}

// GetLatestHealthCheck 获取最新的健康检查
func (mr *MonitoringRepository) GetLatestHealthCheck(traderID string) (*SystemHealth, error) {
	var health *SystemHealth
	err := mr.db.GormDB().
		Where("trader_id = ?", traderID).
		Order("created_at DESC").
		First(&health).Error
	return health, err
}

// GetHealthChecksByTimeRange 获取时间范围内的健康检查
func (mr *MonitoringRepository) GetHealthChecksByTimeRange(traderID string, startTime, endTime time.Time) ([]*SystemHealth, error) {
	var checks []*SystemHealth
	err := mr.db.GormDB().
		Where("trader_id = ? AND created_at BETWEEN ? AND ?", traderID, startTime, endTime).
		Order("created_at DESC").
		Find(&checks).Error
	return checks, err
}

// ====== Metrics Aggregation Operations ======

// SaveAggregation 保存聚合数据
func (mr *MonitoringRepository) SaveAggregation(agg *MetricsAggregation) error {
	return mr.db.GormDB().Create(agg).Error
}

// GetAggregationByPeriod 获取指定周期的聚合数据
func (mr *MonitoringRepository) GetAggregationByPeriod(traderID string, periodType string, limit int) ([]*MetricsAggregation, error) {
	var aggs []*MetricsAggregation
	err := mr.db.GormDB().
		Where("trader_id = ? AND period_type = ?", traderID, periodType).
		Order("period_start DESC").
		Limit(limit).
		Find(&aggs).Error
	return aggs, err
}

// ====== Monitoring Statistics ======

// GetMetricsSummary 获取指标摘要
func (mr *MonitoringRepository) GetMetricsSummary(traderID string, hours int) (map[string]interface{}, error) {
	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)

	var metrics []*PerformanceMetric
	if err := mr.db.GormDB().
		Where("trader_id = ? AND timestamp >= ?", traderID, startTime).
		Order("timestamp ASC").
		Find(&metrics).Error; err != nil {
		return nil, err
	}

	if len(metrics) == 0 {
		return map[string]interface{}{
			"count": 0,
		}, nil
	}

	// 计算统计
	summary := make(map[string]interface{})
	summary["count"] = len(metrics)
	summary["start_time"] = metrics[0].Timestamp
	summary["end_time"] = metrics[len(metrics)-1].Timestamp

	// 计算平均值、最小值、最大值
	var totalWR, totalPF, totalDD float64
	minWR, maxWR := metrics[0].WinRate, metrics[0].WinRate
	minDD, maxDD := metrics[0].MaxDrawdown, metrics[0].MaxDrawdown
	maxTotalPnL := metrics[0].TotalPnL
	minTotalPnL := metrics[0].TotalPnL

	for _, m := range metrics {
		totalWR += m.WinRate
		totalPF += m.ProfitFactor
		totalDD += m.MaxDrawdown

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
		if m.TotalPnL > maxTotalPnL {
			maxTotalPnL = m.TotalPnL
		}
		if m.TotalPnL < minTotalPnL {
			minTotalPnL = m.TotalPnL
		}
	}

	count := float64(len(metrics))
	summary["avg_win_rate"] = totalWR / count
	summary["avg_profit_factor"] = totalPF / count
	summary["avg_drawdown"] = totalDD / count
	summary["min_win_rate"] = minWR
	summary["max_win_rate"] = maxWR
	summary["min_drawdown"] = minDD
	summary["max_drawdown"] = maxDD
	summary["total_pnl"] = metrics[len(metrics)-1].TotalPnL
	summary["peak_pnl"] = maxTotalPnL
	summary["lowest_pnl"] = minTotalPnL

	return summary, nil
}

// CountAlerts 统计告警数量
func (mr *MonitoringRepository) CountAlerts(traderID string, status string) (int64, error) {
	var count int64
	err := mr.db.GormDB().
		Model(&Alert{}).
		Where("trader_id = ? AND status = ?", traderID, status).
		Count(&count).Error
	return count, err
}

// GetAlertStats 获取告警统计
func (mr *MonitoringRepository) GetAlertStats(traderID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	statuses := []string{"triggered", "acknowledged", "resolved"}
	for _, status := range statuses {
		count, err := mr.CountAlerts(traderID, status)
		if err != nil {
			return nil, err
		}
		stats[status] = count
	}

	// 获取严重级别统计
	var severities []struct {
		Severity string
		Count    int64
	}
	if err := mr.db.GormDB().
		Model(&Alert{}).
		Where("trader_id = ?", traderID).
		Select("severity, COUNT(*) as count").
		Group("severity").
		Scan(&severities).Error; err != nil {
		return nil, err
	}

	for _, s := range severities {
		stats[fmt.Sprintf("severity_%s", s.Severity)] = s.Count
	}

	return stats, nil
}

// ====== Cleanup Operations ======

// PruneOldMetrics 清理旧的性能指标
func (mr *MonitoringRepository) PruneOldMetrics(days int) error {
	cutoffTime := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	return mr.db.GormDB().
		Where("timestamp < ?", cutoffTime).
		Delete(&PerformanceMetric{}).Error
}

// PruneOldAlerts 清理已解决的旧告警
func (mr *MonitoringRepository) PruneOldAlerts(days int) error {
	cutoffTime := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	return mr.db.GormDB().
		Where("status = ? AND resolved_at < ?", "resolved", cutoffTime).
		Delete(&Alert{}).Error
}

// PruneOldHealthChecks 清理旧的健康检查记录
func (mr *MonitoringRepository) PruneOldHealthChecks(days int) error {
	cutoffTime := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	return mr.db.GormDB().
		Where("created_at < ?", cutoffTime).
		Delete(&SystemHealth{}).Error
}
