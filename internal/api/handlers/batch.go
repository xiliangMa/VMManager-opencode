package handlers

import (
	"net/http"

	"vmmanager/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BatchHandler struct {
	vmRepo *repository.VMRepository
}

func NewBatchHandler(vmRepo *repository.VMRepository) *BatchHandler {
	return &BatchHandler{vmRepo: vmRepo}
}

type BatchStartRequest struct {
	VMIDs []string `json:"vm_ids" binding:"required,min=1"`
}

type BatchStopRequest struct {
	VMIDs []string `json:"vm_ids" binding:"required,min=1"`
}

type BatchRestartRequest struct {
	VMIDs []string `json:"vm_ids" binding:"required,min=1"`
}

type BatchDeleteRequest struct {
	VMIDs []string `json:"vm_ids" binding:"required,min=1"`
}

type BatchResult struct {
	Success []string     `json:"success"`
	Failed  []FailedItem `json:"failed"`
}

type FailedItem struct {
	VMID   string `json:"vm_id"`
	Reason string `json:"reason"`
}

func (h *BatchHandler) BatchStart(c *gin.Context) {
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	var req BatchStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 4001, "message": err.Error()})
		return
	}

	result := BatchResult{
		Success: make([]string, 0),
		Failed:  make([]FailedItem, 0),
	}

	for _, vmID := range req.VMIDs {
		vm, err := h.vmRepo.FindByID(ctx, vmID)
		if err != nil {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Reason: "VM not found"})
			continue
		}

		if role != "admin" && vm.OwnerID != userUUID {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Reason: "permission denied"})
			continue
		}

		if err := h.vmRepo.UpdateStatus(ctx, vmID, "running"); err != nil {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Reason: err.Error()})
			continue
		}

		result.Success = append(result.Success, vmID)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

func (h *BatchHandler) BatchStop(c *gin.Context) {
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	var req BatchStopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 4001, "message": err.Error()})
		return
	}

	result := BatchResult{
		Success: make([]string, 0),
		Failed:  make([]FailedItem, 0),
	}

	for _, vmID := range req.VMIDs {
		vm, err := h.vmRepo.FindByID(ctx, vmID)
		if err != nil {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Reason: "VM not found"})
			continue
		}

		if role != "admin" && vm.OwnerID != userUUID {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Reason: "permission denied"})
			continue
		}

		if err := h.vmRepo.UpdateStatus(ctx, vmID, "stopped"); err != nil {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Reason: err.Error()})
			continue
		}

		result.Success = append(result.Success, vmID)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

func (h *BatchHandler) BatchRestart(c *gin.Context) {
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	var req BatchRestartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 4001, "message": err.Error()})
		return
	}

	result := BatchResult{
		Success: make([]string, 0),
		Failed:  make([]FailedItem, 0),
	}

	for _, vmID := range req.VMIDs {
		vm, err := h.vmRepo.FindByID(ctx, vmID)
		if err != nil {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Reason: "VM not found"})
			continue
		}

		if role != "admin" && vm.OwnerID != userUUID {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Reason: "permission denied"})
			continue
		}

		if err := h.vmRepo.UpdateStatus(ctx, vmID, "running"); err != nil {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Reason: err.Error()})
			continue
		}

		result.Success = append(result.Success, vmID)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

func (h *BatchHandler) BatchDelete(c *gin.Context) {
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	var req BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 4001, "message": err.Error()})
		return
	}

	result := BatchResult{
		Success: make([]string, 0),
		Failed:  make([]FailedItem, 0),
	}

	for _, vmID := range req.VMIDs {
		vm, err := h.vmRepo.FindByID(ctx, vmID)
		if err != nil {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Reason: "VM not found"})
			continue
		}

		if role != "admin" && vm.OwnerID != userUUID {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Reason: "permission denied"})
			continue
		}

		if err := h.vmRepo.Delete(ctx, vmID); err != nil {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Reason: err.Error()})
			continue
		}

		result.Success = append(result.Success, vmID)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}
