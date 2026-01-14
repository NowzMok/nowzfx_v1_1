package api

import (
	"net/http"
	"sync"
	"time"

	"nofx/logger"

	"github.com/gin-gonic/gin"
)

// CleanupStats 清理操作统计
type CleanupStats struct {
	Timestamp          time.Time `json:"timestamp"`            // 清理时间
	DuplicateOrders    int       `json:"duplicate_orders"`     // 清理的重复订单数
	DuplicateFills     int       `json:"duplicate_fills"`      // 清理的重复成交数
	ExpiredOrders      int       `json:"expired_orders"`       // 清理的过期订单数
	StaleOrders        int       `json:"stale_orders"`         // 清理的陈旧订单数
	TotalCleaned       int       `json:"total_cleaned"`        // 总清理数
	Duration           string    `json:"duration_ms"`          // 耗时
	AffectedTraders    int       `json:"affected_traders"`     // 涉及的交易者数
	LastCleanupSuccess bool      `json:"last_cleanup_success"` // 最后清理是否成功
	NextScheduledClean time.Time `json:"next_scheduled_clean"` // 下次清理时间
	CleanupInterval    string    `json:"cleanup_interval"`     // 清理间隔 (5 minutes)
}

// CleanupStatsTracker 清理统计追踪器
type CleanupStatsTracker struct {
	mu                  sync.RWMutex
	lastStats           *CleanupStats
	lastCleanupTime     time.Time
	lastCleanupDuration time.Duration
	totalCleanupCount   int64
}

var cleanupStatsTracker = &CleanupStatsTracker{
	lastCleanupTime: time.Now(),
}

// RecordCleanupOperation 记录清理操作
func RecordCleanupOperation(dupOrders, dupFills, expiredOrders, staleOrders, affectedTraders int, duration time.Duration, success bool) {
	cleanupStatsTracker.mu.Lock()
	defer cleanupStatsTracker.mu.Unlock()

	totalCleaned := dupOrders + dupFills + expiredOrders + staleOrders

	cleanupStatsTracker.lastStats = &CleanupStats{
		Timestamp:          time.Now(),
		DuplicateOrders:    dupOrders,
		DuplicateFills:     dupFills,
		ExpiredOrders:      expiredOrders,
		StaleOrders:        staleOrders,
		TotalCleaned:       totalCleaned,
		Duration:           duration.String(),
		AffectedTraders:    affectedTraders,
		LastCleanupSuccess: success,
		NextScheduledClean: time.Now().Add(5 * time.Minute),
		CleanupInterval:    "5 minutes",
	}

	cleanupStatsTracker.lastCleanupTime = time.Now()
	cleanupStatsTracker.lastCleanupDuration = duration
	cleanupStatsTracker.totalCleanupCount++

	if totalCleaned > 0 {
		logger.Infof(
			"🧹 Cleanup recorded: %d dup orders, %d dup fills, %d expired, %d stale (total: %d, affected: %d, duration: %v)",
			dupOrders, dupFills, expiredOrders, staleOrders, totalCleaned, affectedTraders, duration,
		)
	}
}

// GetCleanupStats 获取最近一次清理统计
func GetCleanupStats() *CleanupStats {
	cleanupStatsTracker.mu.RLock()
	defer cleanupStatsTracker.mu.RUnlock()

	if cleanupStatsTracker.lastStats == nil {
		return &CleanupStats{
			Timestamp:          time.Now(),
			NextScheduledClean: time.Now().Add(5 * time.Minute),
			CleanupInterval:    "5 minutes",
			LastCleanupSuccess: true,
		}
	}

	stats := *cleanupStatsTracker.lastStats
	return &stats
}

// handleGetCleanupStats 处理获取清理统计请求
func (s *Server) handleGetCleanupStats(c *gin.Context) {
	stats := GetCleanupStats()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"latest": stats,
			"metrics": gin.H{
				"total_cleanup_operations": cleanupStatsTracker.totalCleanupCount,
				"last_cleanup_time":        cleanupStatsTracker.lastCleanupTime.Format(time.RFC3339),
				"last_cleanup_duration_ms": cleanupStatsTracker.lastCleanupDuration.Milliseconds(),
			},
		},
	})
}
