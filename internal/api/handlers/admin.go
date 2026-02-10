package handlers

import (
	"context"
	"log"
	"net/http"

	"vmmanager/internal/api/errors"
	"vmmanager/internal/models"
	"vmmanager/internal/repository"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	userRepo     *repository.UserRepository
	vmRepo       *repository.VMRepository
	templateRepo *repository.TemplateRepository
	auditRepo    *repository.AuditLogRepository
}

func NewAdminHandler(
	userRepo *repository.UserRepository,
	vmRepo *repository.VMRepository,
	templateRepo *repository.TemplateRepository,
	auditRepo *repository.AuditLogRepository,
) *AdminHandler {
	return &AdminHandler{
		userRepo:     userRepo,
		vmRepo:       vmRepo,
		templateRepo: templateRepo,
		auditRepo:    auditRepo,
	}
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	ctx := c.Request.Context()

	page := 1
	pageSize := 20

	if p := c.Query("page"); p != "" {
		_, _ = c.GetQuery("page")
	}

	users, total, err := h.userRepo.List(ctx, (page-1)*pageSize, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to fetch users", err.Error()))
		return
	}

	userResponses := make([]gin.H, 0, len(users))
	for _, user := range users {
		userResponses = append(userResponses, gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"role":       user.Role,
			"is_active":  user.IsActive,
			"created_at": user.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, errors.SuccessWithMeta(userResponses, gin.H{
		"page":        page,
		"per_page":    pageSize,
		"total":       total,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	}))
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
		Role     string `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, "validation error", err.Error()))
		return
	}

	ctx := c.Request.Context()

	existingUser, _ := h.userRepo.FindByUsername(ctx, req.Username)
	if existingUser != nil {
		c.JSON(http.StatusConflict, errors.FailWithDetails(errors.ErrCodeUserExists, "username already exists", req.Username))
		return
	}

	if req.Role == "" {
		req.Role = "user"
	}

	passwordHash, _ := hashPassword(req.Password)

	user := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		Role:         req.Role,
		IsActive:     true,
	}

	if err := h.userRepo.Create(ctx, user); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to create user", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, errors.Success(gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
	}))
}

func (h *AdminHandler) GetUser(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	user, err := h.userRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeUserNotFound, "user not found", id))
		return
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"id":        user.ID,
		"username":  user.Username,
		"email":     user.Email,
		"role":      user.Role,
		"is_active": user.IsActive,
		"quota": gin.H{
			"cpu":      user.QuotaCPU,
			"memory":   user.QuotaMemory,
			"disk":     user.QuotaDisk,
			"vm_count": user.QuotaVMCount,
		},
		"created_at": user.CreatedAt,
	}))
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	user, err := h.userRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeUserNotFound, "user not found", id))
		return
	}

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		IsActive *bool  `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, "validation error", err.Error()))
		return
	}

	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := h.userRepo.Update(ctx, user); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to update user", err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
	}))
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	_, err := h.userRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeUserNotFound, "user not found", id))
		return
	}

	if err := h.userRepo.Delete(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to delete user", err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(nil))
}

func (h *AdminHandler) UpdateUserQuota(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	var req struct {
		CPU     int `json:"cpu"`
		Memory  int `json:"memory"`
		Disk    int `json:"disk"`
		VMCount int `json:"vm_count"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, "validation error", err.Error()))
		return
	}

	if err := h.userRepo.UpdateQuota(ctx, id, req.CPU, req.Memory, req.Disk, req.VMCount); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to update quota", err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"quota": gin.H{
			"cpu":      req.CPU,
			"memory":   req.Memory,
			"disk":     req.Disk,
			"vm_count": req.VMCount,
		},
	}))
}

func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	user, err := h.userRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeUserNotFound, "user not found", id))
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, "validation error", err.Error()))
		return
	}

	user.Role = req.Role
	if err := h.userRepo.Update(ctx, user); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to update role", err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"id":   user.ID,
		"role": user.Role,
	}))
}

func (h *AdminHandler) ListAuditLogs(c *gin.Context) {
	ctx := c.Request.Context()

	page := 1
	pageSize := 100

	logs, total, err := h.auditRepo.List(ctx, (page-1)*pageSize, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to fetch audit logs", err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.SuccessWithMeta(logs, gin.H{
		"page":        page,
		"per_page":    pageSize,
		"total":       total,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	}))
}

func (h *AdminHandler) GetSystemInfo(libvirtClient any) gin.HandlerFunc {
	return func(c *gin.Context) {
		info := gin.H{
			"libvirt_connected": false,
		}

		c.JSON(http.StatusOK, errors.Success(info))
	}
}

func (h *AdminHandler) GetSystemStats(libvirtClient any) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()

		users, _, err := h.userRepo.List(ctx, 0, 0)
		totalUsers := int64(len(users))
		log.Printf("GetSystemStats: users=%d, err=%v", totalUsers, err)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		vms, _, err := h.vmRepo.List(ctx, 0, 0)
		totalVMs := int64(len(vms))
		log.Printf("GetSystemStats: vms=%d, err=%v", totalVMs, err)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		runningVMs, err := h.vmRepo.ListByStatus(ctx, "running")
		runningVMSCount := int64(len(runningVMs))
		log.Printf("GetSystemStats: runningVMs=%d, err=%v", runningVMSCount, err)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		templates, _, err := h.templateRepo.List(ctx, 0, 0)
		totalTemplates := int64(len(templates))
		log.Printf("GetSystemStats: templates=%d, err=%v", totalTemplates, err)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"message": "success",
			"data": gin.H{
				"total_vms":        totalVMs,
				"running_vms":      runningVMSCount,
				"total_users":      totalUsers,
				"total_templates":  totalTemplates,
			},
		})
	}
}
