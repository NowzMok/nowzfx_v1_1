package store

import (
	"time"

	"gorm.io/gorm"
)

// TPSLRecord 止盈止损记录
type TPSLRecord struct {
	ID             int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	TraderID       string     `gorm:"column:trader_id;not null;index:idx_tpsl_trader" json:"trader_id"`
	PositionID     int64      `gorm:"column:position_id;not null;index:idx_tpsl_position" json:"position_id"`
	Symbol         string     `gorm:"column:symbol;not null;index:idx_tpsl_symbol" json:"symbol"`
	Side           string     `gorm:"column:side;not null" json:"side"` // LONG or SHORT
	CurrentTP      float64    `gorm:"column:current_tp" json:"current_tp"`
	CurrentSL      float64    `gorm:"column:current_sl" json:"current_sl"`
	OriginalTP     float64    `gorm:"column:original_tp" json:"original_tp"`
	OriginalSL     float64    `gorm:"column:original_sl" json:"original_sl"`
	EntryPrice     float64    `gorm:"column:entry_price" json:"entry_price"`
	EntryQuantity  float64    `gorm:"column:entry_quantity" json:"entry_quantity"`
	TPTriggered    bool       `gorm:"column:tp_triggered;default:false" json:"tp_triggered"`
	SLTriggered    bool       `gorm:"column:sl_triggered;default:false" json:"sl_triggered"`
	TPTriggeredAt  *time.Time `gorm:"column:tp_triggered_at" json:"tp_triggered_at"`
	SLTriggeredAt  *time.Time `gorm:"column:sl_triggered_at" json:"sl_triggered_at"`
	ModifiedCount  int        `gorm:"column:modified_count;default:0" json:"modified_count"`
	LastModifiedAt *time.Time `gorm:"column:last_modified_at" json:"last_modified_at"`
	Status         string     `gorm:"column:status;default:ACTIVE" json:"status"` // ACTIVE, TRIGGERED, CLOSED
	CreatedAt      time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

// TableName returns the table name
func (TPSLRecord) TableName() string {
	return "tpsl_records"
}

// TPSLStore TP/SL 记录存储
type TPSLStore struct {
	db *gorm.DB
}

// NewTPSLStore 创建 TP/SL 存储实例
func NewTPSLStore(db *gorm.DB) *TPSLStore {
	return &TPSLStore{db: db}
}

// InitTables 初始化表
func (s *TPSLStore) InitTables() error {
	return s.db.AutoMigrate(&TPSLRecord{})
}

// SaveTPSLRecord 保存 TP/SL 记录
func (s *TPSLStore) SaveTPSLRecord(record *TPSLRecord) error {
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	record.UpdatedAt = time.Now().UTC()
	return s.db.Save(record).Error
}

// GetTPSLByPositionID 获取持仓的 TP/SL 记录
func (s *TPSLStore) GetTPSLByPositionID(positionID int64) (*TPSLRecord, error) {
	var record *TPSLRecord
	if err := s.db.Where("position_id = ?", positionID).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return record, nil
}

// GetTPSLBySymbolAndTrader 获取某个交易者某个币对的活跃 TP/SL（symbol 为空时返回所有活跃记录）
func (s *TPSLStore) GetTPSLBySymbolAndTrader(traderID, symbol string) ([]*TPSLRecord, error) {
	var records []*TPSLRecord
	query := s.db.Where("trader_id = ? AND status = 'ACTIVE'", traderID)
	if symbol != "" {
		query = query.Where("symbol = ?", symbol)
	}
	err := query.Find(&records).Error
	return records, err
}

// UpdateTPSL 更新 TP/SL 价格
func (s *TPSLStore) UpdateTPSL(recordID int64, newTP, newSL float64) error {
	now := time.Now().UTC()
	return s.db.Model(&TPSLRecord{}).
		Where("id = ?", recordID).
		Updates(map[string]interface{}{
			"current_tp":       newTP,
			"current_sl":       newSL,
			"modified_count":   gorm.Expr("modified_count + 1"),
			"last_modified_at": now,
			"updated_at":       now,
		}).Error
}

// UpdateTPSLTriggered 更新 TP/SL 触发状态
func (s *TPSLStore) UpdateTPSLTriggered(recordID int64, tpTriggered, slTriggered bool) error {
	updates := map[string]interface{}{
		"updated_at": time.Now().UTC(),
	}

	if tpTriggered {
		now := time.Now().UTC()
		updates["tp_triggered"] = true
		updates["tp_triggered_at"] = now
	}

	if slTriggered {
		now := time.Now().UTC()
		updates["sl_triggered"] = true
		updates["sl_triggered_at"] = now
	}

	return s.db.Model(&TPSLRecord{}).
		Where("id = ?", recordID).
		Updates(updates).Error
}

// UpdateTPSLStatus 更新 TP/SL 状态
func (s *TPSLStore) UpdateTPSLStatus(recordID int64, status string) error {
	return s.db.Model(&TPSLRecord{}).
		Where("id = ?", recordID).
		Update("status", status).Error
}

// GetAllActiveTPSL 获取所有活跃的 TP/SL 记录
func (s *TPSLStore) GetAllActiveTPSL() ([]*TPSLRecord, error) {
	var records []*TPSLRecord
	err := s.db.Where("status = 'ACTIVE'").Find(&records).Error
	return records, err
}

// CloseTPSL 关闭 TP/SL 记录
func (s *TPSLStore) CloseTPSL(positionID int64) error {
	return s.db.Model(&TPSLRecord{}).
		Where("position_id = ?", positionID).
		Update("status", "CLOSED").Error
}
