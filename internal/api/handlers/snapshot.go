package handlers

import (
	"net/http"

	"vmmanager/internal/api/errors"
	"vmmanager/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SnapshotHandler struct {
	vmRepo *repository.VMRepository
}

func NewSnapshotHandler(vmRepo *repository.VMRepository) *SnapshotHandler {
	return &SnapshotHandler{vmRepo: vmRepo}
}

type CreateSnapshotRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type RestoreSnapshotRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *SnapshotHandler) CreateSnapshot(c *gin.Context) {
	vmID := c.Param("id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, vmID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, "VM not found", vmID))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, "permission denied", "not VM owner"))
		return
	}

	var req CreateSnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, "validation error", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, errors.Success(gin.H{
		"vm_id": vm.ID,
		"name":  req.Name,
		"state": vm.Status,
	}))
}

func (h *SnapshotHandler) ListSnapshots(c *gin.Context) {
	vmID := c.Param("id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, vmID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, "VM not found", vmID))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, "permission denied", "not VM owner"))
		return
	}

	c.JSON(http.StatusOK, errors.Success([]gin.H{}))
}

func (h *SnapshotHandler) GetSnapshot(c *gin.Context) {
	vmID := c.Param("id")
	snapshotName := c.Param("name")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, vmID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, "VM not found", vmID))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, "permission denied", "not VM owner"))
		return
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"name":  snapshotName,
		"state": vm.Status,
	}))
}

func (h *SnapshotHandler) RestoreSnapshot(c *gin.Context) {
	vmID := c.Param("id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, vmID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, "VM not found", vmID))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, "permission denied", "not VM owner"))
		return
	}

	var req RestoreSnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, "validation error", err.Error()))
		return
	}

	if err := h.vmRepo.UpdateStatus(ctx, vmID, "running"); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to restore snapshot", err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"vm_id":    vm.ID,
		"snapshot": req.Name,
	}))
}

func (h *SnapshotHandler) DeleteSnapshot(c *gin.Context) {
	vmID := c.Param("id")
	snapshotName := c.Param("name")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, vmID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, "VM not found", vmID))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, "permission denied", "not VM owner"))
		return
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"vm_id":    vm.ID,
		"snapshot": snapshotName,
	}))
}
