package handlers

import (
	"net/http"
	"strconv"

	"vmmanager/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AlertHistoryHandler struct {
	repo *repository.AlertHistoryRepository
}

func NewAlertHistoryHandler(repo *repository.AlertHistoryRepository) *AlertHistoryHandler {
	return &AlertHistoryHandler{repo: repo}
}

func (h *AlertHistoryHandler) ListAlertHistories(c *gin.Context) {
	page := 1
	pageSize := 20

	var err error
	if p := c.Query("page"); p != "" {
		page, err = strconv.Atoi(p)
		if err != nil {
			page = 1
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		pageSize, err = strconv.Atoi(ps)
		if err != nil {
			pageSize = 20
		}
	}

	offset := (page - 1) * pageSize
	histories, total, err := h.repo.List(c.Request.Context(), offset, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to list alert histories",
			"data":    nil,
		})
		return
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    histories,
		"meta": gin.H{
			"page":        page,
			"per_page":    pageSize,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func (h *AlertHistoryHandler) GetAlertHistory(c *gin.Context) {
	id := c.Param("id")
	parsedID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid alert history ID",
			"data":    nil,
		})
		return
	}

	history, err := h.repo.GetByID(c.Request.Context(), parsedID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    40401,
			"message": "Alert history not found",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    history,
	})
}

func (h *AlertHistoryHandler) ResolveAlertHistory(c *gin.Context) {
	id := c.Param("id")
	parsedID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    40001,
			"message": "Invalid alert history ID",
			"data":    nil,
		})
		return
	}

	if err := h.repo.Resolve(c.Request.Context(), parsedID.String()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to resolve alert history",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	})
}

func (h *AlertHistoryHandler) GetActiveAlerts(c *gin.Context) {
	histories, err := h.repo.GetActive(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to get active alerts",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    histories,
	})
}

func (h *AlertHistoryHandler) GetAlertStats(c *gin.Context) {
	total, critical, warning, info, err := h.repo.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    50001,
			"message": "Failed to get alert stats",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"total":    total,
			"critical": critical,
			"warning":  warning,
			"info":     info,
		},
	})
}
