package handlers

import (
	"net/http"
	"strconv"
	"time"

	"vmmanager/internal/api/errors"
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
	pageSize := 20

	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	if ps := c.Query("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
			pageSize = v
		}
	}

	action := c.Query("action")
	status := c.Query("status")
	userID := c.Query("user_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var logs []repository.AuditLogWithUsername
	var total int64
	var err error

	if userID != "" {
		logs, total, err = h.auditRepo.ListByUserWithUsername(ctx, userID, (page-1)*pageSize, pageSize)
	} else if action != "" {
		logs, total, err = h.auditRepo.ListByActionWithUsername(ctx, action, (page-1)*pageSize, pageSize)
	} else if startDate != "" && endDate != "" {
		start, err1 := time.Parse("2006-01-02", startDate)
		end, err2 := time.Parse("2006-01-02", endDate)
		if err1 == nil && err2 == nil {
			logs, total, err = h.auditRepo.ListByDateRangeWithUsername(ctx, start, end, (page-1)*pageSize, pageSize)
		} else {
			logs, total, err = h.auditRepo.ListWithUsername(ctx, (page-1)*pageSize, pageSize)
		}
	} else {
		logs, total, err = h.auditRepo.ListWithUsername(ctx, (page-1)*pageSize, pageSize)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_fetch_audit_logs"), err.Error()))
		return
	}

	result := make([]AuditLogResponse, 0, len(logs))
	for _, log := range logs {
		var resourceID string
		if log.ResourceID != nil {
			resourceID = log.ResourceID.String()
		}
		result = append(result, AuditLogResponse{
			ID:           log.ID.String(),
			UserID:       log.UserID.String(),
			Username:     log.Username,
			Action:       log.Action,
			ResourceType: log.ResourceType,
			ResourceID:   resourceID,
			Details:      log.Details,
			IPAddress:    log.IPAddress,
			UserAgent:    log.UserAgent,
			Status:       log.Status,
			ErrorMessage: log.ErrorMessage,
			CreatedAt:    log.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	filteredResult := result
	if status != "" {
		filteredResult = make([]AuditLogResponse, 0)
		for _, r := range result {
			if r.Status == status {
				filteredResult = append(filteredResult, r)
			}
		}
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"list": filteredResult,
		"meta": gin.H{
			"page":        page,
			"per_page":    pageSize,
			"total":       total,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	}))
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
			IPAddress:    log.IPAddress,
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
			IPAddress:    log.IPAddress,
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
			IPAddress:    log.IPAddress,
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

func (h *AuditHandler) ExportAuditLogsCSV(c *gin.Context) {
	logs, _, err := h.auditRepo.ListWithUsername(c.Request.Context(), 0, 10000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_fetch_audit_logs"), err.Error()))
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=audit_logs.csv")

	c.String(http.StatusOK, "ID,UserID,Action,ResourceType,ResourceID,IPAddress,Status,CreatedAt\n")
	for _, log := range logs {
		username := ""
		if log.UserID != nil {
			username = log.UserID.String()
		}
		ipAddress := log.IPAddress
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
