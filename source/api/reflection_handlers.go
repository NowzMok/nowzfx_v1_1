package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"nofx/backtest"
	"nofx/logger"
	"nofx/store"

	"github.com/gin-gonic/gin"
)

// ReflectionHandlers handles reflection-related API requests
type ReflectionHandlers struct {
	scheduler *backtest.ReflectionScheduler
	store     *store.Store
}

// NewReflectionHandlers creates new reflection handlers
func NewReflectionHandlers(scheduler *backtest.ReflectionScheduler, store *store.Store) *ReflectionHandlers {
	return &ReflectionHandlers{
		scheduler: scheduler,
		store:     store,
	}
}

// RegisterReflectionRoutes registers reflection routes
func (h *ReflectionHandlers) RegisterReflectionRoutes(r *gin.Engine) {
	group := r.Group("/api/reflection")

	// Reflection endpoints (more specific routes first)
	group.GET("/:traderID/recent", h.GetRecentReflections)
	group.POST("/:traderID/analyze", h.TriggerReflection)
	group.POST("/:traderID/create", h.CreateReflection)
	group.GET("/:traderID/stats", h.GetReflectionStats)
	// Use a distinct prefix for lookup by reflection ID to avoid route wildcard conflicts
	group.GET("/id/:id", h.GetReflection)       // Lookup by reflection ID
	group.PUT("/id/:id", h.UpdateReflection)    // Update reflection
	group.DELETE("/id/:id", h.DeleteReflection) // Delete reflection

	// Adjustment endpoints
	adjGroup := r.Group("/api/adjustment")
	adjGroup.GET("/:traderID/pending", h.GetPendingAdjustments)
	adjGroup.GET("/:traderID/history", h.GetAdjustmentHistory)
	adjGroup.POST("/:id/apply", h.ApplyAdjustment)
	adjGroup.POST("/:id/reject", h.RejectAdjustment)
	adjGroup.POST("/:id/revert", h.RevertAdjustment)
	adjGroup.PUT("/:id", h.UpdateAdjustment) // Update adjustment

	// Learning memory endpoints
	memGroup := r.Group("/api/memory")
	memGroup.GET("/:traderID", h.GetLearningMemories)
	memGroup.DELETE("/:id", h.DeleteLearningMemory)

	logger.Infof("✅ Reflection routes registered")
}

// ============ Reflection Endpoints ============

// GetRecentReflections gets recent reflections for a trader
func (h *ReflectionHandlers) GetRecentReflections(c *gin.Context) {
	traderID := c.Param("traderID")
	limit := 10

	if l, err := strconv.Atoi(c.DefaultQuery("limit", "10")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	reflections, err := h.store.Reflection().GetRecentReflections(traderID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reflections"})
		return
	}

	if reflections == nil {
		reflections = []*store.ReflectionRecord{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   reflections,
		"count":  len(reflections),
		"trader": traderID,
	})
}

// GetReflection gets a specific reflection
func (h *ReflectionHandlers) GetReflection(c *gin.Context) {
	id := c.Param("id")

	reflection, err := h.store.Reflection().GetReflectionByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reflection not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": reflection,
	})
}

// TriggerReflection manually triggers reflection for a trader
func (h *ReflectionHandlers) TriggerReflection(c *gin.Context) {
	traderID := c.Param("traderID")

	// Validate trader exists
	trader, err := h.store.Trader().GetByID(traderID)
	if err != nil || trader == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Trader %s not found", traderID)})
		return
	}

	// Trigger reflection
	if err := h.scheduler.ManualTrigger(traderID); err != nil {
		logger.Errorf("Failed to trigger reflection: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to trigger reflection"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Reflection triggered successfully",
		"trader":  traderID,
	})
}

// GetReflectionStats gets reflection statistics
func (h *ReflectionHandlers) GetReflectionStats(c *gin.Context) {
	traderID := c.Param("traderID")
	days := 30

	if d, err := strconv.Atoi(c.DefaultQuery("days", "30")); err == nil && d > 0 && d <= 365 {
		days = d
	}

	stats, err := h.store.Reflection().GetReflectionStats(traderID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   stats,
		"trader": traderID,
		"days":   days,
	})
}

// ============ Adjustment Endpoints ============

// GetPendingAdjustments gets pending adjustments for a trader
func (h *ReflectionHandlers) GetPendingAdjustments(c *gin.Context) {
	traderID := c.Param("traderID")

	adjustments, err := h.store.Reflection().GetAdjustmentsByStatus(traderID, "PENDING")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch adjustments"})
		return
	}

	if adjustments == nil {
		adjustments = []*store.SystemAdjustment{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   adjustments,
		"count":  len(adjustments),
		"trader": traderID,
		"status": "PENDING",
	})
}

// ApplyAdjustment applies a pending adjustment
func (h *ReflectionHandlers) ApplyAdjustment(c *gin.Context) {
	id := c.Param("id")

	// Get adjustment
	adjustment, err := h.store.Reflection().GetAdjustmentByID(id)
	if err != nil || adjustment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Adjustment not found"})
		return
	}

	if adjustment.Status != "PENDING" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cannot apply adjustment with status: %s", adjustment.Status)})
		return
	}

	// Update status to APPLIED
	adjustment.Status = "APPLIED"
	now := time.Now().UTC()
	adjustment.AppliedAt = &now

	if err := h.store.Reflection().SaveSystemAdjustment(adjustment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply adjustment"})
		return
	}

	logger.Infof("📝 Adjustment %s applied by user", id)

	c.JSON(http.StatusOK, gin.H{
		"message": "Adjustment applied successfully",
		"id":      id,
		"status":  "APPLIED",
	})
}

// RejectAdjustment rejects a pending adjustment
func (h *ReflectionHandlers) RejectAdjustment(c *gin.Context) {
	id := c.Param("id")

	// Get adjustment
	adjustment, err := h.store.Reflection().GetAdjustmentByID(id)
	if err != nil || adjustment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Adjustment not found"})
		return
	}

	if adjustment.Status != "PENDING" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cannot reject adjustment with status: %s", adjustment.Status)})
		return
	}

	// Update status to REJECTED
	adjustment.Status = "REJECTED"
	now := time.Now().UTC()
	adjustment.AppliedAt = &now

	if err := h.store.Reflection().SaveSystemAdjustment(adjustment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject adjustment"})
		return
	}

	logger.Infof("❌ Adjustment %s rejected by user", id)

	c.JSON(http.StatusOK, gin.H{
		"message": "Adjustment rejected successfully",
		"id":      id,
		"status":  "REJECTED",
	})
}

// RevertAdjustment reverts an applied adjustment
func (h *ReflectionHandlers) RevertAdjustment(c *gin.Context) {
	id := c.Param("id")

	// Get adjustment
	adjustment, err := h.store.Reflection().GetAdjustmentByID(id)
	if err != nil || adjustment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Adjustment not found"})
		return
	}

	if adjustment.Status != "APPLIED" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cannot revert adjustment with status: %s", adjustment.Status)})
		return
	}

	// Update status to REVERTED
	adjustment.Status = "REVERTED"
	now := time.Now().UTC()
	adjustment.AppliedAt = &now

	if err := h.store.Reflection().SaveSystemAdjustment(adjustment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revert adjustment"})
		return
	}

	logger.Infof("⏮ Adjustment %s reverted by user", id)

	c.JSON(http.StatusOK, gin.H{
		"message": "Adjustment reverted successfully",
		"id":      id,
		"status":  "REVERTED",
	})
}

// GetAdjustmentHistory gets adjustment history for a trader
func (h *ReflectionHandlers) GetAdjustmentHistory(c *gin.Context) {
	traderID := c.Param("traderID")
	limit := 50

	if l, err := strconv.Atoi(c.DefaultQuery("limit", "50")); err == nil && l > 0 && l <= 500 {
		limit = l
	}

	// Get all adjustments (all statuses)
	adjustments, err := h.store.Reflection().GetAdjustmentsByStatus(traderID, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}

	if len(adjustments) > limit {
		adjustments = adjustments[:limit]
	}

	if adjustments == nil {
		adjustments = []*store.SystemAdjustment{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   adjustments,
		"count":  len(adjustments),
		"trader": traderID,
	})
}

// ============ Learning Memory Endpoints ============

// GetLearningMemories gets learning memories for a trader
func (h *ReflectionHandlers) GetLearningMemories(c *gin.Context) {
	traderID := c.Param("traderID")
	limit := 50

	if l, err := strconv.Atoi(c.DefaultQuery("limit", "50")); err == nil && l > 0 && l <= 500 {
		limit = l
	}

	memories, err := h.store.Reflection().GetLearningMemoriesByTrader(traderID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch memories"})
		return
	}

	if memories == nil {
		memories = []*store.AILearningMemory{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   memories,
		"count":  len(memories),
		"trader": traderID,
	})
}

// DeleteLearningMemory deletes a learning memory (archive)
func (h *ReflectionHandlers) DeleteLearningMemory(c *gin.Context) {
	id := c.Param("id")

	// Get memory
	memory, err := h.store.Reflection().GetLearningMemoryByID(id)
	if err != nil || memory == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Memory not found"})
		return
	}

	// Archive by setting expiration to now
	memory.ExpiresAt = time.Now().UTC()

	if err := h.store.Reflection().SaveLearningMemory(memory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to archive memory"})
		return
	}

	logger.Infof("🗂 Learning memory %s archived", id)

	c.JSON(http.StatusOK, gin.H{
		"message": "Memory archived successfully",
		"id":      id,
	})
}

// ============ Edit Reflection Endpoints ============

// CreateReflectionRequest is the request body for creating a new reflection
type CreateReflectionRequest struct {
	AnalysisType string `json:"analysisType" binding:"required,oneof=performance risk strategy"`
	Findings     string `json:"findings" binding:"required"`
	Severity     string `json:"severity" binding:"required,oneof=info warning error"`
}

// CreateReflection creates a new reflection manually
func (h *ReflectionHandlers) CreateReflection(c *gin.Context) {
	traderID := c.Param("traderID")
	var req CreateReflectionRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate trader exists
	trader, err := h.store.Trader().GetByID(traderID)
	if err != nil || trader == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Trader %s not found", traderID)})
		return
	}

	// Create reflection
	reflection := &store.ReflectionRecord{
		ID:             fmt.Sprintf("refl_%d", time.Now().UnixNano()),
		TraderID:       traderID,
		ReflectionTime: time.Now().UTC(),
		AIReflection:   fmt.Sprintf("[%s] %s", req.AnalysisType, req.Findings),
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	// Save to database
	if err := h.store.Reflection().SaveReflection(reflection); err != nil {
		logger.Errorf("Failed to save reflection: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create reflection"})
		return
	}

	logger.Infof("✏️ Reflection %s created manually by user", reflection.ID)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Reflection created successfully",
		"id":      reflection.ID,
		"data":    reflection,
	})
}

// UpdateReflectionRequest is the request body for updating a reflection
type UpdateReflectionRequest struct {
	AIReflection string `json:"findings"`
	Severity     string `json:"severity"`
}

// UpdateReflection updates an existing reflection
func (h *ReflectionHandlers) UpdateReflection(c *gin.Context) {
	id := c.Param("id")
	var req UpdateReflectionRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get reflection
	reflection, err := h.store.Reflection().GetReflectionByID(id)
	if err != nil || reflection == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reflection not found"})
		return
	}

	// Update fields
	if req.AIReflection != "" {
		reflection.AIReflection = req.AIReflection
	}
	reflection.UpdatedAt = time.Now().UTC()

	// Save to database
	if err := h.store.Reflection().SaveReflection(reflection); err != nil {
		logger.Errorf("Failed to update reflection: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update reflection"})
		return
	}

	logger.Infof("✏️ Reflection %s updated by user", id)

	c.JSON(http.StatusOK, gin.H{
		"message": "Reflection updated successfully",
		"id":      id,
		"data":    reflection,
	})
}

// DeleteReflection deletes a reflection
func (h *ReflectionHandlers) DeleteReflection(c *gin.Context) {
	id := c.Param("id")

	// Get reflection first
	reflection, err := h.store.Reflection().GetReflectionByID(id)
	if err != nil || reflection == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reflection not found"})
		return
	}

	// We'll soft-delete by setting AIReflection to "[DELETED]"
	reflection.AIReflection = "[DELETED]"
	reflection.UpdatedAt = time.Now().UTC()

	if err := h.store.Reflection().SaveReflection(reflection); err != nil {
		logger.Errorf("Failed to delete reflection: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete reflection"})
		return
	}

	logger.Infof("🗑️ Reflection %s deleted by user", id)

	c.JSON(http.StatusOK, gin.H{
		"message": "Reflection deleted successfully",
		"id":      id,
	})
}

// UpdateAdjustmentRequest is the request body for updating an adjustment
type UpdateAdjustmentRequest struct {
	AdjustmentReason string `json:"reasoning"`
	Priority         string `json:"priority"`
}

// UpdateAdjustment updates an adjustment
func (h *ReflectionHandlers) UpdateAdjustment(c *gin.Context) {
	id := c.Param("id")
	var req UpdateAdjustmentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get adjustment
	adjustment, err := h.store.Reflection().GetAdjustmentByID(id)
	if err != nil || adjustment == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Adjustment not found"})
		return
	}

	// Only allow editing pending adjustments
	if adjustment.Status != "PENDING" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Can only edit pending adjustments"})
		return
	}

	// Update fields
	if req.AdjustmentReason != "" {
		adjustment.AdjustmentReason = req.AdjustmentReason
	}
	if req.Priority != "" {
		// Priority maps to confidence level
		switch req.Priority {
		case "high":
			adjustment.ConfidenceLevel = 0.9
		case "medium":
			adjustment.ConfidenceLevel = 0.7
		case "low":
			adjustment.ConfidenceLevel = 0.5
		}
	}

	// Save to database
	if err := h.store.Reflection().SaveSystemAdjustment(adjustment); err != nil {
		logger.Errorf("Failed to update adjustment: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update adjustment"})
		return
	}

	logger.Infof("✏️ Adjustment %s updated by user", id)

	c.JSON(http.StatusOK, gin.H{
		"message": "Adjustment updated successfully",
		"id":      id,
		"data":    adjustment,
	})
}
