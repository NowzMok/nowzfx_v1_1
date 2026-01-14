/**
 * NOFX 策略管理 API 优化指南
 *
 * 本文件包含后端 API 的优化建议和完整实现示例
 */

// ============================================================================
// 1. 完整的配置验证实现 (api/strategy_validation.go)
// ============================================================================

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"nofx/store"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ValidationError 验证错误
type ValidationError struct {
	Field   string   `json:"field"`
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
}

// ConfigValidationResult 配置验证结果
type ConfigValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors"`
	Warnings []ValidationError `json:"warnings"`
}

// ValidateStrategyConfigFull 完整的策略配置验证
func ValidateStrategyConfigFull(config *store.StrategyConfig) ConfigValidationResult {
	result := ConfigValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}

	// ==================== 币种来源验证 ====================
	errs := validateCoinSource(&config.CoinSource)
	if len(errs) > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "coin_source",
			Code:    "INVALID_COIN_SOURCE",
			Message: "Invalid coin source configuration",
			Details: errs,
		})
	}

	// ==================== 技术指标验证 ====================
	errs, warns := validateIndicators(&config.Indicators)
	if len(errs) > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "indicators",
			Code:    "INVALID_INDICATORS",
			Message: "Invalid indicator configuration",
			Details: errs,
		})
	}
	if len(warns) > 0 {
		result.Warnings = append(result.Warnings, ValidationError{
			Field:   "indicators",
			Code:    "INDICATOR_WARNING",
			Message: "Indicator configuration warnings",
			Details: warns,
		})
	}

	// ==================== 风控参数验证 ====================
	errs = validateRiskControl(&config.RiskControl)
	if len(errs) > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "risk_control",
			Code:    "INVALID_RISK_CONTROL",
			Message: "Invalid risk control configuration",
			Details: errs,
		})
	}

	// ==================== Prompt 验证 ====================
	errs = validatePromptSections(&config.PromptSections)
	if len(errs) > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "prompt_sections",
			Code:    "INVALID_PROMPT",
			Message: "Invalid prompt configuration",
			Details: errs,
		})
	}

	return result
}

// validateCoinSource 验证币种来源配置
func validateCoinSource(config *store.CoinSourceConfig) []string {
	var errs []string

	// 来源类型必填
	if config.SourceType == "" {
		errs = append(errs, "Source type is required")
		return errs
	}

	// 验证来源类型有效值
	validTypes := map[string]bool{"static": true, "ai500": true, "oi_top": true, "mixed": true}
	if !validTypes[config.SourceType] {
		errs = append(errs, fmt.Sprintf("Invalid source type: %s", config.SourceType))
	}

	// 静态来源验证
	if config.SourceType == "static" || config.SourceType == "mixed" {
		if len(config.StaticCoins) == 0 {
			errs = append(errs, "At least one static coin is required")
		}
		if len(config.StaticCoins) > 50 {
			errs = append(errs, "Maximum 50 static coins allowed")
		}
	}

	// AI500 限制验证
	if config.UseAI500 && config.AI500Limit > 100 {
		errs = append(errs, "AI500 limit must not exceed 100")
	}

	// OI Top 限制验证
	if config.UseOITop && config.OITopLimit > 50 {
		errs = append(errs, "OI Top limit must not exceed 50")
	}

	return errs
}

// validateIndicators 验证技术指标配置
func validateIndicators(config *store.IndicatorConfig) ([]string, []string) {
	var errs, warns []string

	// K-line 验证
	if config.Klines.PrimaryTimeframe == "" {
		errs = append(errs, "Primary timeframe is required")
	} else {
		validTimeframes := map[string]bool{
			"1m": true, "3m": true, "5m": true, "15m": true,
			"30m": true, "1h": true, "4h": true, "1d": true, "1w": true,
		}
		if !validTimeframes[config.Klines.PrimaryTimeframe] {
			errs = append(errs, fmt.Sprintf("Invalid primary timeframe: %s", config.Klines.PrimaryTimeframe))
		}
	}

	// K-line 数量验证
	if config.Klines.PrimaryCount < 10 || config.Klines.PrimaryCount > 1000 {
		errs = append(errs, "Primary K-line count must be between 10 and 1000")
	}

	// 至少启用一个指标
	hasIndicator := config.EnableEMA || config.EnableMACD || config.EnableRSI ||
		config.EnableATR || config.EnableBOLL || config.EnableVolume ||
		config.EnableOI || config.EnableFundingRate || config.EnableRawKlines

	if !hasIndicator {
		errs = append(errs, "At least one indicator must be enabled")
	}

	// 指标周期验证
	if config.EnableEMA && len(config.EMAPeriods) == 0 {
		warns = append(warns, "EMA enabled but no periods specified, using defaults [20, 50]")
	}

	if config.EnableRSI && len(config.RSIPeriods) == 0 {
		warns = append(warns, "RSI enabled but no periods specified, using defaults [7, 14]")
	}

	// NofxOS 依赖检查
	needsNofxOS := config.EnableQuantData || config.EnableOIRanking ||
		config.EnableNetFlowRanking || config.EnablePriceRanking

	if needsNofxOS && config.NofxOSAPIKey == "" {
		errs = append(errs, "NofxOS API key is required for selected data sources")
	}

	// 外部数据源验证
	for i, ds := range config.ExternalDataSources {
		if ds.Name == "" {
			errs = append(errs, fmt.Sprintf("External data source %d: name is required", i))
		}
		if ds.URL == "" {
			errs = append(errs, fmt.Sprintf("External data source %d: URL is required", i))
		}
	}

	return errs, warns
}

// validateRiskControl 验证风控参数
func validateRiskControl(config *store.RiskControlConfig) []string {
	var errs []string

	// 单笔风险验证
	if config.SingleTradeLoss <= 0 || config.SingleTradeLoss > 100 {
		errs = append(errs, "Single trade loss must be between 0 and 100 (percent)")
	}

	// 日总风险验证
	if config.DailyMaxLoss <= 0 || config.DailyMaxLoss > 100 {
		errs = append(errs, "Daily max loss must be between 0 and 100 (percent)")
	}

	if config.DailyMaxLoss < config.SingleTradeLoss {
		errs = append(errs, "Daily max loss must be greater than single trade loss")
	}

	// 止盈止损验证
	if config.TakeProfitPercent < 0 || config.TakeProfitPercent > 500 {
		errs = append(errs, "Take profit percent must be between 0 and 500")
	}

	if config.StopLossPercent < 0 || config.StopLossPercent > 100 {
		errs = append(errs, "Stop loss percent must be between 0 and 100")
	}

	// 仓位管理验证
	if config.MaxPositionSize <= 0 || config.MaxPositionSize > 100 {
		errs = append(errs, "Max position size must be between 0 and 100 (percent)")
	}

	if config.MaxOpenPositions <= 0 || config.MaxOpenPositions > 50 {
		errs = append(errs, "Max open positions must be between 1 and 50")
	}

	return errs
}

// validatePromptSections 验证 Prompt 部分
func validatePromptSections(config *store.PromptSectionsConfig) []string {
	var errs []string

	// 如果定义了 Prompt，检查长度
	maxLength := 5000
	if len(config.RoleDefinition) > maxLength {
		errs = append(errs, fmt.Sprintf("Role definition exceeds %d characters", maxLength))
	}
	if len(config.TradingFrequency) > maxLength {
		errs = append(errs, fmt.Sprintf("Trading frequency exceeds %d characters", maxLength))
	}
	if len(config.EntryStandards) > maxLength {
		errs = append(errs, fmt.Sprintf("Entry standards exceeds %d characters", maxLength))
	}
	if len(config.DecisionProcess) > maxLength {
		errs = append(errs, fmt.Sprintf("Decision process exceeds %d characters", maxLength))
	}

	return errs
}

// ============================================================================
// 2. 新增验证端点 (api/strategy.go 中添加)
// ============================================================================

// handleValidateStrategyConfig 验证策略配置
func (s *Server) handleValidateStrategyConfig(c *gin.Context) {
	var config store.StrategyConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		SafeBadRequest(c, "Invalid request parameters")
		return
	}

	result := ValidateStrategyConfigFull(&config)

	c.JSON(http.StatusOK, result)
}

// ============================================================================
// 3. 部分更新支持 (PATCH 端点)
// ============================================================================

// StrategyPatchRequest 部分更新请求
type StrategyPatchRequest struct {
	Name          *string               `json:"name,omitempty"`
	Description   *string               `json:"description,omitempty"`
	Config        *store.StrategyConfig `json:"config,omitempty"`
	IsPublic      *bool                 `json:"is_public,omitempty"`
	ConfigVisible *bool                 `json:"config_visible,omitempty"`
}

// handlePatchStrategy 部分更新策略
func (s *Server) handlePatchStrategy(c *gin.Context) {
	userID := c.GetString("user_id")
	strategyID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 获取现有策略
	strategy, err := s.store.Strategy().Get(userID, strategyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Strategy not found"})
		return
	}

	// 检查是否是系统默认策略
	if strategy.IsDefault {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot modify system default strategy"})
		return
	}

	// 解析部分更新请求
	var patch StrategyPatchRequest
	if err := c.ShouldBindJSON(&patch); err != nil {
		SafeBadRequest(c, "Invalid request parameters")
		return
	}

	// 应用补丁
	if patch.Name != nil {
		strategy.Name = *patch.Name
	}
	if patch.Description != nil {
		strategy.Description = *patch.Description
	}
	if patch.IsPublic != nil {
		strategy.IsPublic = *patch.IsPublic
	}
	if patch.ConfigVisible != nil {
		strategy.ConfigVisible = *patch.ConfigVisible
	}

	// 如果更新配置，执行完整验证
	if patch.Config != nil {
		// 验证配置
		result := ValidateStrategyConfigFull(patch.Config)
		if !result.Valid {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid configuration",
				"details": result.Errors,
			})
			return
		}

		// 序列化配置
		configJSON, err := json.Marshal(patch.Config)
		if err != nil {
			SafeInternalError(c, "Failed to serialize configuration", err)
			return
		}
		strategy.Config = string(configJSON)
	}

	// 保存更新
	if err := s.store.Strategy().Update(strategy); err != nil {
		SafeInternalError(c, "Failed to update strategy", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Strategy updated successfully",
		"id":      strategy.ID,
	})
}

// ============================================================================
// 4. 配置对比端点
// ============================================================================

// handleGetStrategyDiff 获取策略差异
func (s *Server) handleGetStrategyDiff(c *gin.Context) {
	userID := c.GetString("user_id")
	strategyID := c.Param("id")

	// 查询参数：与指定版本对比
	compareID := c.DefaultQuery("compare_with", "")

	strategy, err := s.store.Strategy().Get(userID, strategyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Strategy not found"})
		return
	}

	var compareConfig store.StrategyConfig
	json.Unmarshal([]byte(strategy.Config), &compareConfig)

	if compareID != "" {
		// 与另一个版本对比
		compareStrategy, err := s.store.Strategy().Get(userID, compareID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Compare strategy not found"})
			return
		}
		json.Unmarshal([]byte(compareStrategy.Config), &compareConfig)
	}

	var currentConfig store.StrategyConfig
	json.Unmarshal([]byte(strategy.Config), &currentConfig)

	// 计算差异
	diff := computeConfigDiff(&compareConfig, &currentConfig)

	c.JSON(http.StatusOK, gin.H{
		"diff": diff,
	})
}

// computeConfigDiff 计算配置差异
func computeConfigDiff(old, new *store.StrategyConfig) map[string]interface{} {
	diff := make(map[string]interface{})

	// 比较每个字段...
	// 这是一个简化的实现

	oldJSON, _ := json.MarshalIndent(old, "", "  ")
	newJSON, _ := json.MarshalIndent(new, "", "  ")

	diff["old"] = string(oldJSON)
	diff["new"] = string(newJSON)

	return diff
}

// ============================================================================
// 5. 配置快照和版本管理
// ============================================================================

// StrategySnapshot 策略快照
type StrategySnapshot struct {
	ID          string `gorm:"primaryKey"`
	StrategyID  string `gorm:"index"`
	UserID      string `gorm:"index"`
	Config      string // JSON
	Name        string // 快照名称
	Description string
	CreatedAt   time.Time
	CreatedBy   string // 创建者用户 ID
}

// handleCreateStrategySnapshot 创建策略快照
func (s *Server) handleCreateStrategySnapshot(c *gin.Context) {
	userID := c.GetString("user_id")
	strategyID := c.Param("id")

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		SafeBadRequest(c, "Invalid request parameters")
		return
	}

	strategy, err := s.store.Strategy().Get(userID, strategyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Strategy not found"})
		return
	}

	snapshot := &StrategySnapshot{
		ID:          uuid.New().String(),
		StrategyID:  strategyID,
		UserID:      userID,
		Config:      strategy.Config,
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   time.Now(),
		CreatedBy:   userID,
	}

	// 保存快照
	if err := s.store.SaveSnapshot(snapshot); err != nil {
		SafeInternalError(c, "Failed to create snapshot", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      snapshot.ID,
		"message": "Snapshot created successfully",
	})
}

// handleRestoreStrategySnapshot 恢复策略快照
func (s *Server) handleRestoreStrategySnapshot(c *gin.Context) {
	userID := c.GetString("user_id")
	strategyID := c.Param("id")
	snapshotID := c.Param("snapshot_id")

	strategy, err := s.store.Strategy().Get(userID, strategyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Strategy not found"})
		return
	}

	snapshot, err := s.store.GetSnapshot(snapshotID)
	if err != nil || snapshot.StrategyID != strategyID {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snapshot not found"})
		return
	}

	// 验证快照配置
	var config store.StrategyConfig
	json.Unmarshal([]byte(snapshot.Config), &config)

	result := ValidateStrategyConfigFull(&config)
	if !result.Valid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Snapshot configuration is invalid",
			"details": result.Errors,
		})
		return
	}

	// 恢复配置
	strategy.Config = snapshot.Config
	if err := s.store.Strategy().Update(strategy); err != nil {
		SafeInternalError(c, "Failed to restore snapshot", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Strategy restored successfully",
	})
}

// ============================================================================
// 6. 更新路由 (server.go 中的 setupRoutes)
// ============================================================================

/*
在 setupRoutes 中添加以下路由:

// Strategy management - enhanced endpoints
protected.GET("/strategies", s.handleGetStrategies)
protected.GET("/strategies/:id", s.handleGetStrategy)
protected.POST("/strategies", s.handleCreateStrategy)
protected.PUT("/strategies/:id", s.handleUpdateStrategy)
protected.PATCH("/strategies/:id", s.handlePatchStrategy)  // 新增：部分更新
protected.DELETE("/strategies/:id", s.handleDeleteStrategy)
protected.POST("/strategies/:id/activate", s.handleActivateStrategy)
protected.POST("/strategies/:id/duplicate", s.handleDuplicateStrategy)

// Validation
protected.POST("/strategies/validate-config", s.handleValidateStrategyConfig)  // 新增

// Diff and comparison
protected.GET("/strategies/:id/diff", s.handleGetStrategyDiff)  // 新增

// Snapshots
protected.POST("/strategies/:id/snapshots", s.handleCreateStrategySnapshot)  // 新增
protected.GET("/strategies/:id/snapshots", s.handleListStrategySnapshots)    // 新增
protected.POST("/strategies/:id/snapshots/:snapshot_id/restore", s.handleRestoreStrategySnapshot)  // 新增
*/

// ============================================================================
// 7. 错误响应标准化
// ============================================================================

// ErrorResponse 标准化错误响应
type ErrorResponse struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// SafeBadRequest 返回 400 错误
func SafeBadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Code:      "BAD_REQUEST",
		Message:   message,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// SafeInternalError 返回 500 错误
func SafeInternalError(c *gin.Context, message string, err error) {
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Code:      "INTERNAL_ERROR",
		Message:   message,
		Details:   err.Error(),
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// SafeValidationError 返回验证错误
func SafeValidationError(c *gin.Context, errors []ValidationError) {
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Code:      "VALIDATION_ERROR",
		Message:   "Configuration validation failed",
		Details:   errors,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// ============================================================================
// 8. 性能优化建议
// ============================================================================

/*
1. 数据库查询优化：
   - 为 strategies 表添加索引: (user_id, is_active)
   - 为 strategy_snapshots 表添加索引: (strategy_id, created_at)

2. 缓存策略：
   - 使用 Redis 缓存激活的策略配置
   - 缓存键: strategy:{user_id}:active
   - TTL: 5 分钟

3. API 响应优化：
   - 只在列表端点返回必要字段
   - 在详情端点返回完整配置
   - 使用 gzip 压缩响应

4. 分页支持：
   - 策略列表支持分页 ?page=1&limit=20
   - 快照列表支持分页

示例缓存实现:

import "github.com/redis/go-redis/v9"

func (s *Server) getActiveStrategyWithCache(ctx context.Context, userID string) (*store.Strategy, error) {
	cacheKey := fmt.Sprintf("strategy:%s:active", userID)

	// 尝试从缓存读取
	val, err := s.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var strategy store.Strategy
		json.Unmarshal([]byte(val), &strategy)
		return &strategy, nil
	}

	// 缓存未命中，从数据库读取
	strategy, err := s.store.Strategy().GetActive(userID)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	data, _ := json.Marshal(strategy)
	s.redisClient.Set(ctx, cacheKey, string(data), 5*time.Minute)

	return strategy, nil
}
*/
