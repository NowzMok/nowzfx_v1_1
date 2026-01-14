package backtest

import (
	"fmt"
	"nofx/logger"
	"nofx/store"
	"sync"
	"time"
)

// ReflectionScheduler 反思定时调度器
type ReflectionScheduler struct {
	reflectionEngine *ReflectionEngine
	store            *store.Store
	traders          map[string]bool // traderID -> enabled
	mu               sync.RWMutex

	// Schedule configuration
	enabled      bool
	schedule     string        // Cron expression or simple schedule
	analysisDays int           // 分析周期（天数）
	stopCh       chan struct{} // Stop signal
	wg           sync.WaitGroup
}

// NewReflectionScheduler creates a new reflection scheduler
func NewReflectionScheduler(engine *ReflectionEngine, store *store.Store) *ReflectionScheduler {
	return &ReflectionScheduler{
		reflectionEngine: engine,
		store:            store,
		traders:          make(map[string]bool),
		enabled:          true,
		analysisDays:     7, // 默认分析 7 天
		stopCh:           make(chan struct{}),
	}
}

// Start starts the scheduler
func (rs *ReflectionScheduler) Start() error {
	if !rs.enabled {
		logger.Infof("🛑 Reflection scheduler is disabled")
		return nil
	}

	logger.Infof("🚀 Reflection scheduler started")
	rs.wg.Add(1)
	go rs.schedulerLoop()

	return nil
}

// Stop stops the scheduler
func (rs *ReflectionScheduler) Stop() {
	close(rs.stopCh)
	rs.wg.Wait()
	logger.Infof("⏹ Reflection scheduler stopped")
}

// RegisterTrader registers a trader for reflection
func (rs *ReflectionScheduler) RegisterTrader(traderID string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.traders[traderID] = true
	logger.Infof("📝 Trader %s registered for reflection scheduling", traderID)
}

// UnregisterTrader unregisters a trader
func (rs *ReflectionScheduler) UnregisterTrader(traderID string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	delete(rs.traders, traderID)
	logger.Infof("❌ Trader %s unregistered from reflection scheduling", traderID)
}

// schedulerLoop is the main scheduler loop
func (rs *ReflectionScheduler) schedulerLoop() {
	defer rs.wg.Done()

	// 初始延迟到下一个周期时间（默认每周日 22:00）
	ticker := time.NewTicker(24 * time.Hour) // 每天检查一次
	defer ticker.Stop()

	logger.Infof("📅 Reflection scheduler loop started, checking daily at scheduled time")

	for {
		select {
		case <-rs.stopCh:
			return
		case <-ticker.C:
			// 检查是否到达调度时间
			if rs.shouldRunReflection() {
				rs.runAllReflections()
			}
		}
	}
}

// shouldRunReflection checks if reflection should run now
func (rs *ReflectionScheduler) shouldRunReflection() bool {
	now := time.Now()

	// 默认策略：每周日 22:00
	if now.Weekday() == time.Sunday && now.Hour() == 22 && now.Minute() < 5 {
		logger.Infof("⏰ Scheduled reflection time reached (Sunday 22:00)")
		return true
	}

	// TODO: 支持更复杂的 Cron 表达式

	return false
}

// runAllReflections runs reflection for all registered traders
func (rs *ReflectionScheduler) runAllReflections() {
	rs.mu.RLock()
	traders := make([]string, 0, len(rs.traders))
	for traderID := range rs.traders {
		traders = append(traders, traderID)
	}
	rs.mu.RUnlock()

	if len(traders) == 0 {
		logger.Infof("⚠️  No traders registered for reflection")
		return
	}

	logger.Infof("🔄 Running reflections for %d traders", len(traders))

	// 并发运行多个交易员的反思（限制并发数）
	semaphore := make(chan struct{}, 3) // 最多 3 个并发
	var wg sync.WaitGroup

	for _, traderID := range traders {
		wg.Add(1)
		go func(tid string) {
			defer wg.Done()
			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			if err := rs.runReflectionForTrader(tid); err != nil {
				logger.Errorf("❌ Reflection failed for trader %s: %v", tid, err)
			}
		}(traderID)
	}

	wg.Wait()
	logger.Infof("✅ All reflections completed")
}

// runReflectionForTrader runs reflection for a single trader
func (rs *ReflectionScheduler) runReflectionForTrader(traderID string) error {
	logger.Infof("🔍 Running reflection for trader: %s", traderID)

	// 计算分析周期
	endTime := time.Now().UTC()
	startTime := endTime.AddDate(0, 0, -rs.analysisDays)

	// 运行反思分析
	reflection, err := rs.reflectionEngine.AnalyzePeriod(traderID, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to analyze period: %w", err)
	}

	if reflection == nil {
		logger.Infof("⚠️  No trades in period for trader %s, skipping", traderID)
		return nil
	}

	// 应用建议
	if err := rs.reflectionEngine.ApplyRecommendations(reflection); err != nil {
		return fmt.Errorf("failed to apply recommendations: %w", err)
	}

	logger.Infof("✅ Reflection completed for trader %s, %d trades analyzed",
		traderID, reflection.TotalTrades)

	// 发送通知（可选）
	rs.sendNotification(traderID, reflection)

	return nil
}

// sendNotification sends notification about reflection results
func (rs *ReflectionScheduler) sendNotification(traderID string, reflection *store.ReflectionRecord) {
	// TODO: 实现通知机制（邮件、webhook、消息等）
	logger.Infof("📬 Notification: Reflection completed for trader %s", traderID)
	logger.Infof("   - Total trades: %d", reflection.TotalTrades)
	logger.Infof("   - Success rate: %.2f%%", reflection.SuccessRate*100)
	logger.Infof("   - Total PnL: %.2f USDT", reflection.TotalPnL)
}

// ManualTrigger manually triggers reflection for a trader
func (rs *ReflectionScheduler) ManualTrigger(traderID string) error {
	logger.Infof("🚀 Manual reflection triggered for trader: %s", traderID)
	return rs.runReflectionForTrader(traderID)
}

// GetRecentReflections gets recent reflections for a trader
func (rs *ReflectionScheduler) GetRecentReflections(traderID string, limit int) ([]*store.ReflectionRecord, error) {
	return rs.store.Reflection().GetRecentReflections(traderID, limit)
}

// GetReflectionStats gets reflection statistics
func (rs *ReflectionScheduler) GetReflectionStats(traderID string, days int) (map[string]interface{}, error) {
	return rs.store.Reflection().GetReflectionStats(traderID, days)
}

// SetAnalysisDays sets the analysis period in days
func (rs *ReflectionScheduler) SetAnalysisDays(days int) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if days > 0 && days <= 90 {
		rs.analysisDays = days
		logger.Infof("📊 Analysis period set to %d days", days)
	}
}

// SetSchedule sets the schedule (for future cron support)
func (rs *ReflectionScheduler) SetSchedule(schedule string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.schedule = schedule
	logger.Infof("⏰ Schedule set to: %s", schedule)
}

// Enable enables the scheduler
func (rs *ReflectionScheduler) Enable() {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.enabled = true
	logger.Infof("✅ Reflection scheduler enabled")
}

// Disable disables the scheduler
func (rs *ReflectionScheduler) Disable() {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.enabled = false
	logger.Infof("🛑 Reflection scheduler disabled")
}
