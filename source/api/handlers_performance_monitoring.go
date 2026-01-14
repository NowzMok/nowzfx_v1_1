package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// handleGetPerformanceMetrics 获取性能指标
func (s *Server) handleGetPerformanceMetrics(c *gin.Context) {
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	metrics := GetPerformanceMetrics(limit)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"metrics": metrics,
			"count":   len(metrics),
		},
	})
}

// handleGetDailySummaries 获取日度性能汇总
func (s *Server) handleGetDailySummaries(c *gin.Context) {
	days := 7
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	summaries := GetDailySummaries(days)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"summaries": summaries,
			"count":     len(summaries),
		},
	})
}

// handleGetHourlyTrends 获取小时性能趋势
func (s *Server) handleGetHourlyTrends(c *gin.Context) {
	trends := GetHourlyTrends()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"trends": trends,
			"count":  len(trends),
		},
	})
}

// handleGetPerformanceStats 获取聚合性能统计
func (s *Server) handleGetPerformanceStats(c *gin.Context) {
	stats := GetPerformanceStats()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// handleGetPerformanceAlerts 获取性能告警
func (s *Server) handleGetPerformanceAlerts(c *gin.Context) {
	// 查询参数: active (true/false), limit
	activeOnly := c.Query("active") == "true"
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	var alerts []PerformanceAlertEvent

	if activeOnly {
		alerts = GetActiveAlerts()
	} else {
		alerts = GetRecentAlerts(limit)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"alerts":       alerts,
			"count":        len(alerts),
			"active_count": len(GetActiveAlerts()),
		},
	})
}

// handleResolveAlert 标记告警为已解决
func (s *Server) handleResolveAlert(c *gin.Context) {
	alertID := c.Param("alertId")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "alert_id is required",
		})
		return
	}

	// 这里需要实现告警ID的管理，暂时返回成功
	// 在完整实现中，需要使用UUID或数据库ID

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Alert marked as resolved",
	})
}
