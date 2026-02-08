package handlers

import (
	"net/http"

	"vmmanager/internal/repository"

	"github.com/gin-gonic/gin"
)

type AuditHandler struct {
	auditRepo *repository.AuditLogRepository
}

func NewAuditHandler(auditRepo *repository.AuditLogRepository) *AuditHandler {
	return &AuditHandler{auditRepo: auditRepo}
}

type AuditLogResponse struct {
	ID           string `json:"id"`
	UserID       string `json:"userId"`
	Username     string `json:"username"`
	Action       string `json:"action"`
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
	Details      string `json:"details"`
	IPAddress    string `json:"ipAddress"`
	UserAgent    string `json:"userAgent"`
	Status       string `json:"status"`
	ErrorMessage string `json:"errorMessage"`
	CreatedAt    string `json:"createdAt"`
}

func (h *AuditHandler) ListAuditLogs(c *gin.Context) {
	ctx := c.Request.Context()

	page := 1
	pageSize := 50

	if p := c.Query("page"); p != "" {
		_, _ = c.GetQuery("page")
	}

	logs, total, err := h.auditRepo.List(ctx, (page-1)*pageSize, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "failed to fetch audit logs"})
		return
	}

	result := make([]AuditLogResponse, 0, len(logs))
	for _, log := range logs {
		username := ""
		if log.UserID != nil {
			username = log.UserID.String()[:8]
		}
		result = append(result, AuditLogResponse{
			ID:           log.ID.String(),
			UserID:       log.UserID.String(),
			Username:     username,
			Action:       log.Action,
			ResourceType: log.ResourceType,
			ResourceID:   log.ResourceID.String(),
			Details:      log.Details,
			IPAddress:    log.IPAddress.String(),
			UserAgent:    log.UserAgent,
			Status:       log.Status,
			ErrorMessage: log.ErrorMessage,
			CreatedAt:    log.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
		"meta": gin.H{
			"page":        page,
			"per_page":    pageSize,
			"total":       total,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

func (h *AuditHandler) GetAuditLog(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	log, err := h.auditRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "audit log not found"})
		return
	}

	username := ""
	if log.UserID != nil {
		username = log.UserID.String()[:8]
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": AuditLogResponse{
			ID:           log.ID.String(),
			UserID:       log.UserID.String(),
			Username:     username,
			Action:       log.Action,
			ResourceType: log.ResourceType,
			ResourceID:   log.ResourceID.String(),
			Details:      log.Details,
			IPAddress:    log.IPAddress.String(),
			UserAgent:    log.UserAgent,
			Status:       log.Status,
			ErrorMessage: log.ErrorMessage,
			CreatedAt:    log.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	})
}

func (h *AuditHandler) ListByUser(c *gin.Context) {
	userID := c.Param("id")
	ctx := c.Request.Context()

	page := 1
	pageSize := 50

	logs, total, err := h.auditRepo.ListByUser(ctx, userID, (page-1)*pageSize, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "failed to fetch audit logs"})
		return
	}

	result := make([]AuditLogResponse, 0, len(logs))
	for _, log := range logs {
		result = append(result, AuditLogResponse{
			ID:           log.ID.String(),
			UserID:       log.UserID.String(),
			Action:       log.Action,
			ResourceType: log.ResourceType,
			ResourceID:   log.ResourceID.String(),
			Details:      log.Details,
			IPAddress:    log.IPAddress.String(),
			UserAgent:    log.UserAgent,
			Status:       log.Status,
			ErrorMessage: log.ErrorMessage,
			CreatedAt:    log.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
		"meta": gin.H{
			"page":        page,
			"per_page":    pageSize,
			"total":       total,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

func (h *AuditHandler) ListByAction(c *gin.Context) {
	action := c.Param("action")
	ctx := c.Request.Context()

	page := 1
	pageSize := 50

	logs, total, err := h.auditRepo.ListByAction(ctx, action, (page-1)*pageSize, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "failed to fetch audit logs"})
		return
	}

	result := make([]AuditLogResponse, 0, len(logs))
	for _, log := range logs {
		result = append(result, AuditLogResponse{
			ID:           log.ID.String(),
			UserID:       log.UserID.String(),
			Action:       log.Action,
			ResourceType: log.ResourceType,
			ResourceID:   log.ResourceID.String(),
			Details:      log.Details,
			IPAddress:    log.IPAddress.String(),
			UserAgent:    log.UserAgent,
			Status:       log.Status,
			ErrorMessage: log.ErrorMessage,
			CreatedAt:    log.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
		"meta": gin.H{
			"page":        page,
			"per_page":    pageSize,
			"total":       total,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

func (h *AuditHandler) ExportAuditLogs(c *gin.Context) {
	ctx := c.Request.Context()

	logs, _, err := h.auditRepo.List(ctx, 0, 10000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    5001,
			"message": "failed to fetch audit logs",
			"details": err.Error(),
		})
		return
	}

	if len(logs) == 0 {
		c.String(http.StatusOK, "No audit logs found\n")
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=audit_logs.csv")

	c.String(http.StatusOK, "ID,UserID,Action,ResourceType,ResourceID,IPAddress,Status,CreatedAt\n")
	for _, log := range logs {
		username := ""
		if log.UserID != nil {
			username = log.UserID.String()
		}
		ipAddress := ""
		if log.IPAddress != nil {
			ipAddress = log.IPAddress.String()
		}
		c.String(http.StatusOK, "%s,%s,%s,%s,%s,%s,%s,%s\n",
			log.ID.String(),
			username,
			log.Action,
			log.ResourceType,
			log.ResourceID.String(),
			ipAddress,
			log.Status,
			log.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		)
	}
}
