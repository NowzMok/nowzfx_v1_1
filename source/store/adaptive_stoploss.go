package store

import (
	"time"
)

// AdaptiveStopLossRecord 动态止损记录
type AdaptiveStopLossRecord struct {
	ID              string    `json:"id" gorm:"primaryKey"`
	TraderID        string    `json:"trader_id"`
	Symbol          string    `json:"symbol"`
	PositionID      string    `json:"position_id"`       // 关联的持仓ID
	EntryPrice      float64   `json:"entry_price"`       // 入场价格
	CurrentStopLoss float64   `json:"current_stop_loss"` // 当前止损价格
	InitialStopLoss float64   `json:"initial_stop_loss"` // 初始止损价格
	TakeProfit      float64   `json:"take_profit"`       // 止盈价格（保持不变）
	CurrentPrice    float64   `json:"current_price"`     // 当前价格
	IsInProfit      bool      `json:"is_in_profit"`      // 是否盈利
	ProfitDistance  float64   `json:"profit_distance"`   // 盈利距离
	TimeProgression float64   `json:"time_progression"`  // 时间进度 (0-1)
	ElapsedSeconds  int       `json:"elapsed_seconds"`   // 已过秒数
	Status          string    `json:"status"`            // ACTIVE / CLOSED
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TableName 表名
func (AdaptiveStopLossRecord) TableName() string {
	return "adaptive_stoploss_records"
}

// AdaptiveStopLossStore 动态止损数据访问接口
type AdaptiveStopLossStore interface {
	SaveRecord(record *AdaptiveStopLossRecord) error
	GetActiveRecords(traderID string) ([]*AdaptiveStopLossRecord, error)
	GetRecordBySymbol(traderID, symbol string) (*AdaptiveStopLossRecord, error)
	UpdateRecord(record *AdaptiveStopLossRecord) error
	CloseRecord(id string) error
	CloseRecordBySymbol(traderID, symbol string) error // 新增：按币种关闭ASL记录
	DeleteClosedRecords(traderID string) error
}
