package handlers

import (
	"net/http"

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
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "VM not found"})
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"code": 4003, "message": "permission denied"})
		return
	}

	var req CreateSnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 4001, "message": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"vm_id": vm.ID,
			"name":  req.Name,
			"state": vm.Status,
		},
	})
}

func (h *SnapshotHandler) ListSnapshots(c *gin.Context) {
	vmID := c.Param("id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, vmID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "VM not found"})
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"code": 4003, "message": "permission denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    []gin.H{},
	})
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
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "VM not found"})
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"code": 4003, "message": "permission denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"name":  snapshotName,
			"state": vm.Status,
		},
	})
}

func (h *SnapshotHandler) RestoreSnapshot(c *gin.Context) {
	vmID := c.Param("id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, vmID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "VM not found"})
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"code": 4003, "message": "permission denied"})
		return
	}

	var req RestoreSnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 4001, "message": err.Error()})
		return
	}

	if err := h.vmRepo.UpdateStatus(ctx, vmID, "running"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "failed to restore snapshot"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"vm_id":    vm.ID,
			"snapshot": req.Name,
		},
	})
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
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "VM not found"})
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"code": 4003, "message": "permission denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"vm_id":    vm.ID,
			"snapshot": snapshotName,
		},
	})
}
