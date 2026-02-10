package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"syscall"

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

func (h *AdminHandler) GetSystemResources(libvirtClient any) gin.HandlerFunc {
	return func(c *gin.Context) {
		cpuPercent, _ := getCPUUsage()
		totalMem, usedMem, _ := getMemoryUsage()
		totalDisk, usedDisk, _ := getDiskUsage()

		memPercent := float64(0)
		if totalMem > 0 {
			memPercent = float64(usedMem) / float64(totalMem) * 100
		}

		diskPercent := float64(0)
		if totalDisk > 0 {
			diskPercent = float64(usedDisk) / float64(totalDisk) * 100
		}

		c.JSON(http.StatusOK, errors.Success(gin.H{
			"cpu_percent":     cpuPercent,
			"memory_percent":  memPercent,
			"disk_percent":    diskPercent,
			"total_memory_mb": totalMem,
			"used_memory_mb":  usedMem,
			"total_disk_gb":   totalDisk,
			"used_disk_gb":    usedDisk,
		}))
	}
}

func getCPUUsage() (float64, error) {
	content, err := readFile("/proc/stat")
	if err != nil {
		return 0, nil
	}

	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return 0, nil
	}

	fields := strings.Fields(lines[0])
	if len(fields) < 8 {
		return 0, nil
	}

	var user, nice, system, idle uint64
	fmt.Sscanf(lines[0], "cpu %d %d %d %d", &user, &nice, &system, &idle)

	total := user + nice + system + idle
	if total == 0 {
		return 0, nil
	}

	usage := float64(user+nice+system) / float64(total) * 100
	return usage, nil
}

func getMemoryUsage() (total, used int, err error) {
	content, err := readFile("/proc/meminfo")
	if err != nil {
		return 0, 0, nil
	}

	var memTotal, memAvailable int
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "MemTotal:") {
			fmt.Sscanf(line, "MemTotal: %d kB", &memTotal)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			fmt.Sscanf(line, "MemAvailable: %d kB", &memAvailable)
		}
	}

	if memTotal > 0 {
		used = memTotal - memAvailable
		return memTotal / 1024, used / 1024, nil
	}
	return 0, 0, nil
}

func getDiskUsage() (total, used int, err error) {
	var stat syscall.Statfs_t
	err = syscall.Statfs("/", &stat)
	if err != nil {
		return 0, 0, err
	}

	totalBytes := stat.Blocks * uint64(stat.Bsize)
	freeBytes := stat.Bfree * uint64(stat.Bsize)
	usedBytes := totalBytes - freeBytes

	total = int(totalBytes / 1024 / 1024 / 1024)
	used = int(usedBytes / 1024 / 1024 / 1024)
	return total, used, nil
}

func readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
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
			"code":    0,
			"message": "success",
			"data": gin.H{
				"total_vms":       totalVMs,
				"running_vms":     runningVMSCount,
				"total_users":     totalUsers,
				"total_templates": totalTemplates,
			},
		})
	}
}
