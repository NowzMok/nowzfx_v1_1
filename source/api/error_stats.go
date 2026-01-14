package api

import (
	"fmt"
	"net/http"

	"nofx/trader"

	"github.com/gin-gonic/gin"
)

// ErrorStats 错误统计结构
type ErrorStatsResponse struct {
	ErrorType       string         `json:"error_type"`
	Count           int            `json:"count"`
	FirstSeen       string         `json:"first_seen"`
	LastSeen        string         `json:"last_seen"`
	AffectedSymbols map[string]int `json:"affected_symbols"`
}

// ErrorRecordResponse 错误记录结构
type ErrorRecordResponse struct {
	Timestamp string `json:"timestamp"`
	ErrorType string `json:"error_type"`
	Symbol    string `json:"symbol"`
	Message   string `json:"message"`
	Severity  string `json:"severity"`
}

// GetErrorStats 获取错误统计信息
func GetErrorStats(traderInstance *trader.AutoTrader) gin.HandlerFunc {
	return func(c *gin.Context) {
		if traderInstance == nil || traderInstance.GetErrorTracker() == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Error tracker not available",
			})
			return
		}

		stats := traderInstance.GetErrorTracker().GetStats()

		response := make(map[string]interface{})
		response["total_error_types"] = len(stats)
		response["stats"] = stats

		c.JSON(http.StatusOK, response)
	}
}

// GetRecentErrors 获取最近的错误
func GetRecentErrors(traderInstance *trader.AutoTrader) gin.HandlerFunc {
	return func(c *gin.Context) {
		if traderInstance == nil || traderInstance.GetErrorTracker() == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Error tracker not available",
			})
			return
		}

		// 默认获取10条，可通过query参数修改
		count := 10
		if countStr := c.Query("count"); countStr != "" {
			if _, err := c.Cookie("count"); err == nil {
				// 安全解析count
				if n, err := parseInt(countStr); err == nil && n > 0 && n <= 100 {
					count = n
				}
			}
		}

		records := traderInstance.GetErrorTracker().GetRecentErrors(count)

		response := make([]ErrorRecordResponse, len(records))
		for i, record := range records {
			response[i] = ErrorRecordResponse{
				Timestamp: record.Timestamp.Format("2006-01-02 15:04:05"),
				ErrorType: record.ErrorType,
				Symbol:    record.Symbol,
				Message:   record.Message,
				Severity:  record.Severity,
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"count":  len(response),
			"errors": response,
		})
	}
}

// GetErrorReport 获取完整的错误报告
func GetErrorReport(traderInstance *trader.AutoTrader) gin.HandlerFunc {
	return func(c *gin.Context) {
		if traderInstance == nil || traderInstance.GetErrorTracker() == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Error tracker not available",
			})
			return
		}

		report := traderInstance.GetErrorTracker().GenerateReport()
		c.JSON(http.StatusOK, gin.H{
			"report": report,
		})
	}
}

// ClearErrors 清除错误统计
func ClearErrors(traderInstance *trader.AutoTrader) gin.HandlerFunc {
	return func(c *gin.Context) {
		if traderInstance == nil || traderInstance.GetErrorTracker() == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Error tracker not available",
			})
			return
		}

		traderInstance.GetErrorTracker().Clear()
		c.JSON(http.StatusOK, gin.H{
			"message": "Error statistics cleared",
		})
	}
}

// GetErrorRate 获取错误率（每分钟）
func GetErrorRate(traderInstance *trader.AutoTrader) gin.HandlerFunc {
	return func(c *gin.Context) {
		if traderInstance == nil || traderInstance.GetErrorTracker() == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Error tracker not available",
			})
			return
		}

		rate := traderInstance.GetErrorTracker().GetErrorRate()
		c.JSON(http.StatusOK, gin.H{
			"error_rate_per_minute": rate,
		})
	}
}

// 辅助函数
func parseInt(s string) (int, error) {
	n := 0
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
