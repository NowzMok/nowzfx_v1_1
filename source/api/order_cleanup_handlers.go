package api

import (
	"fmt"
	"net/http"
	"nofx/logger"
	"nofx/store"
	"nofx/trader"

	"github.com/gin-gonic/gin"
)

// OrderCleanupHandlers 订单清理API处理器
type OrderCleanupHandlers struct {
	store *store.Store
}

// NewOrderCleanupHandlers 创建订单清理API处理器
func NewOrderCleanupHandlers(store *store.Store) *OrderCleanupHandlers {
	return &OrderCleanupHandlers{
		store: store,
	}
}

// RegisterRoutes 注册路由
func (h *OrderCleanupHandlers) RegisterRoutes(router *gin.RouterGroup) {
	orderGroup := router.Group("/order-cleanup")
	{
		orderGroup.GET("/stats", h.GetOrderStats)
		orderGroup.POST("/clean-duplicates", h.CleanDuplicates)
		orderGroup.POST("/clean-expired", h.CleanExpired)
		orderGroup.POST("/clean-old", h.CleanOldRecords)
		orderGroup.POST("/auto-clean", h.AutoClean)
	}
}

// GetOrderStats 获取订单统计信息
// @Summary 获取订单统计信息
// @Description 获取当前交易员的所有订单统计信息，包括各状态订单数量和重复情况
// @Tags 订单清理
// @Accept json
// @Produce json
// @Param trader_id query string true "交易员ID"
// @Success 200 {object} map[string]interface{} "订单统计信息"
// @Failure 400 {object} map[string]string "参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /order-cleanup/stats [get]
func (h *OrderCleanupHandlers) GetOrderStats(c *gin.Context) {
	traderID := c.Query("trader_id")
	if traderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trader_id is required"})
		return
	}

	dm := trader.NewOrderDeduplicationManager(traderID, h.store)
	stats, err := dm.GetOrderStats()
	if err != nil {
		logger.Errorf("Failed to get order stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get stats: %v", err)})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// CleanDuplicates 清理重复订单
// @Summary 清理重复订单
// @Description 清理同币种的重复订单，保留置信度最高的订单
// @Tags 订单清理
// @Accept json
// @Produce json
// @Param trader_id query string true "交易员ID"
// @Success 200 {object} map[string]interface{} "清理结果"
// @Failure 400 {object} map[string]string "参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /order-cleanup/clean-duplicates [post]
func (h *OrderCleanupHandlers) CleanDuplicates(c *gin.Context) {
	traderID := c.Query("trader_id")
	if traderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trader_id is required"})
		return
	}

	dm := trader.NewOrderDeduplicationManager(traderID, h.store)
	count, err := dm.CleanDuplicateOrders()
	if err != nil {
		logger.Errorf("Failed to clean duplicates: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to clean duplicates: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Duplicate orders cleaned successfully",
		"cleaned_count": count,
	})
}

// CleanExpired 清理过期订单
// @Summary 清理过期订单
// @Description 清理已过期的待执行订单
// @Tags 订单清理
// @Accept json
// @Produce json
// @Param trader_id query string true "交易员ID"
// @Success 200 {object} map[string]interface{} "清理结果"
// @Failure 400 {object} map[string]string "参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /order-cleanup/clean-expired [post]
func (h *OrderCleanupHandlers) CleanExpired(c *gin.Context) {
	traderID := c.Query("trader_id")
	if traderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trader_id is required"})
		return
	}

	dm := trader.NewOrderDeduplicationManager(traderID, h.store)
	count, err := dm.CleanExpiredOrders()
	if err != nil {
		logger.Errorf("Failed to clean expired orders: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to clean expired orders: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Expired orders cleaned successfully",
		"cleaned_count": count,
	})
}

// CleanOldRecords 清理旧记录
// @Summary 清理旧订单记录
// @Description 清理超过7天的已成交和已取消订单记录
// @Tags 订单清理
// @Accept json
// @Produce json
// @Param trader_id query string true "交易员ID"
// @Success 200 {object} map[string]interface{} "清理结果"
// @Failure 400 {object} map[string]string "参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /order-cleanup/clean-old [post]
func (h *OrderCleanupHandlers) CleanOldRecords(c *gin.Context) {
	traderID := c.Query("trader_id")
	if traderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trader_id is required"})
		return
	}

	dm := trader.NewOrderDeduplicationManager(traderID, h.store)
	count, err := dm.CleanFilledAndCancelledOrders()
	if err != nil {
		logger.Errorf("Failed to clean old records: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to clean old records: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Old records cleaned successfully",
		"cleaned_count": count,
	})
}

// AutoClean 自动清理所有
// @Summary 自动清理所有订单
// @Description 执行完整的订单清理流程：重复订单、过期订单、旧记录
// @Tags 订单清理
// @Accept json
// @Produce json
// @Param trader_id query string true "交易员ID"
// @Success 200 {object} map[string]interface{} "清理结果"
// @Failure 400 {object} map[string]string "参数错误"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /order-cleanup/auto-clean [post]
func (h *OrderCleanupHandlers) AutoClean(c *gin.Context) {
	traderID := c.Query("trader_id")
	if traderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trader_id is required"})
		return
	}

	dm := trader.NewOrderDeduplicationManager(traderID, h.store)
	results, err := dm.AutoClean()
	if err != nil {
		logger.Errorf("Failed to auto clean: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to auto clean: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Auto cleanup completed successfully",
		"results": results,
	})
}
