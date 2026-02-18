package handlers

import (
	"net/http"

	"microsoft_connector/internal/services"

	"github.com/gin-gonic/gin"
)

type BatchHandler struct {
	graphService *services.GraphService
}

func NewBatchHandler(gs *services.GraphService) *BatchHandler {
	return &BatchHandler{graphService: gs}
}

// POST /api/batch
func (h *BatchHandler) ExecuteBatch(c *gin.Context) {
	var body map[string]any
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.graphService.Post("/$batch", body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
