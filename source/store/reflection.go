package store

import (
	"encoding/json"
	"time"
)

// ============================================================================
// Reflection Data Models
// ============================================================================

// ReflectionRecord AI 反思记录
type ReflectionRecord struct {
	ID                 string                 `gorm:"primaryKey;column:id" json:"id"`
	TraderID           string                 `gorm:"index:idx_trader_time;column:trader_id" json:"trader_id"`
	ReflectionTime     time.Time              `gorm:"index:idx_trader_time;column:reflection_time" json:"reflection_time"`
	PeriodStartTime    time.Time              `column:period_start_time" json:"period_start_time"`
	PeriodEndTime      time.Time              `column:period_end_time" json:"period_end_time"`
	TotalTrades        int                    `column:total_trades" json:"total_trades"`
	SuccessfulTrades   int                    `column:successful_trades" json:"successful_trades"`
	FailedTrades       int                    `column:failed_trades" json:"failed_trades"`
	SuccessRate        float64                `column:success_rate" json:"success_rate"`
	AveragePnL         float64                `column:average_pnl" json:"average_pnl"`
	MaxProfit          float64                `column:max_profit" json:"max_profit"`
	MaxLoss            float64                `column:max_loss" json:"max_loss"`
	TotalPnL           float64                `column:total_pnl" json:"total_pnl"`
	PnLPercentage      float64                `column:pnl_percentage" json:"pnl_percentage"`
	SharpeRatio        float64                `column:sharpe_ratio" json:"sharpe_ratio"`
	MaxDrawdown        float64                `column:max_drawdown" json:"max_drawdown"`
	WinLossRatio       float64                `column:win_loss_ratio" json:"win_loss_ratio"`
	ConfidenceAccuracy map[string]float64     `gorm:"type:json;column:confidence_accuracy" json:"confidence_accuracy"` // 信心度准确率
	SymbolPerformance  map[string]interface{} `gorm:"type:json;column:symbol_performance" json:"symbol_performance"`   // 按交易对的表现
	AIReflection       string                 `gorm:"type:text;column:ai_reflection" json:"ai_reflection"`             // AI 反思内容
	Recommendations    []json.RawMessage      `gorm:"type:json;column:recommendations" json:"recommendations"`         // 改进建议
	TradeSystemAdvice  []json.RawMessage      `gorm:"type:json;column:trade_system_advice" json:"trade_system_advice"` // 交易系统建议
	AILearningAdvice   []json.RawMessage      `gorm:"type:json;column:ai_learning_advice" json:"ai_learning_advice"`   // AI 学习建议
	CreatedAt          time.Time              `column:created_at" json:"created_at"`
	UpdatedAt          time.Time              `column:updated_at" json:"updated_at"`
}

// TableName specifies the table name
func (r *ReflectionRecord) TableName() string {
	return "reflections"
}

// SystemAdjustment 系统参数调整记录
type SystemAdjustment struct {
	ID               string     `gorm:"primaryKey;column:id" json:"id"`
	TraderID         string     `gorm:"index;column:trader_id" json:"trader_id"`
	ReflectionID     string     `gorm:"index;column:reflection_id" json:"reflection_id"` // 关联的反思记录
	AdjustmentTime   time.Time  `column:adjustment_time" json:"adjustment_time"`
	ConfidenceLevel  float64    `column:confidence_level" json:"confidence_level"`                   // 信心度阈值
	BTCETHLeverage   int        `column:btc_eth_leverage" json:"btc_eth_leverage"`                   // BTC/ETH 杠杆
	AltcoinLeverage  int        `column:altcoin_leverage" json:"altcoin_leverage"`                   // 山寨币杠杆
	MaxPositionSize  float64    `column:max_position_size" json:"max_position_size"`                 // 单笔最大仓位
	MaxDailyLoss     float64    `column:max_daily_loss" json:"max_daily_loss"`                       // 日最大亏损
	StopLossPct      float64    `column:stop_loss_pct" json:"stop_loss_pct"`                         // 止损比例
	TakeProfitPct    float64    `column:take_profit_pct" json:"take_profit_pct"`                     // 止盈比例
	AdjustmentReason string     `gorm:"type:text;column:adjustment_reason" json:"adjustment_reason"` // 调整原因
	AppliedAt        *time.Time `column:applied_at" json:"applied_at"`                               // 应用时间
	Status           string     `column:status" json:"status"`                                       // "PENDING", "APPLIED", "REVERTED"
	CreatedAt        time.Time  `column:created_at" json:"created_at"`
}

// TableName specifies the table name
func (s *SystemAdjustment) TableName() string {
	return "system_adjustments"
}

// AILearningMemory AI 学习记忆库
type AILearningMemory struct {
	ID              string     `gorm:"primaryKey;column:id" json:"id"`
	TraderID        string     `gorm:"index;column:trader_id" json:"trader_id"`
	ReflectionID    string     `gorm:"index;column:reflection_id" json:"reflection_id"`
	CreatedAt       time.Time  `column:created_at" json:"created_at"`
	ExpiresAt       time.Time  `column:expires_at" json:"expires_at"`                             // 记忆过期时间
	MemoryType      string     `column:memory_type" json:"memory_type"`                           // "bias", "pattern", "lesson", "warning"
	Symbol          string     `column:symbol" json:"symbol"`                                     // 相关交易对（可选）
	Content         string     `gorm:"type:text;column:content" json:"content"`                   // 记忆内容
	Confidence      float64    `column:confidence" json:"confidence"`                             // 这个记忆的可信度
	UsageCount      int        `column:usage_count" json:"usage_count"`                           // 被使用次数
	LastUsedAt      *time.Time `column:last_used_at" json:"last_used_at"`                         // 最后使用时间
	PromptInjection string     `gorm:"type:text;column:prompt_injection" json:"prompt_injection"` // AI prompt 注入内容
	UpdatedAt       time.Time  `column:updated_at" json:"updated_at"`
}

// TableName specifies the table name
func (m *AILearningMemory) TableName() string {
	return "ai_learning_memory"
}

// ReflectionRecommendation 单条建议
type ReflectionRecommendation struct {
	Type        string  `json:"type"`        // "trade_system", "ai_learning"
	Category    string  `json:"category"`    // "confidence", "leverage", "position_size", "risk_control"
	Symbol      string  `json:"symbol"`      // 相关交易对
	Current     float64 `json:"current"`     // 当前值
	Recommended float64 `json:"recommended"` // 推荐值
	Reason      string  `json:"reason"`      // 原因
	Impact      string  `json:"impact"`      // "high", "medium", "low"
	Priority    int     `json:"priority"`    // 优先级 1-5
}

// ============================================================================
// Reflection Store Interface
// ============================================================================

// ReflectionStore reflection storage interface
type ReflectionStore interface {
	// Reflection management
	SaveReflection(reflection *ReflectionRecord) error
	GetReflectionByID(id string) (*ReflectionRecord, error)
	GetRecentReflections(traderID string, limit int) ([]*ReflectionRecord, error)
	GetReflectionByPeriod(traderID string, startTime, endTime time.Time) (*ReflectionRecord, error)

	// System adjustment management
	SaveSystemAdjustment(adjustment *SystemAdjustment) error
	GetAdjustmentByID(id string) (*SystemAdjustment, error)
	GetAdjustmentsByStatus(traderID string, status string) ([]*SystemAdjustment, error)
	GetLatestAdjustment(traderID string) (*SystemAdjustment, error)
	UpdateAdjustmentStatus(id string, status string, appliedAt *time.Time) error
	GetAdjustmentHistory(traderID string, limit int) ([]*SystemAdjustment, error)

	// AI learning memory management
	SaveLearningMemory(memory *AILearningMemory) error
	GetActiveLearningMemory(traderID string) ([]*AILearningMemory, error)
	GetLearningMemoryByID(id string) (*AILearningMemory, error)
	GetLearningMemoriesByTrader(traderID string, limit int) ([]*AILearningMemory, error)
	GetLearningMemoryBySymbol(traderID, symbol string) ([]*AILearningMemory, error)
	UpdateMemoryUsage(id string) error
	DeleteExpiredMemory(traderID string) error
	GetLearningMemoryForPrompt(traderID string, symbol string) ([]string, error) // 获取要注入 prompt 的内容

	// Statistics
	GetReflectionStats(traderID string, days int) (map[string]interface{}, error)
}
