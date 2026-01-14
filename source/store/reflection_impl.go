package store

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ReflectionImpl reflection data access implementation
type ReflectionImpl struct {
	db *gorm.DB
}

// NewReflectionImpl creates a new reflection store instance
func NewReflectionImpl(db *gorm.DB) ReflectionStore {
	return &ReflectionImpl{db: db}
}

// InitSchema initializes reflection tables
func (r *ReflectionImpl) InitSchema() error {
	tables := []interface{}{
		&ReflectionRecord{},
		&SystemAdjustment{},
		&AILearningMemory{},
	}

	for _, table := range tables {
		if err := r.db.AutoMigrate(table); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", table, err)
		}
	}

	// Create indices
	if err := r.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_reflection_period 
		ON reflections(trader_id, reflection_time DESC);
	`).Error; err != nil {
		return fmt.Errorf("failed to create reflection indices: %w", err)
	}

	if err := r.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_adjustment_status 
		ON system_adjustments(trader_id, status);
	`).Error; err != nil {
		return fmt.Errorf("failed to create adjustment indices: %w", err)
	}

	if err := r.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_memory_active 
		ON ai_learning_memory(trader_id, expires_at DESC);
	`).Error; err != nil {
		return fmt.Errorf("failed to create memory indices: %w", err)
	}

	return nil
}

// ============================================================================
// Reflection Management
// ============================================================================

// SaveReflection saves reflection record
func (r *ReflectionImpl) SaveReflection(reflection *ReflectionRecord) error {
	if reflection.ID == "" {
		reflection.ID = generateUUID()
	}
	if reflection.CreatedAt.IsZero() {
		reflection.CreatedAt = time.Now().UTC()
	}
	reflection.UpdatedAt = time.Now().UTC()

	return r.db.Save(reflection).Error
}

// GetReflectionByID gets reflection by ID
func (r *ReflectionImpl) GetReflectionByID(id string) (*ReflectionRecord, error) {
	var reflection ReflectionRecord
	if err := r.db.Where("id = ?", id).First(&reflection).Error; err != nil {
		return nil, err
	}
	return &reflection, nil
}

// GetRecentReflections gets recent reflections
func (r *ReflectionImpl) GetRecentReflections(traderID string, limit int) ([]*ReflectionRecord, error) {
	var reflections []*ReflectionRecord
	if err := r.db.
		Where("trader_id = ?", traderID).
		Order("reflection_time DESC").
		Limit(limit).
		Find(&reflections).Error; err != nil {
		return nil, err
	}
	return reflections, nil
}

// GetReflectionByPeriod gets reflection for a period
func (r *ReflectionImpl) GetReflectionByPeriod(traderID string, startTime, endTime time.Time) (*ReflectionRecord, error) {
	var reflection ReflectionRecord
	if err := r.db.
		Where("trader_id = ? AND period_start_time = ? AND period_end_time = ?", traderID, startTime, endTime).
		First(&reflection).Error; err != nil {
		return nil, err
	}
	return &reflection, nil
}

// ============================================================================
// System Adjustment Management
// ============================================================================

// SaveSystemAdjustment saves system adjustment
func (r *ReflectionImpl) SaveSystemAdjustment(adjustment *SystemAdjustment) error {
	if adjustment.ID == "" {
		adjustment.ID = generateUUID()
	}
	if adjustment.CreatedAt.IsZero() {
		adjustment.CreatedAt = time.Now().UTC()
	}
	if adjustment.Status == "" {
		adjustment.Status = "PENDING"
	}

	return r.db.Save(adjustment).Error
}

// GetAdjustmentByID gets adjustment by ID
func (r *ReflectionImpl) GetAdjustmentByID(id string) (*SystemAdjustment, error) {
	var adjustment SystemAdjustment
	if err := r.db.Where("id = ?", id).First(&adjustment).Error; err != nil {
		return nil, err
	}
	return &adjustment, nil
}

// GetAdjustmentsByStatus gets adjustments by status
func (r *ReflectionImpl) GetAdjustmentsByStatus(traderID string, status string) ([]*SystemAdjustment, error) {
	var adjustments []*SystemAdjustment
	query := r.db.Where("trader_id = ?", traderID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.
		Order("adjustment_time DESC").
		Find(&adjustments).Error; err != nil {
		return nil, err
	}
	return adjustments, nil
}

// GetLatestAdjustment gets latest adjustment
func (r *ReflectionImpl) GetLatestAdjustment(traderID string) (*SystemAdjustment, error) {
	var adjustment SystemAdjustment
	if err := r.db.
		Where("trader_id = ?", traderID).
		Order("adjustment_time DESC").
		First(&adjustment).Error; err != nil {
		return nil, err
	}
	return &adjustment, nil
}

// UpdateAdjustmentStatus updates adjustment status
func (r *ReflectionImpl) UpdateAdjustmentStatus(id string, status string, appliedAt *time.Time) error {
	return r.db.
		Model(&SystemAdjustment{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{"status": status, "applied_at": appliedAt}).Error
}

// GetAdjustmentHistory gets adjustment history
func (r *ReflectionImpl) GetAdjustmentHistory(traderID string, limit int) ([]*SystemAdjustment, error) {
	var adjustments []*SystemAdjustment
	if err := r.db.
		Where("trader_id = ?", traderID).
		Order("adjustment_time DESC").
		Limit(limit).
		Find(&adjustments).Error; err != nil {
		return nil, err
	}
	return adjustments, nil
}

// ============================================================================
// AI Learning Memory Management
// ============================================================================

// SaveLearningMemory saves learning memory
func (r *ReflectionImpl) SaveLearningMemory(memory *AILearningMemory) error {
	if memory.ID == "" {
		memory.ID = generateUUID()
	}
	if memory.CreatedAt.IsZero() {
		memory.CreatedAt = time.Now().UTC()
	}
	if memory.ExpiresAt.IsZero() {
		memory.ExpiresAt = time.Now().UTC().AddDate(0, 1, 0) // 默认 1 个月过期
	}
	memory.UpdatedAt = time.Now().UTC()

	return r.db.Save(memory).Error
}

// GetActiveLearningMemory gets active learning memory
func (r *ReflectionImpl) GetActiveLearningMemory(traderID string) ([]*AILearningMemory, error) {
	var memories []*AILearningMemory
	if err := r.db.
		Where("trader_id = ? AND expires_at > ?", traderID, time.Now().UTC()).
		Order("confidence DESC, usage_count DESC").
		Find(&memories).Error; err != nil {
		return nil, err
	}
	return memories, nil
}

// GetLearningMemoryByID gets learning memory by ID
func (r *ReflectionImpl) GetLearningMemoryByID(id string) (*AILearningMemory, error) {
	var memory AILearningMemory
	if err := r.db.Where("id = ?", id).First(&memory).Error; err != nil {
		return nil, err
	}
	return &memory, nil
}

// GetLearningMemoriesByTrader gets all learning memories by trader
func (r *ReflectionImpl) GetLearningMemoriesByTrader(traderID string, limit int) ([]*AILearningMemory, error) {
	var memories []*AILearningMemory
	if err := r.db.
		Where("trader_id = ? AND expires_at > ?", traderID, time.Now().UTC()).
		Order("confidence DESC, created_at DESC").
		Limit(limit).
		Find(&memories).Error; err != nil {
		return nil, err
	}
	return memories, nil
}

// GetLearningMemoryBySymbol gets learning memory by symbol
func (r *ReflectionImpl) GetLearningMemoryBySymbol(traderID, symbol string) ([]*AILearningMemory, error) {
	var memories []*AILearningMemory
	if err := r.db.
		Where("trader_id = ? AND (symbol = ? OR symbol = '') AND expires_at > ?", traderID, symbol, time.Now().UTC()).
		Order("confidence DESC").
		Find(&memories).Error; err != nil {
		return nil, err
	}
	return memories, nil
}

// UpdateMemoryUsage updates memory usage
func (r *ReflectionImpl) UpdateMemoryUsage(id string) error {
	now := time.Now().UTC()
	return r.db.
		Model(&AILearningMemory{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"usage_count":  gorm.Expr("usage_count + 1"),
			"last_used_at": now,
		}).Error
}

// DeleteExpiredMemory deletes expired memories
func (r *ReflectionImpl) DeleteExpiredMemory(traderID string) error {
	return r.db.
		Where("trader_id = ? AND expires_at <= ?", traderID, time.Now().UTC()).
		Delete(&AILearningMemory{}).Error
}

// GetLearningMemoryForPrompt gets learning memory content for prompt injection
func (r *ReflectionImpl) GetLearningMemoryForPrompt(traderID string, symbol string) ([]string, error) {
	var memories []*AILearningMemory
	if err := r.db.
		Where("trader_id = ? AND (symbol = ? OR symbol = '') AND expires_at > ? AND confidence >= 0.6",
			traderID, symbol, time.Now().UTC()).
		Order("confidence DESC").
		Limit(10).
		Find(&memories).Error; err != nil {
		return nil, err
	}

	var prompts []string
	for _, m := range memories {
		if m.PromptInjection != "" {
			prompts = append(prompts, m.PromptInjection)
		}
	}
	return prompts, nil
}

// ============================================================================
// Statistics
// ============================================================================

// GetReflectionStats gets reflection statistics
func (r *ReflectionImpl) GetReflectionStats(traderID string, days int) (map[string]interface{}, error) {
	startTime := time.Now().UTC().AddDate(0, 0, -days)

	var reflections []*ReflectionRecord
	if err := r.db.
		Where("trader_id = ? AND reflection_time >= ?", traderID, startTime).
		Order("reflection_time DESC").
		Find(&reflections).Error; err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_reflections": len(reflections),
		"total_trades":      0,
		"avg_success_rate":  0.0,
		"avg_pnl":           0.0,
		"total_pnl":         0.0,
		"best_day":          nil,
		"worst_day":         nil,
	}

	if len(reflections) == 0 {
		return stats, nil
	}

	totalTrades := 0
	totalSuccessRate := 0.0
	totalPnL := 0.0
	bestPnL := 0.0
	worstPnL := 0.0
	var bestReflection, worstReflection *ReflectionRecord

	for _, refl := range reflections {
		totalTrades += refl.TotalTrades
		totalSuccessRate += refl.SuccessRate
		totalPnL += refl.TotalPnL

		if refl.TotalPnL > bestPnL || bestReflection == nil {
			bestPnL = refl.TotalPnL
			bestReflection = refl
		}
		if refl.TotalPnL < worstPnL || worstReflection == nil {
			worstPnL = refl.TotalPnL
			worstReflection = refl
		}
	}

	stats["total_trades"] = totalTrades
	stats["avg_success_rate"] = totalSuccessRate / float64(len(reflections))
	stats["avg_pnl"] = totalPnL / float64(len(reflections))
	stats["total_pnl"] = totalPnL

	if bestReflection != nil {
		best := map[string]interface{}{
			"time":         bestReflection.ReflectionTime,
			"pnl":          bestReflection.TotalPnL,
			"success_rate": bestReflection.SuccessRate,
			"trades":       bestReflection.TotalTrades,
		}
		stats["best_day"] = best
	}

	if worstReflection != nil {
		worst := map[string]interface{}{
			"time":         worstReflection.ReflectionTime,
			"pnl":          worstReflection.TotalPnL,
			"success_rate": worstReflection.SuccessRate,
			"trades":       worstReflection.TotalTrades,
		}
		stats["worst_day"] = worst
	}

	return stats, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// 注意: generateUUID 已在 analysis_impl.go 中定义，这里不再重复定义
