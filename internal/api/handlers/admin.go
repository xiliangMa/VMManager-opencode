package handlers

import (
	"net/http"

	"vmmanager/internal/libvirt"
	"vmmanager/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db *gorm.DB
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	var users []models.User
	query := h.db

	var page, pageSize int
	page = 1
	pageSize = 20

	var total int64
	query.Model(&models.User{}).Count(&total)

	offset := (page - 1) * pageSize
	query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    users,
		"meta": gin.H{
			"page":        page,
			"per_page":    pageSize,
			"total":       total,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
		Role     string `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 4001, "message": err.Error()})
		return
	}

	if req.Role == "" {
		req.Role = "user"
	}

	passwordHash, _ := hashPassword(req.Password)

	user := models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		Role:         req.Role,
		IsActive:     true,
	}

	h.db.Create(&user)

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}

func (h *AdminHandler) GetUser(c *gin.Context) {
	id := c.Param("id")
	userUUID, _ := uuid.Parse(id)

	var user models.User
	if err := h.db.First(&user, "id = ?", userUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    user,
	})
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	userUUID, _ := uuid.Parse(id)

	var user models.User
	if err := h.db.First(&user, "id = ?", userUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "user not found"})
		return
	}

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		IsActive *bool  `json:"is_active"`
	}

	c.ShouldBindJSON(&req)

	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	h.db.Save(&user)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    user,
	})
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	userUUID, _ := uuid.Parse(id)

	var user models.User
	if err := h.db.First(&user, "id = ?", userUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "user not found"})
		return
	}

	h.db.Delete(&user)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
}

func (h *AdminHandler) UpdateUserQuota(c *gin.Context) {
	id := c.Param("id")
	userUUID, _ := uuid.Parse(id)

	var user models.User
	if err := h.db.First(&user, "id = ?", userUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "user not found"})
		return
	}

	var req struct {
		CPU     int `json:"cpu"`
		Memory  int `json:"memory"`
		Disk    int `json:"disk"`
		VMCount int `json:"vm_count"`
	}

	c.ShouldBindJSON(&req)

	user.QuotaCPU = req.CPU
	user.QuotaMemory = req.Memory
	user.QuotaDisk = req.Disk
	user.QuotaVMCount = req.VMCount

	h.db.Save(&user)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"quota": gin.H{
				"cpu":      user.QuotaCPU,
				"memory":   user.QuotaMemory,
				"disk":     user.QuotaDisk,
				"vm_count": user.QuotaVMCount,
			},
		},
	})
}

func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	id := c.Param("id")
	userUUID, _ := uuid.Parse(id)

	var user models.User
	if err := h.db.First(&user, "id = ?", userUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "user not found"})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}

	c.ShouldBindJSON(&req)

	user.Role = req.Role
	h.db.Save(&user)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"id":   user.ID,
			"role": user.Role,
		},
	})
}

func (h *AdminHandler) ListAuditLogs(c *gin.Context) {
	var logs []models.AuditLog
	query := h.db

	var total int64
	query.Model(&models.AuditLog{}).Count(&total)

	query.Order("created_at DESC").Limit(100).Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    logs,
		"meta": gin.H{
			"total": total,
		},
	})
}

func (h *AdminHandler) GetSystemInfo(libvirtClient *libvirt.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		info := gin.H{
			"libvirt_connected": false,
		}

		if libvirtClient != nil && libvirtClient.IsConnected() {
			hostInfo, _ := libvirtClient.GetHostInfo()
			info["libvirt_connected"] = true
			info["host"] = hostInfo
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data":    info,
		})
	}
}

func (h *AdminHandler) GetSystemStats(libvirtClient *libvirt.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var stats struct {
			TotalUsers     int `json:"total_users"`
			TotalVMs       int `json:"total_vms"`
			RunningVMs     int `json:"running_vms"`
			TotalTemplates int `json:"total_templates"`
		}

		h.db.Model(&models.User{}).Count(&stats.TotalUsers)
		h.db.Model(&models.VirtualMachine{}).Count(&stats.TotalVMs)
		h.db.Model(&models.VirtualMachine{}).Where("status = ?", "running").Count(&stats.RunningVMs)
		h.db.Model(&models.VMTemplate{}).Count(&stats.TotalTemplates)

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data":    stats,
		})
	}
}
