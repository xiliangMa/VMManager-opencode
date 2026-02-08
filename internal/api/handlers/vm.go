package handlers

import (
	"fmt"
	"net/http"
	"time"

	"vmmanager/internal/api/errors"
	"vmmanager/internal/models"
	"vmmanager/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type VMHandler struct {
	vmRepo       *repository.VMRepository
	userRepo     *repository.UserRepository
	templateRepo *repository.TemplateRepository
	statsRepo    *repository.VMStatsRepository
}

func NewVMHandler(
	vmRepo *repository.VMRepository,
	userRepo *repository.UserRepository,
	templateRepo *repository.TemplateRepository,
	statsRepo *repository.VMStatsRepository,
) *VMHandler {
	return &VMHandler{
		vmRepo:       vmRepo,
		userRepo:     userRepo,
		templateRepo: templateRepo,
		statsRepo:    statsRepo,
	}
}

func (h *VMHandler) ListVMs(c *gin.Context) {
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	page := 1
	pageSize := 20

	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if ps := c.Query("page_size"); ps != "" {
		fmt.Sscanf(ps, "%d", &pageSize)
	}

	var vms []models.VirtualMachine
	var total int64
	var err error

	if role != "admin" {
		vms, total, err = h.vmRepo.FindByOwner(ctx, userUUID.String(), (page-1)*pageSize, pageSize)
	} else {
		vms, total, err = h.vmRepo.List(ctx, (page-1)*pageSize, pageSize)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to fetch VMs", err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.SuccessWithMeta(vms, gin.H{
		"page":        page,
		"per_page":    pageSize,
		"total":       total,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	}))
}

func (h *VMHandler) GetVM(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, "VM not found", id))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, "permission denied", "not VM owner"))
		return
	}

	c.JSON(http.StatusOK, errors.Success(vm))
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
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, "validation error", err.Error()))
		return
	}

	ctx := c.Request.Context()

	user, err := h.userRepo.FindByID(ctx, userUUID.String())
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeUserNotFound, "user not found", userUUID.String()))
		return
	}

	vmCount, _ := h.vmRepo.CountByOwner(ctx, userUUID.String())
	if user.QuotaVMCount > 0 && int(vmCount) >= user.QuotaVMCount {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeQuotaExceeded, "VM quota exceeded", fmt.Sprintf("current: %d, limit: %d", vmCount, user.QuotaVMCount)))
		return
	}

	if req.CPUAllocated > user.QuotaCPU {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeQuotaExceeded, "CPU quota exceeded", fmt.Sprintf("requested: %d, limit: %d", req.CPUAllocated, user.QuotaCPU)))
		return
	}

	if req.MemoryAllocated > user.QuotaMemory {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeQuotaExceeded, "memory quota exceeded", fmt.Sprintf("requested: %d, limit: %d", req.MemoryAllocated, user.QuotaMemory)))
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

	if err := h.vmRepo.Create(ctx, &vm); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to create VM", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, errors.Success(vm))
}

func (h *VMHandler) UpdateVM(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, "VM not found", id))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, "permission denied", "not VM owner"))
		return
	}

	var req struct {
		Name      string   `json:"name"`
		BootOrder string   `json:"boot_order"`
		Autostart bool     `json:"autostart"`
		Notes     string   `json:"notes"`
		Tags      []string `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, "validation error", err.Error()))
		return
	}

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

	if err := h.vmRepo.Update(ctx, vm); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to update VM", err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(vm))
}

func (h *VMHandler) DeleteVM(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, "VM not found", id))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, "permission denied", "not VM owner"))
		return
	}

	if err := h.vmRepo.Delete(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to delete VM", err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(nil))
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
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, "VM not found", id))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, "permission denied", "not VM owner"))
		return
	}

	if err := h.vmRepo.UpdateStatus(ctx, id, status); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to update VM status", err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"id":     vm.ID,
		"status": status,
	}))
}

func (h *VMHandler) GetConsole(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, "VM not found", id))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, "permission denied", "not VM owner"))
		return
	}

	if vm.VNCPassword == "" {
		vm.VNCPassword, _ = models.GenerateVNCPassword(12)
		h.vmRepo.Update(ctx, vm)
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"type":          "vnc",
		"host":          c.Request.Host,
		"port":          vm.VNCPort,
		"password":      vm.VNCPassword,
		"websocket_url": fmt.Sprintf("ws://%s/ws/vnc/%s", c.Request.Host, vm.ID),
		"expires_at":    time.Now().Add(30 * time.Minute).Format(time.RFC3339),
	}))
}

func (h *VMHandler) GetVMStats(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	vmUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, "invalid VM ID", err.Error()))
		return
	}

	stats, err := h.statsRepo.FindByVMID(ctx, vmUUID.String(), 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to fetch VM stats", err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(stats))
}
