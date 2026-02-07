package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VMHandler struct {
	db *gorm.DB
}

func NewVMHandler(db *gorm.DB) *VMHandler {
	return &VMHandler{db: db}
}

func (h *VMHandler) ListVMs(c *gin.Context) {
	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	var vms []models.VirtualMachine
	query := h.db.Preload("Owner").Preload("Template")

	if role != "admin" {
		query = query.Where("owner_id = ?", userUUID)
	}

	var status string
	if status = c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var search string
	if search = c.Query("search"); search != "" {
		query = query.Where("name ILIKE ?", "%"+search+"%")
	}

	var page, pageSize int
	page = 1
	pageSize = 20

	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps := c.Query("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	offset := (page - 1) * pageSize

	var total int64
	query.Model(&models.VirtualMachine{}).Count(&total)

	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&vms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "failed to fetch VMs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    vms,
		"meta": gin.H{
			"page":        page,
			"per_page":    pageSize,
			"total":       total,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

func (h *VMHandler) GetVM(c *gin.Context) {
	id := c.Param("id")
	vmUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 4001, "message": "invalid VM ID"})
		return
	}

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	var vm models.VirtualMachine
	query := h.db.Preload("Owner").Preload("Template")

	if role != "admin" {
		query = query.Where("owner_id = ?", userUUID)
	}

	if err := query.First(&vm, "id = ?", vmUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "VM not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    vm,
	})
}

func (h *VMHandler) CreateVM(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userUUID, _ := uuid.Parse(userID.(string))

	var req struct {
		Name            string   `json:"name" binding:"required"`
		Description     string   `json:"description"`
		TemplateID      *string  `json:"template_id"`
		CPUAllocated    int      `json:"cpu_allocated" binding:"required,min=1"`
		MemoryAllocated int      `json:"memory_allocated" binding:"required,min=512"`
		DiskAllocated   int      `json:"disk_allocated" binding:"required,min=10"`
		BootOrder       string   `json:"boot_order"`
		Autostart       bool     `json:"autostart"`
		Tags            []string `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 4001, "message": err.Error()})
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", userUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "user not found"})
		return
	}

	if user.QuotaVMCount > 0 {
		var vmCount int64
		h.db.Model(&models.VirtualMachine{}).Where("owner_id = ? AND deleted_at IS NULL", userUUID).Count(&vmCount)
		if int(vmCount) >= user.QuotaVMCount {
			c.JSON(http.StatusForbidden, gin.H{"code": 4006, "message": "VM quota exceeded"})
			return
		}
	}

	if req.CPUAllocated > user.QuotaCPU {
		c.JSON(http.StatusForbidden, gin.H{"code": 4006, "message": "CPU quota exceeded"})
		return
	}

	if req.MemoryAllocated > user.QuotaMemory {
		c.JSON(http.StatusForbidden, gin.H{"code": 4006, "message": "memory quota exceeded"})
		return
	}

	macAddress, _ := models.GenerateMACAddress()
	vncPassword, _ := models.GenerateVNCPassword(12)

	vm := models.VirtualMachine{
		ID:              uuid.New(),
		Name:            req.Name,
		Description:     req.Description,
		OwnerID:         userUUID,
		Status:          "pending",
		MACAddress:      macAddress,
		VNCPassword:     vncPassword,
		CPUAllocated:    req.CPUAllocated,
		MemoryAllocated: req.MemoryAllocated,
		DiskAllocated:   req.DiskAllocated,
		BootOrder:       req.BootOrder,
		Autostart:       req.Autostart,
		Tags:            req.Tags,
	}

	if req.TemplateID != nil {
		templateUUID, _ := uuid.Parse(*req.TemplateID)
		vm.TemplateID = &templateUUID
	}

	if err := h.db.Create(&vm).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "failed to create VM"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "success",
		"data":    vm,
	})
}

func (h *VMHandler) UpdateVM(c *gin.Context) {
	id := c.Param("id")
	vmUUID, _ := uuid.Parse(id)

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	var vm models.VirtualMachine
	query := h.db

	if role != "admin" {
		query = query.Where("owner_id = ?", userUUID)
	}

	if err := query.First(&vm, "id = ?", vmUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "VM not found"})
		return
	}

	var req struct {
		Name      string   `json:"name"`
		BootOrder string   `json:"boot_order"`
		Autostart bool     `json:"autostart"`
		Notes     string   `json:"notes"`
		Tags      []string `json:"tags"`
	}

	c.ShouldBindJSON(&req)

	if req.Name != "" {
		vm.Name = req.Name
	}
	if req.BootOrder != "" {
		vm.BootOrder = req.BootOrder
	}
	vm.Autostart = req.Autostart
	if req.Notes != "" {
		vm.Notes = req.Notes
	}
	if req.Tags != nil {
		vm.Tags = req.Tags
	}

	h.db.Save(&vm)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    vm,
	})
}

func (h *VMHandler) DeleteVM(c *gin.Context) {
	id := c.Param("id")
	vmUUID, _ := uuid.Parse(id)

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	var vm models.VirtualMachine
	query := h.db

	if role != "admin" {
		query = query.Where("owner_id = ?", userUUID)
	}

	if err := query.First(&vm, "id = ?", vmUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "VM not found"})
		return
	}

	now := time.Now()
	vm.DeletedAt = &now
	vm.Status = "deleted"

	h.db.Save(&vm)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
}

func (h *VMHandler) StartVM(c *gin.Context) {
	h.updateVMStatus(c.Param("id"), c, "running")
}

func (h *VMHandler) StopVM(c *gin.Context) {
	h.updateVMStatus(c.Param("id"), c, "stopping")
}

func (h *VMHandler) ForceStopVM(c *gin.Context) {
	h.updateVMStatus(c.Param("id"), c, "stopped")
}

func (h *VMHandler) RebootVM(c *gin.Context) {
	h.updateVMStatus(c.Param("id"), c, "running")
}

func (h *VMHandler) SuspendVM(c *gin.Context) {
	h.updateVMStatus(c.Param("id"), c, "suspended")
}

func (h *VMHandler) ResumeVM(c *gin.Context) {
	h.updateVMStatus(c.Param("id"), c, "running")
}

func (h *VMHandler) updateVMStatus(id string, c *gin.Context, status string) {
	vmUUID, _ := uuid.Parse(id)

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	var vm models.VirtualMachine
	query := h.db

	if role != "admin" {
		query = query.Where("owner_id = ?", userUUID)
	}

	if err := query.First(&vm, "id = ?", vmUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "VM not found"})
		return
	}

	vm.Status = status
	h.db.Save(&vm)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"id":     vm.ID,
			"status": vm.Status,
		},
	})
}

func (h *VMHandler) GetConsole(c *gin.Context) {
	id := c.Param("id")
	vmUUID, _ := uuid.Parse(id)

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	var vm models.VirtualMachine
	query := h.db

	if role != "admin" {
		query = query.Where("owner_id = ?", userUUID)
	}

	if err := query.First(&vm, "id = ?", vmUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "VM not found"})
		return
	}

	if vm.VNCPassword == "" {
		vm.VNCPassword, _ = models.GenerateVNCPassword(12)
		h.db.Save(&vm)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"type":          "vnc",
			"host":          c.Request.Host,
			"port":          vm.VNCPort,
			"password":      vm.VNCPassword,
			"websocket_url": fmt.Sprintf("ws://%s/ws/vnc/%s", c.Request.Host, vm.ID),
			"expires_at":    time.Now().Add(30 * time.Minute).Format(time.RFC3339),
		},
	})
}

func (h *VMHandler) GetVMStats(c *gin.Context) {
	id := c.Param("id")
	vmUUID, _ := uuid.Parse(id)

	var stats []models.VMStats
	h.db.Where("vm_id = ?", vmUUID).Order("collected_at DESC").Limit(100).Find(&stats)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    stats,
	})
}
