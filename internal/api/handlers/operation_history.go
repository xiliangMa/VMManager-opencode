package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"vmmanager/internal/api/errors"
	"vmmanager/internal/models"
	"vmmanager/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OperationHistoryHandler struct {
	loginHistoryRepo          *repository.LoginHistoryRepository
	resourceChangeHistoryRepo *repository.ResourceChangeHistoryRepository
	vmOperationHistoryRepo    *repository.VMOperationHistoryRepository
}

func NewOperationHistoryHandler(repos *repository.Repositories) *OperationHistoryHandler {
	return &OperationHistoryHandler{
		loginHistoryRepo:          repos.LoginHistory,
		resourceChangeHistoryRepo: repos.ResourceChangeHistory,
		vmOperationHistoryRepo:    repos.VMOperationHistory,
	}
}

type LoginHistoryResponse struct {
	ID              string `json:"id"`
	UserID          string `json:"userId"`
	Username        string `json:"username"`
	Email           string `json:"email"`
	LoginType       string `json:"loginType"`
	IPAddress       string `json:"ipAddress"`
	UserAgent       string `json:"userAgent"`
	Location        string `json:"location"`
	DeviceInfo      string `json:"deviceInfo"`
	Status          string `json:"status"`
	FailureReason   string `json:"failureReason"`
	LogoutAt        string `json:"logoutAt"`
	SessionDuration int    `json:"sessionDuration"`
	CreatedAt       string `json:"createdAt"`
}

func (h *OperationHistoryHandler) ListLoginHistories(c *gin.Context) {
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

	userID := c.Query("user_id")
	status := c.Query("status")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var logs []repository.LoginHistoryWithUser
	var total int64
	var err error

	if userID != "" {
		logs, total, err = h.loginHistoryRepo.ListByUserWithUser(ctx, userID, (page-1)*pageSize, pageSize)
	} else if startDate != "" && endDate != "" {
		start, err1 := time.Parse("2006-01-02", startDate)
		end, err2 := time.Parse("2006-01-02", endDate)
		if err1 == nil && err2 == nil {
			logs, total, err = h.loginHistoryRepo.ListWithUser(ctx, (page-1)*pageSize, pageSize)
			var filteredLogs []repository.LoginHistoryWithUser
			for _, log := range logs {
				if log.CreatedAt.After(start) && log.CreatedAt.Before(end.Add(24*time.Hour)) {
					filteredLogs = append(filteredLogs, log)
				}
			}
			logs = filteredLogs
		} else {
			logs, total, err = h.loginHistoryRepo.ListWithUser(ctx, (page-1)*pageSize, pageSize)
		}
	} else {
		logs, total, err = h.loginHistoryRepo.ListWithUser(ctx, (page-1)*pageSize, pageSize)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_fetch_login_histories"), err.Error()))
		return
	}

	result := make([]LoginHistoryResponse, 0, len(logs))
	for _, log := range logs {
		var logoutAt string
		if log.LogoutAt != nil {
			logoutAt = log.LogoutAt.Format("2006-01-02T15:04:05Z07:00")
		}
		result = append(result, LoginHistoryResponse{
			ID:              log.ID.String(),
			UserID:          log.UserID.String(),
			Username:        log.Username,
			Email:           log.Email,
			LoginType:       log.LoginType,
			IPAddress:       log.IPAddress,
			UserAgent:       log.UserAgent,
			Location:        log.Location,
			DeviceInfo:      log.DeviceInfo,
			Status:          log.Status,
			FailureReason:   log.FailureReason,
			LogoutAt:        logoutAt,
			SessionDuration: log.SessionDuration,
			CreatedAt:       log.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	filteredResult := result
	if status != "" {
		filteredResult = make([]LoginHistoryResponse, 0)
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

func (h *OperationHistoryHandler) GetLoginHistory(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	log, err := h.loginHistoryRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "login history not found"})
		return
	}

	var logoutAt string
	if log.LogoutAt != nil {
		logoutAt = log.LogoutAt.Format("2006-01-02T15:04:05Z07:00")
	}

	c.JSON(http.StatusOK, errors.Success(LoginHistoryResponse{
		ID:              log.ID.String(),
		UserID:          log.UserID.String(),
		LoginType:       log.LoginType,
		IPAddress:       log.IPAddress,
		UserAgent:       log.UserAgent,
		Location:        log.Location,
		DeviceInfo:      log.DeviceInfo,
		Status:          log.Status,
		FailureReason:   log.FailureReason,
		LogoutAt:        logoutAt,
		SessionDuration: log.SessionDuration,
		CreatedAt:       log.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}))
}

func (h *OperationHistoryHandler) RecordLoginHistory(userID uuid.UUID, loginType, ipAddress, userAgent, status, failureReason string) error {
	ctx := context.Background()
	history := &models.LoginHistory{
		UserID:        userID,
		LoginType:     loginType,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
		Status:        status,
		FailureReason: failureReason,
	}
	return h.loginHistoryRepo.Create(ctx, history)
}

type ResourceChangeHistoryResponse struct {
	ID           string `json:"id"`
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
	ResourceName string `json:"resourceName"`
	Action       string `json:"action"`
	OldValue     string `json:"oldValue"`
	NewValue     string `json:"newValue"`
	ChangedBy    string `json:"changedBy"`
	Username     string `json:"username"`
	ChangeReason string `json:"changeReason"`
	IPAddress    string `json:"ipAddress"`
	UserAgent    string `json:"userAgent"`
	CreatedAt    string `json:"createdAt"`
}

func (h *OperationHistoryHandler) ListResourceChangeHistories(c *gin.Context) {
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

	resourceType := c.Query("resource_type")
	resourceID := c.Query("resource_id")
	action := c.Query("action")

	var histories []repository.ResourceChangeHistoryWithUser
	var total int64
	var err error

	if resourceType != "" && resourceID != "" {
		histories, total, err = h.resourceChangeHistoryRepo.ListByResourceWithUser(ctx, resourceType, resourceID, (page-1)*pageSize, pageSize)
	} else if action != "" {
		var rawHistories []models.ResourceChangeHistory
		rawHistories, total, err = h.resourceChangeHistoryRepo.ListByAction(ctx, action, (page-1)*pageSize, pageSize)
		histories = make([]repository.ResourceChangeHistoryWithUser, len(rawHistories))
		for i, h := range rawHistories {
			histories[i] = repository.ResourceChangeHistoryWithUser{
				ResourceChangeHistory: h,
			}
		}
	} else {
		histories, total, err = h.resourceChangeHistoryRepo.ListWithUser(ctx, (page-1)*pageSize, pageSize)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_fetch_resource_change_histories"), err.Error()))
		return
	}

	result := make([]ResourceChangeHistoryResponse, 0, len(histories))
	for _, history := range histories {
		var changedBy string
		if history.ChangedBy != nil {
			changedBy = history.ChangedBy.String()
		}
		result = append(result, ResourceChangeHistoryResponse{
			ID:           history.ID.String(),
			ResourceType: history.ResourceType,
			ResourceID:   history.ResourceID.String(),
			ResourceName: history.ResourceName,
			Action:       history.Action,
			OldValue:     history.OldValue,
			NewValue:     history.NewValue,
			ChangedBy:    changedBy,
			Username:     history.Username,
			ChangeReason: history.ChangeReason,
			IPAddress:    history.IPAddress,
			UserAgent:    history.UserAgent,
			CreatedAt:    history.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"list": result,
		"meta": gin.H{
			"page":        page,
			"per_page":    pageSize,
			"total":       total,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	}))
}

func (h *OperationHistoryHandler) GetResourceChangeHistory(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	history, err := h.resourceChangeHistoryRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "resource change history not found"})
		return
	}

	var changedBy string
	if history.ChangedBy != nil {
		changedBy = history.ChangedBy.String()
	}

	c.JSON(http.StatusOK, errors.Success(ResourceChangeHistoryResponse{
		ID:           history.ID.String(),
		ResourceType: history.ResourceType,
		ResourceID:   history.ResourceID.String(),
		ResourceName: history.ResourceName,
		Action:       history.Action,
		OldValue:     history.OldValue,
		NewValue:     history.NewValue,
		ChangedBy:    changedBy,
		ChangeReason: history.ChangeReason,
		IPAddress:    history.IPAddress,
		UserAgent:    history.UserAgent,
		CreatedAt:    history.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}))
}

type VMOperationHistoryResponse struct {
	ID            string `json:"id"`
	VMID          string `json:"vmId"`
	VMName        string `json:"vmName"`
	Operation     string `json:"operation"`
	Status        string `json:"status"`
	StartedAt     string `json:"startedAt"`
	CompletedAt   string `json:"completedAt"`
	Duration      int    `json:"duration"`
	TriggeredBy   string `json:"triggeredBy"`
	Username      string `json:"username"`
	IPAddress     string `json:"ipAddress"`
	UserAgent     string `json:"userAgent"`
	RequestParams string `json:"requestParams"`
	ResponseData  string `json:"responseData"`
	ErrorMessage  string `json:"errorMessage"`
}

func (h *OperationHistoryHandler) ListVMOperationHistories(c *gin.Context) {
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

	vmID := c.Query("vm_id")
	operation := c.Query("operation")
	status := c.Query("status")

	var histories []repository.VMOperationHistoryWithUser
	var total int64
	var err error

	if vmID != "" {
		histories, total, err = h.vmOperationHistoryRepo.ListByVMWithUser(ctx, vmID, (page-1)*pageSize, pageSize)
	} else if operation != "" {
		var rawHistories []models.VMOperationHistory
		rawHistories, total, err = h.vmOperationHistoryRepo.ListByOperation(ctx, operation, (page-1)*pageSize, pageSize)
		histories = make([]repository.VMOperationHistoryWithUser, len(rawHistories))
		for i, h := range rawHistories {
			histories[i] = repository.VMOperationHistoryWithUser{
				VMOperationHistory: h,
			}
		}
	} else {
		histories, total, err = h.vmOperationHistoryRepo.ListWithUser(ctx, (page-1)*pageSize, pageSize)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_fetch_vm_operation_histories"), err.Error()))
		return
	}

	result := make([]VMOperationHistoryResponse, 0, len(histories))
	for _, history := range histories {
		var completedAt string
		if history.CompletedAt != nil {
			completedAt = history.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
		}
		var triggeredBy string
		if history.TriggeredBy != nil {
			triggeredBy = history.TriggeredBy.String()
		}
		result = append(result, VMOperationHistoryResponse{
			ID:            history.ID.String(),
			VMID:          history.VMID.String(),
			VMName:        history.VMName,
			Operation:     history.Operation,
			Status:        history.Status,
			StartedAt:     history.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
			CompletedAt:   completedAt,
			Duration:      history.Duration,
			TriggeredBy:   triggeredBy,
			Username:      history.Username,
			IPAddress:     history.IPAddress,
			UserAgent:     history.UserAgent,
			RequestParams: history.RequestParams,
			ResponseData:  history.ResponseData,
			ErrorMessage:  history.ErrorMessage,
		})
	}

	filteredResult := result
	if status != "" {
		filteredResult = make([]VMOperationHistoryResponse, 0)
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

func (h *OperationHistoryHandler) GetVMOperationHistory(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	history, err := h.vmOperationHistoryRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "vm operation history not found"})
		return
	}

	var completedAt string
	if history.CompletedAt != nil {
		completedAt = history.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	var triggeredBy string
	if history.TriggeredBy != nil {
		triggeredBy = history.TriggeredBy.String()
	}

	c.JSON(http.StatusOK, errors.Success(VMOperationHistoryResponse{
		ID:            history.ID.String(),
		VMID:          history.VMID.String(),
		Operation:     history.Operation,
		Status:        history.Status,
		StartedAt:     history.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		CompletedAt:   completedAt,
		Duration:      history.Duration,
		TriggeredBy:   triggeredBy,
		IPAddress:     history.IPAddress,
		UserAgent:     history.UserAgent,
		RequestParams: history.RequestParams,
		ResponseData:  history.ResponseData,
		ErrorMessage:  history.ErrorMessage,
	}))
}

func (h *OperationHistoryHandler) RecordVMOperation(vmID uuid.UUID, operation, status string, triggeredBy *uuid.UUID, ipAddress, userAgent, requestParams, responseData, errorMessage string) error {
	ctx := context.Background()
	history := &models.VMOperationHistory{
		VMID:          vmID,
		Operation:     operation,
		Status:        status,
		TriggeredBy:   triggeredBy,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
		RequestParams: requestParams,
		ResponseData:  responseData,
		ErrorMessage:  errorMessage,
	}
	return h.vmOperationHistoryRepo.Create(ctx, history)
}

func (h *OperationHistoryHandler) RecordResourceChange(resourceType string, resourceID uuid.UUID, resourceName, action, oldValue, newValue string, changedBy *uuid.UUID, changeReason, ipAddress, userAgent string) error {
	ctx := context.Background()
	history := &models.ResourceChangeHistory{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceName: resourceName,
		Action:       action,
		OldValue:     oldValue,
		NewValue:     newValue,
		ChangedBy:    changedBy,
		ChangeReason: changeReason,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	}
	return h.resourceChangeHistoryRepo.Create(ctx, history)
}
