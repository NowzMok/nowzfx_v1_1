package store

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// AnalysisRecord AI 分析记录（保存 AI 分析结果）
type AnalysisRecord struct {
	ID              string        `json:"id" gorm:"primaryKey"`
	TraderID        string        `json:"trader_id"`
	Symbol          string        `json:"symbol"`
	TargetPrice     float64       `json:"target_price"`
	SupportLevels   SupportLevels `json:"support_levels" gorm:"type:json"`
	ResistanceLevel float64       `json:"resistance_level"`
	Confidence      float64       `json:"confidence"`      // 0.0 - 1.0
	AnalysisReason  string        `json:"analysis_reason"` // "RSI超卖，资金流入..."
	AnalysisPrompt  string        `json:"analysis_prompt"` // 原始提示词
	AIResponse      string        `json:"ai_response"`     // AI 原始响应
	AnalysisTime    time.Time     `json:"analysis_time"`
	CreatedAt       time.Time     `json:"created_at"`
	ExpiresAt       time.Time     `json:"expires_at"` // 分析有效期（4小时后过期）
	Status          string        `json:"status"`     // ACTIVE / EXPIRED / REPLACED
}

// TableName 表名
func (AnalysisRecord) TableName() string {
	return "ai_analysis"
}

// SupportLevels 支撑位数组
type SupportLevels []float64

// Value 实现 GORM driver.Valuer 接口
func (sl SupportLevels) Value() (driver.Value, error) {
	return json.Marshal(sl)
}

// Scan 实现 GORM sql.Scanner 接口
func (sl *SupportLevels) Scan(value interface{}) error {
	bytes, _ := value.([]byte)
	return json.Unmarshal(bytes, &sl)
}

// PendingOrder 待执行订单
type PendingOrder struct {
	ID               string     `json:"id" gorm:"primaryKey"`
	TraderID         string     `json:"trader_id"`
	Symbol           string     `json:"symbol"`
	AnalysisID       string     `json:"analysis_id"`   // 关联的分析记录
	TargetPrice      float64    `json:"target_price"`  // AI 目标价
	TriggerPrice     float64    `json:"trigger_price"` // 实际触发价格
	PositionSize     float64    `json:"position_size"` // USDT 单位
	Leverage         int        `json:"leverage"`
	StopLoss         float64    `json:"stop_loss"`       // SL 价格
	TakeProfit       float64    `json:"take_profit"`     // TP 价格
	Confidence       float64    `json:"confidence"`      // 置信度
	Status           string     `json:"status"`          // PENDING / TRIGGERED / FILLED / CANCELLED / EXPIRED
	TriggeredPrice   float64    `json:"triggered_price"` // 实际触发的价格
	TriggeredAt      *time.Time `json:"triggered_at"`
	FilledAt         *time.Time `json:"filled_at"`
	ExecutedAt       *time.Time `json:"executed_at"`                                                 // 执行完成时间
	IsExecuting      bool       `gorm:"column:is_executing;default:false" json:"is_executing"`       // 是否正在执行（防止重复）
	ExecutionVersion int64      `gorm:"column:execution_version;default:0" json:"execution_version"` // 执行版本（原子操作）
	CreatedAt        time.Time  `json:"created_at"`
	ExpiresAt        time.Time  `json:"expires_at"`    // 订单有效期（1天）
	CancelReason     string     `json:"cancel_reason"` // 取消原因
	OrderID          int64      `json:"order_id"`      // 关联的交易所订单 ID
}

// TableName 表名
func (PendingOrder) TableName() string {
	return "pending_orders"
}

// TradeHistoryRecord 交易历史（已平仓交易）
type TradeHistoryRecord struct {
	ID             string    `json:"id" gorm:"primaryKey"`
	TraderID       string    `json:"trader_id"`
	Symbol         string    `json:"symbol"`
	AnalysisID     string    `json:"analysis_id"`      // 关联的分析记录
	PendingOrderID string    `json:"pending_order_id"` // 关联的待执行订单
	EntryPrice     float64   `json:"entry_price"`
	ExitPrice      float64   `json:"exit_price"`
	Quantity       float64   `json:"quantity"`
	Leverage       int       `json:"leverage"`
	RealizedPnL    float64   `json:"realized_pnl"` // 实际盈亏（USDT）
	PnL            float64   `json:"pnl"`          // 盈亏金额
	PnLPct         float64   `json:"pnl_pct"`      // 盈亏百分比
	Confidence     float64   `json:"confidence"`   // 原始分析的信心度 (0-1)
	EntryTime      time.Time `json:"entry_time"`
	ExitTime       time.Time `json:"exit_time"`
	HoldDuration   int64     `json:"hold_duration"` // 持仓时长（分钟）
	CreatedAt      time.Time `json:"created_at"`
}

// TableName 表名
func (TradeHistoryRecord) TableName() string {
	return "trade_history"
}

// AnalysisStore AI 分析数据访问接口
type AnalysisStore interface {
	// 分析记录相关
	SaveAnalysis(analysis *AnalysisRecord) error
	GetAnalysisByID(id string) (*AnalysisRecord, error)
	GetActiveAnalyses(traderID string) ([]*AnalysisRecord, error)
	GetAnalysesBySymbol(traderID, symbol string) (*AnalysisRecord, error)
	UpdateAnalysisStatus(id, status string) error
	DeleteExpiredAnalyses(traderID string) error

	// 待执行订单相关
	SavePendingOrder(order *PendingOrder) error
	GetPendingOrderByID(id string) (*PendingOrder, error)
	GetPendingOrdersByTrader(traderID string) ([]*PendingOrder, error)
	GetPendingOrdersByStatus(traderID, status string) ([]*PendingOrder, error)
	UpdatePendingOrderStatus(id, status string, triggeredPrice float64, triggeredAt time.Time) error
	UpdatePendingOrderFilled(id string, filledAt time.Time, orderID int64) error
	UpdatePendingOrderFilledWithPrice(id string, triggeredPrice float64, filledAt time.Time, orderID int64) error
	CancelPendingOrder(id, reason string) error
	DeleteExpiredPendingOrders(traderID string) error

	// 待决策系统增强功能
	MarkExpiredOrdersAsExpired(traderID string) (int64, error)                                  // 标记过期订单为 EXPIRED
	CleanupStaleOrders(traderID string, maxAge time.Duration) (int64, error)                    // 清理过时订单
	GetPendingOrderCount(traderID string) (int64, error)                                        // 获取待决策订单数量
	CancelOldestPendingOrders(traderID string, keepCount int) (int64, error)                    // 取消最旧的订单，保留最新的 N 个
	GetOrdersWithPriceDeviation(traderID string, maxDeviation float64) ([]*PendingOrder, error) // 获取价格偏离过大的订单

	// 原子性执行标记（防止竞态条件）
	TryMarkAsExecuting(orderID string) bool
	MarkAsExecuted(orderID string) error
	CancelExecution(orderID string) error

	// 交易历史相关
	SaveTradeHistory(trade *TradeHistoryRecord) error
	GetTradeHistoryByID(id string) (*TradeHistoryRecord, error)
	GetTradeHistoriesByTrader(traderID string, limit int) ([]*TradeHistoryRecord, error)
	GetTradeHistoriesBySymbol(traderID, symbol string, limit int) ([]*TradeHistoryRecord, error)
	GetTradeHistoryInPeriod(traderID string, startTime, endTime time.Time) ([]*TradeHistoryRecord, error)
}
