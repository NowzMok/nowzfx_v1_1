package trader

import (
	"fmt"
	"nofx/logger"
	"sync"
	"time"
)

// ErrorTracker 错误追踪器
type ErrorTracker struct {
	mu           sync.RWMutex
	errors       map[string]*ErrorStats // key: error type
	recentErrors []ErrorRecord
	maxRecent    int
}

// ErrorStats 错误统计
type ErrorStats struct {
	ErrorType       string
	Count           int
	FirstSeen       time.Time
	LastSeen        time.Time
	AffectedSymbols map[string]int // symbol -> count
}

// ErrorRecord 错误记录
type ErrorRecord struct {
	Timestamp time.Time
	ErrorType string
	Symbol    string
	Message   string
	Severity  string // INFO, WARN, ERROR, CRITICAL
}

// NewErrorTracker 创建错误追踪器
func NewErrorTracker(maxRecent int) *ErrorTracker {
	return &ErrorTracker{
		errors:       make(map[string]*ErrorStats),
		recentErrors: make([]ErrorRecord, 0, maxRecent),
		maxRecent:    maxRecent,
	}
}

// RecordError 记录错误
func (et *ErrorTracker) RecordError(errorType, symbol, message, severity string) {
	et.mu.Lock()
	defer et.mu.Unlock()

	now := time.Now()

	// 更新统计
	stats, exists := et.errors[errorType]
	if !exists {
		stats = &ErrorStats{
			ErrorType:       errorType,
			Count:           0,
			FirstSeen:       now,
			AffectedSymbols: make(map[string]int),
		}
		et.errors[errorType] = stats
	}

	stats.Count++
	stats.LastSeen = now
	if symbol != "" {
		stats.AffectedSymbols[symbol]++
	}

	// 添加到最近错误列表
	record := ErrorRecord{
		Timestamp: now,
		ErrorType: errorType,
		Symbol:    symbol,
		Message:   message,
		Severity:  severity,
	}

	et.recentErrors = append(et.recentErrors, record)
	if len(et.recentErrors) > et.maxRecent {
		et.recentErrors = et.recentErrors[1:]
	}

	// 根据严重性记录日志
	logMsg := fmt.Sprintf("[ErrorTracker] %s - %s: %s (Symbol: %s)",
		severity, errorType, message, symbol)

	switch severity {
	case "CRITICAL":
		logger.Errorf("🔴 %s", logMsg)
	case "ERROR":
		logger.Errorf("❌ %s", logMsg)
	case "WARN":
		logger.Warnf("⚠️  %s", logMsg)
	default:
		logger.Infof("ℹ️  %s", logMsg)
	}
}

// GetStats 获取错误统计
func (et *ErrorTracker) GetStats() map[string]*ErrorStats {
	et.mu.RLock()
	defer et.mu.RUnlock()

	result := make(map[string]*ErrorStats)
	for k, v := range et.errors {
		// 深拷贝
		symbolsCopy := make(map[string]int)
		for sk, sv := range v.AffectedSymbols {
			symbolsCopy[sk] = sv
		}

		result[k] = &ErrorStats{
			ErrorType:       v.ErrorType,
			Count:           v.Count,
			FirstSeen:       v.FirstSeen,
			LastSeen:        v.LastSeen,
			AffectedSymbols: symbolsCopy,
		}
	}
	return result
}

// GetRecentErrors 获取最近的错误
func (et *ErrorTracker) GetRecentErrors(count int) []ErrorRecord {
	et.mu.RLock()
	defer et.mu.RUnlock()

	if count <= 0 || count > len(et.recentErrors) {
		count = len(et.recentErrors)
	}

	start := len(et.recentErrors) - count
	result := make([]ErrorRecord, count)
	copy(result, et.recentErrors[start:])
	return result
}

// GenerateReport 生成错误报告
func (et *ErrorTracker) GenerateReport() string {
	et.mu.RLock()
	defer et.mu.RUnlock()

	report := "\n╔══════════════════════════════════════════════════════════════╗\n"
	report += "║              📊 错误监控报告                                 ║\n"
	report += "╚══════════════════════════════════════════════════════════════╝\n\n"

	if len(et.errors) == 0 {
		report += "✅ 无错误记录\n"
		return report
	}

	report += fmt.Sprintf("错误类型总数: %d\n", len(et.errors))
	report += fmt.Sprintf("最近错误数: %d\n\n", len(et.recentErrors))

	report += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
	report += "错误类型统计:\n"
	report += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"

	for errorType, stats := range et.errors {
		report += fmt.Sprintf("\n🔸 %s\n", errorType)
		report += fmt.Sprintf("   次数: %d\n", stats.Count)
		report += fmt.Sprintf("   首次: %s\n", stats.FirstSeen.Format("2006-01-02 15:04:05"))
		report += fmt.Sprintf("   最近: %s\n", stats.LastSeen.Format("2006-01-02 15:04:05"))

		if len(stats.AffectedSymbols) > 0 {
			report += "   影响币种:\n"
			for symbol, count := range stats.AffectedSymbols {
				report += fmt.Sprintf("     - %s: %d次\n", symbol, count)
			}
		}
	}

	// 最近的错误
	if len(et.recentErrors) > 0 {
		report += "\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
		report += "最近10条错误:\n"
		report += "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"

		count := 10
		if len(et.recentErrors) < count {
			count = len(et.recentErrors)
		}

		start := len(et.recentErrors) - count
		for i := start; i < len(et.recentErrors); i++ {
			err := et.recentErrors[i]
			report += fmt.Sprintf("\n%s [%s] %s\n",
				err.Timestamp.Format("15:04:05"), err.Severity, err.ErrorType)
			if err.Symbol != "" {
				report += fmt.Sprintf("  Symbol: %s\n", err.Symbol)
			}
			report += fmt.Sprintf("  %s\n", err.Message)
		}
	}

	report += "\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"

	return report
}

// Clear 清除统计数据
func (et *ErrorTracker) Clear() {
	et.mu.Lock()
	defer et.mu.Unlock()

	et.errors = make(map[string]*ErrorStats)
	et.recentErrors = make([]ErrorRecord, 0, et.maxRecent)
	logger.Info("🧹 Error tracker cleared")
}

// GetErrorRate 获取错误率（每分钟）
func (et *ErrorTracker) GetErrorRate() float64 {
	et.mu.RLock()
	defer et.mu.RUnlock()

	if len(et.recentErrors) == 0 {
		return 0
	}

	now := time.Now()
	oneMinuteAgo := now.Add(-1 * time.Minute)

	count := 0
	for i := len(et.recentErrors) - 1; i >= 0; i-- {
		if et.recentErrors[i].Timestamp.After(oneMinuteAgo) {
			count++
		} else {
			break
		}
	}

	return float64(count)
}
