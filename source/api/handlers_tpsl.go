package api

import (
	"fmt"
	"net/http"
	"nofx/logger"
	"nofx/store"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// handleModifyTPSL 修改止盈止损
func (s *Server) handleModifyTPSL(c *gin.Context) {
	userID := c.GetString("user_id")
	traderID := c.Param("id")

	var req struct {
		PositionID int64   `json:"position_id" binding:"required"`
		NewTP      float64 `json:"new_tp" binding:"required"`
		NewSL      float64 `json:"new_sl" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter error: position_id, new_tp, new_sl are required"})
		return
	}

	logger.Infof("📝 User %s (trader=%s) requested to modify TP/SL: position=%d, newTP=%.2f, newSL=%.2f",
		userID, traderID, req.PositionID, req.NewTP, req.NewSL)

	// 获取持仓信息
	position, err := s.store.Position().GetByID(req.PositionID)
	if err != nil || position == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Position not found"})
		return
	}

	// 验证持仓是否属于该交易者
	if position.TraderID != traderID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Position does not belong to this trader"})
		return
	}

	// 检查持仓是否仍开放
	if position.Status != "OPEN" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Position is not open"})
		return
	}

	// 获取或创建 TP/SL 记录
	tpslRecord, err := s.store.TPSL().GetTPSLByPositionID(req.PositionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get TP/SL record"})
		return
	}

	// 如果没有 TP/SL 记录，创建新的
	if tpslRecord == nil {
		tpslRecord = &store.TPSLRecord{
			TraderID:      traderID,
			PositionID:    req.PositionID,
			Symbol:        position.Symbol,
			Side:          position.Side,
			CurrentTP:     req.NewTP,
			CurrentSL:     req.NewSL,
			OriginalTP:    req.NewTP,
			OriginalSL:    req.NewSL,
			EntryPrice:    position.EntryPrice,
			EntryQuantity: position.Quantity,
			Status:        "ACTIVE",
			CreatedAt:     time.Now().UTC(),
		}
		if err := s.store.TPSL().SaveTPSLRecord(tpslRecord); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create TP/SL record"})
			return
		}
		logger.Infof("✅ Created new TP/SL record for position %d: TP=%.2f, SL=%.2f", req.PositionID, req.NewTP, req.NewSL)
	} else {
		// 更新现有 TP/SL
		if err := s.store.TPSL().UpdateTPSL(tpslRecord.ID, req.NewTP, req.NewSL); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update TP/SL record"})
			return
		}
		logger.Infof("✅ Updated TP/SL record %d: old TP=%.2f->%.2f, SL=%.2f->%.2f",
			tpslRecord.ID, tpslRecord.CurrentTP, req.NewTP, tpslRecord.CurrentSL, req.NewSL)
	}

	// 尝试在交易所修改 TP/SL（如果支持）
	if err := s.modifyTPSLOnExchange(traderID, position, req.NewTP, req.NewSL); err != nil {
		logger.Warnf("⚠️ Failed to modify TP/SL on exchange: %v", err)
		// 记录修改，但不失败
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "TP/SL modified successfully",
		"symbol":  position.Symbol,
		"side":    position.Side,
		"new_tp":  req.NewTP,
		"new_sl":  req.NewSL,
	})
}

// modifyTPSLOnExchange 在交易所修改止盈止损
func (s *Server) modifyTPSLOnExchange(traderID string, position *store.TraderPosition, newTP, newSL float64) error {
	// 获取交易者配置
	fullConfig, err := s.store.Trader().GetFullConfig("", traderID) // 用户ID可能不可用，使用空值
	if err != nil || fullConfig == nil || fullConfig.Exchange == nil {
		return fmt.Errorf("cannot get trader configuration")
	}

	exchangeCfg := fullConfig.Exchange

	// 注意：实际的 TP/SL 修改需要交易者实现相应的接口
	// 这里我们主要是记录到数据库，实际的交易所同步交给监控线程处理
	logger.Infof("  📡 TP/SL will be synced to %s via monitoring loop", exchangeCfg.ExchangeType)
	return nil
}

// GetTPSLBySymbol 获取某个币对的所有活跃 TP/SL 记录
func (s *Server) GetTPSLBySymbol(traderID, symbol string) ([]*store.TPSLRecord, error) {
	return s.store.TPSL().GetTPSLBySymbolAndTrader(traderID, symbol)
}

// handleGetTPSLHistory API 端点：获取 TP/SL 修改历史（可选）
func (s *Server) handleGetTPSLHistory(c *gin.Context) {
	traderID := c.Query("trader_id")
	if traderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trader_id is required"})
		return
	}

	positionIDStr := c.Query("position_id")
	if positionIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "position_id is required"})
		return
	}

	positionID, err := strconv.ParseInt(positionIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid position_id"})
		return
	}

	record, err := s.store.TPSL().GetTPSLByPositionID(positionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get TP/SL record"})
		return
	}

	if record == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No TP/SL record found for this position"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             record.ID,
		"position_id":    record.PositionID,
		"symbol":         record.Symbol,
		"side":           record.Side,
		"current_tp":     record.CurrentTP,
		"current_sl":     record.CurrentSL,
		"original_tp":    record.OriginalTP,
		"original_sl":    record.OriginalSL,
		"entry_price":    record.EntryPrice,
		"tp_triggered":   record.TPTriggered,
		"sl_triggered":   record.SLTriggered,
		"modified_count": record.ModifiedCount,
		"status":         record.Status,
		"created_at":     record.CreatedAt,
		"updated_at":     record.UpdatedAt,
	})
}

// handleGetTPSLRecords API 端点：获取交易者的所有 TP/SL 记录
func (s *Server) handleGetTPSLRecords(c *gin.Context) {
	traderID := c.Param("id")
	if traderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trader_id is required"})
		return
	}

	// 获取所有活跃的 TP/SL 记录
	records, err := s.store.TPSL().GetTPSLBySymbolAndTrader(traderID, "")
	if err != nil {
		logger.Warnf("⚠️ Failed to get TP/SL records: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get TP/SL records"})
		return
	}

	// 构建响应
	var responseRecords []map[string]interface{}
	for _, record := range records {
		responseRecords = append(responseRecords, map[string]interface{}{
			"id":               record.ID,
			"trader_id":        record.TraderID,
			"position_id":      record.PositionID,
			"symbol":           record.Symbol,
			"side":             record.Side,
			"current_tp":       record.CurrentTP,
			"current_sl":       record.CurrentSL,
			"original_tp":      record.OriginalTP,
			"original_sl":      record.OriginalSL,
			"entry_price":      record.EntryPrice,
			"entry_quantity":   record.EntryQuantity,
			"tp_triggered":     record.TPTriggered,
			"sl_triggered":     record.SLTriggered,
			"tp_triggered_at":  record.TPTriggeredAt,
			"sl_triggered_at":  record.SLTriggeredAt,
			"modified_count":   record.ModifiedCount,
			"last_modified_at": record.LastModifiedAt,
			"status":           record.Status,
			"created_at":       record.CreatedAt,
			"updated_at":       record.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, responseRecords)
}
