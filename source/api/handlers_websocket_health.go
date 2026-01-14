package api

import (
	"net/http"
	"time"

	"nofx/logger"

	"github.com/gin-gonic/gin"
)

// WebSocketHealthStatus WebSocket连接健康状态
type WebSocketHealthStatus struct {
	Connected             bool      `json:"connected"`                 // 是否已连接
	LastHeartbeat         time.Time `json:"last_heartbeat"`            // 最后心跳时间
	UpcomingDuration      string    `json:"uptime_duration"`           // 运行时长
	ReconnectCount        int       `json:"reconnect_count"`           // 重连次数
	MessageProcessedCount int64     `json:"message_processed_count"`   // 处理的消息数
	MessageRate           float64   `json:"message_rate_per_sec"`      // 消息处理速率 (每秒)
	LastError             string    `json:"last_error,omitempty"`      // 最后一个错误
	LastErrorTime         time.Time `json:"last_error_time,omitempty"` // 最后错误时间
	RecommendedAction     string    `json:"recommended_action"`        // 推荐行动
	Status                string    `json:"status"`                    // 状态: "healthy", "degraded", "unhealthy"
}

// WebSocketHealthTracker WebSocket健康追踪器
type WebSocketHealthTracker struct {
	Connected             bool
	LastHeartbeat         time.Time
	StartTime             time.Time
	ReconnectCount        int
	MessageProcessedCount int64
	LastError             string
	LastErrorTime         time.Time
	MessageCountSnapshot  int64
	LastCountTime         time.Time
}

var wsHealthTracker = &WebSocketHealthTracker{
	StartTime:     time.Now(),
	LastHeartbeat: time.Now(),
	LastCountTime: time.Now(),
}

// UpdateWebSocketHealth 更新WebSocket健康状态
func UpdateWebSocketHealth(connected bool, messageCount int64) {
	wsHealthTracker.Connected = connected
	wsHealthTracker.LastHeartbeat = time.Now()
	wsHealthTracker.MessageProcessedCount = messageCount

	if connected {
		// 已连接，清除错误状态
		wsHealthTracker.LastError = ""
	}
}

// RecordWebSocketError 记录WebSocket错误
func RecordWebSocketError(err string) {
	wsHealthTracker.LastError = err
	wsHealthTracker.LastErrorTime = time.Now()
	logger.Warnf("⚠️ WebSocket error recorded: %s", err)
}

// IncrementReconnectCount 增加重连次数
func IncrementReconnectCount() {
	wsHealthTracker.ReconnectCount++
}

// handleWebSocketHealth 处理WebSocket健康检查请求
func (s *Server) handleWebSocketHealth(c *gin.Context) {
	health := getWebSocketHealthStatus()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    health,
		"message": health.Status,
	})
}

// getWebSocketHealthStatus 获取WebSocket健康状态
func getWebSocketHealthStatus() WebSocketHealthStatus {
	now := time.Now()
	uptime := now.Sub(wsHealthTracker.StartTime)

	// 计算消息速率 (最近5秒内的消息数)
	timeSinceLastCount := now.Sub(wsHealthTracker.LastCountTime).Seconds()
	var messageRate float64
	if timeSinceLastCount > 0 {
		messageDiff := wsHealthTracker.MessageProcessedCount - wsHealthTracker.MessageCountSnapshot
		messageRate = float64(messageDiff) / timeSinceLastCount
	}
	wsHealthTracker.MessageCountSnapshot = wsHealthTracker.MessageProcessedCount
	wsHealthTracker.LastCountTime = now

	// 判断健康状态
	timeSinceLastHeartbeat := now.Sub(wsHealthTracker.LastHeartbeat)
	status := "healthy"
	recommendedAction := "no action needed"

	if !wsHealthTracker.Connected {
		status = "unhealthy"
		recommendedAction = "waiting for reconnection"
		if timeSinceLastHeartbeat > 30*time.Second {
			recommendedAction = "check network connectivity"
		}
	} else if timeSinceLastHeartbeat > 10*time.Second {
		status = "degraded"
		recommendedAction = "heartbeat delayed, monitor closely"
	} else if wsHealthTracker.LastError != "" && timeSinceLastHeartbeat < 5*time.Second {
		// 最近有错误但已恢复
		status = "degraded"
		recommendedAction = "recovery in progress"
	}

	// 如果最后一个错误是10秒前的，清除
	if wsHealthTracker.LastError != "" && now.Sub(wsHealthTracker.LastErrorTime) > 10*time.Second {
		wsHealthTracker.LastError = ""
	}

	return WebSocketHealthStatus{
		Connected:             wsHealthTracker.Connected,
		LastHeartbeat:         wsHealthTracker.LastHeartbeat,
		UpcomingDuration:      uptime.String(),
		ReconnectCount:        wsHealthTracker.ReconnectCount,
		MessageProcessedCount: wsHealthTracker.MessageProcessedCount,
		MessageRate:           messageRate,
		LastError:             wsHealthTracker.LastError,
		LastErrorTime:         wsHealthTracker.LastErrorTime,
		RecommendedAction:     recommendedAction,
		Status:                status,
	}
}

// registerWebSocketHealthRoutes 注册WebSocket健康检查路由
func registerWebSocketHealthRoutes(router *gin.RouterGroup) {
	router.GET("/health/websocket", func(c *gin.Context) {
		health := getWebSocketHealthStatus()
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    health,
		})
	})
}
