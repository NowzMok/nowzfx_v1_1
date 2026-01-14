package store

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// AdaptiveStopLossImpl 动态止损数据访问实现
type AdaptiveStopLossImpl struct {
	db *gorm.DB
}

// NewAdaptiveStopLossImpl 创建动态止损数据访问实例
func NewAdaptiveStopLossImpl(db *gorm.DB) AdaptiveStopLossStore {
	return &AdaptiveStopLossImpl{db: db}
}

// SaveRecord 保存动态止损记录
func (a *AdaptiveStopLossImpl) SaveRecord(record *AdaptiveStopLossRecord) error {
	if record.ID == "" {
		record.ID = generateUUID()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	record.UpdatedAt = time.Now().UTC()
	return a.db.Save(record).Error
}

// GetActiveRecords 获取活跃的动态止损记录
func (a *AdaptiveStopLossImpl) GetActiveRecords(traderID string) ([]*AdaptiveStopLossRecord, error) {
	var records []*AdaptiveStopLossRecord
	err := a.db.Where("trader_id = ? AND status = ?", traderID, "ACTIVE").
		Order("created_at DESC").
		Find(&records).Error
	return records, err
}

// GetRecordBySymbol 获取某交易对的动态止损记录
func (a *AdaptiveStopLossImpl) GetRecordBySymbol(traderID, symbol string) (*AdaptiveStopLossRecord, error) {
	var record AdaptiveStopLossRecord
	err := a.db.Where("trader_id = ? AND symbol = ? AND status = ?", traderID, symbol, "ACTIVE").
		Order("created_at DESC").
		First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

// UpdateRecord 更新动态止损记录
func (a *AdaptiveStopLossImpl) UpdateRecord(record *AdaptiveStopLossRecord) error {
	record.UpdatedAt = time.Now().UTC()
	return a.db.Save(record).Error
}

// CloseRecord 关闭动态止损记录
func (a *AdaptiveStopLossImpl) CloseRecord(id string) error {
	return a.db.Model(&AdaptiveStopLossRecord{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     "CLOSED",
			"updated_at": time.Now().UTC(),
		}).Error
}

// CloseRecordBySymbol 关闭特定交易对的ASL记录
func (a *AdaptiveStopLossImpl) CloseRecordBySymbol(traderID, symbol string) error {
	if traderID == "" || symbol == "" {
		return fmt.Errorf("trader_id and symbol cannot be empty")
	}

	result := a.db.Model(&AdaptiveStopLossRecord{}).
		Where("trader_id = ? AND symbol = ? AND status = ?", traderID, symbol, "ACTIVE").
		Updates(map[string]interface{}{
			"status":     "CLOSED",
			"updated_at": time.Now().UTC(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to close ASL record: %w", result.Error)
	}

	return nil
}

// DeleteClosedRecords 删除已关闭的记录
func (a *AdaptiveStopLossImpl) DeleteClosedRecords(traderID string) error {
	return a.db.Where("trader_id = ? AND status = ?", traderID, "CLOSED").
		Delete(&AdaptiveStopLossRecord{}).Error
}

// InitSchema 初始化表结构
func (a *AdaptiveStopLossImpl) InitSchema() error {
	if err := a.db.AutoMigrate(&AdaptiveStopLossRecord{}); err != nil {
		return fmt.Errorf("failed to migrate adaptive_stoploss_records table: %w", err)
	}

	// 创建索引
	_ = a.db.Exec(`CREATE INDEX IF NOT EXISTS idx_adaptive_trader_symbol ON adaptive_stoploss_records(trader_id, symbol)`)
	_ = a.db.Exec(`CREATE INDEX IF NOT EXISTS idx_adaptive_status ON adaptive_stoploss_records(status)`)

	return nil
}
