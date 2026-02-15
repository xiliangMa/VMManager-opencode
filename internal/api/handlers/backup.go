package handlers

import (
	"net/http"
	"strconv"
	"time"

	"vmmanager/internal/api/errors"
	"vmmanager/internal/models"
	"vmmanager/internal/repository"
	"vmmanager/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BackupHandler struct {
	repo    *repository.Repositories
	service *services.BackupService
}

func NewBackupHandler(repo *repository.Repositories, service *services.BackupService) *BackupHandler {
	return &BackupHandler{
		repo:    repo,
		service: service,
	}
}

func (h *BackupHandler) ListBackups(c *gin.Context) {
	ctx := c.Request.Context()
	vmID := c.Param("id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	backups, total, err := h.repo.VMBackup.ListByVM(ctx, vmID, (page-1)*pageSize, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "backup.failedToList"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.SuccessWithPage(backups, total, page, pageSize))
}

type CreateBackupRequest struct {
	Name        string     `json:"name" binding:"required"`
	Description string     `json:"description"`
	BackupType  string     `json:"backupType"`
	ScheduledAt *time.Time `json:"scheduledAt"`
	ExpiresAt   *time.Time `json:"expiresAt"`
}

func (h *BackupHandler) CreateBackup(c *gin.Context) {
	ctx := c.Request.Context()
	vmID := c.Param("id")

	var req CreateBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
		return
	}

	_, err := h.repo.VM.FindByID(ctx, vmID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "vm.vmNotFound")))
		return
	}

	backupType := req.BackupType
	if backupType == "" {
		backupType = "full"
	}

	userID, _ := c.Get("user_id")
	userUUID, _ := uuid.Parse(userID.(string))

	status := "pending"
	if req.ScheduledAt == nil {
		status = "pending"
	}

	backup := &models.VMBackup{
		VMID:        uuid.MustParse(vmID),
		Name:        req.Name,
		Description: req.Description,
		BackupType:  backupType,
		Status:      status,
		ScheduledAt: req.ScheduledAt,
		ExpiresAt:   req.ExpiresAt,
		CreatedBy:   &userUUID,
	}

	if err := h.repo.VMBackup.Create(ctx, backup); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "backup.failedToCreate"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(backup))
}

func (h *BackupHandler) GetBackup(c *gin.Context) {
	ctx := c.Request.Context()
	backupID := c.Param("backup_id")

	backup, err := h.repo.VMBackup.FindByID(ctx, backupID)
	if err != nil {
		if err == repository.ErrBackupNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "backup.notFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "backup.failedToGet"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(backup))
}

func (h *BackupHandler) DeleteBackup(c *gin.Context) {
	ctx := c.Request.Context()
	backupID := c.Param("backup_id")

	backup, err := h.repo.VMBackup.FindByID(ctx, backupID)
	if err != nil {
		if err == repository.ErrBackupNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "backup.notFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "backup.failedToGet"), err.Error()))
		return
	}

	if backup.Status == "running" {
		c.JSON(http.StatusBadRequest, errors.FailWithCode(errors.ErrCodeBadRequest, t(c, "backup.cannotDeleteRunning")))
		return
	}

	if h.service != nil {
		if err := h.service.DeleteBackup(backupID); err != nil {
			c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "backup.failedToDelete"), err.Error()))
			return
		}
	} else {
		if err := h.repo.VMBackup.Delete(ctx, backupID); err != nil {
			c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "backup.failedToDelete"), err.Error()))
			return
		}
	}

	c.JSON(http.StatusOK, errors.Success(nil))
}

func (h *BackupHandler) RestoreBackup(c *gin.Context) {
	ctx := c.Request.Context()
	backupID := c.Param("backup_id")
	vmID := c.Param("id")

	backup, err := h.repo.VMBackup.FindByID(ctx, backupID)
	if err != nil {
		if err == repository.ErrBackupNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "backup.notFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "backup.failedToGet"), err.Error()))
		return
	}

	if backup.Status != "completed" {
		c.JSON(http.StatusBadRequest, errors.FailWithCode(errors.ErrCodeBadRequest, t(c, "backup.canOnlyRestoreCompleted")))
		return
	}

	if h.service != nil {
		if err := h.service.RestoreBackup(backupID, vmID); err != nil {
			c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "backup.failedToRestore"), err.Error()))
			return
		}
	}

	c.JSON(http.StatusOK, errors.Success(map[string]string{
		"message":  "Backup restored successfully",
		"backupId": backupID,
	}))
}

func (h *BackupHandler) ListSchedules(c *gin.Context) {
	ctx := c.Request.Context()
	vmID := c.Param("id")

	schedules, err := h.repo.BackupSchedule.ListByVM(ctx, vmID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "backup.failedToListSchedules"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(schedules))
}

type CreateScheduleRequest struct {
	Name       string `json:"name" binding:"required"`
	CronExpr   string `json:"cronExpr" binding:"required"`
	BackupType string `json:"backupType"`
	Retention  int    `json:"retention"`
	Enabled    *bool  `json:"enabled"`
}

func (h *BackupHandler) CreateSchedule(c *gin.Context) {
	ctx := c.Request.Context()
	vmID := c.Param("id")

	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
		return
	}

	_, err := h.repo.VM.FindByID(ctx, vmID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "vm.vmNotFound")))
		return
	}

	backupType := req.BackupType
	if backupType == "" {
		backupType = "full"
	}

	retention := req.Retention
	if retention <= 0 {
		retention = 7
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	userID, _ := c.Get("user_id")
	userUUID, _ := uuid.Parse(userID.(string))

	schedule := &models.BackupSchedule{
		VMID:       uuid.MustParse(vmID),
		Name:       req.Name,
		CronExpr:   req.CronExpr,
		BackupType: backupType,
		Retention:  retention,
		Enabled:    enabled,
		CreatedBy:  &userUUID,
	}

	if err := h.repo.BackupSchedule.Create(ctx, schedule); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "backup.failedToCreateSchedule"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(schedule))
}

func (h *BackupHandler) UpdateSchedule(c *gin.Context) {
	ctx := c.Request.Context()
	scheduleID := c.Param("schedule_id")

	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
		return
	}

	schedule, err := h.repo.BackupSchedule.FindByID(ctx, scheduleID)
	if err != nil {
		if err == repository.ErrScheduleNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "backup.scheduleNotFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "backup.failedToGetSchedule"), err.Error()))
		return
	}

	if req.Name != "" {
		schedule.Name = req.Name
	}
	if req.CronExpr != "" {
		schedule.CronExpr = req.CronExpr
	}
	if req.BackupType != "" {
		schedule.BackupType = req.BackupType
	}
	if req.Retention > 0 {
		schedule.Retention = req.Retention
	}
	if req.Enabled != nil {
		schedule.Enabled = *req.Enabled
	}

	if err := h.repo.BackupSchedule.Update(ctx, schedule); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "backup.failedToUpdateSchedule"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(schedule))
}

func (h *BackupHandler) DeleteSchedule(c *gin.Context) {
	ctx := c.Request.Context()
	scheduleID := c.Param("schedule_id")

	if err := h.repo.BackupSchedule.Delete(ctx, scheduleID); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "backup.failedToDeleteSchedule"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(nil))
}

func (h *BackupHandler) ToggleSchedule(c *gin.Context) {
	ctx := c.Request.Context()
	scheduleID := c.Param("schedule_id")

	schedule, err := h.repo.BackupSchedule.FindByID(ctx, scheduleID)
	if err != nil {
		if err == repository.ErrScheduleNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "backup.scheduleNotFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "backup.failedToGetSchedule"), err.Error()))
		return
	}

	schedule.Enabled = !schedule.Enabled
	if err := h.repo.BackupSchedule.Update(ctx, schedule); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "backup.failedToUpdateSchedule"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(schedule))
}
