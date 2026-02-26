package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"vmmanager/internal/api/errors"
	"vmmanager/internal/libvirt"
	"vmmanager/internal/models"
	"vmmanager/internal/repository"
	"vmmanager/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BatchHandler struct {
	vmRepo       *repository.VMRepository
	libvirt      *libvirt.Client
	storagePath  string
	auditService *services.AuditService
}

func NewBatchHandler(vmRepo *repository.VMRepository, libvirt *libvirt.Client, storagePath string, auditService *services.AuditService) *BatchHandler {
	return &BatchHandler{
		vmRepo:       vmRepo,
		libvirt:      libvirt,
		storagePath:  storagePath,
		auditService: auditService,
	}
}

type BatchStartRequest struct {
	VMIDs []string `json:"vm_ids" binding:"required,min=1"`
}

type BatchStopRequest struct {
	VMIDs []string `json:"vm_ids" binding:"required,min=1"`
	Force bool     `json:"force"`
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
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

func (h *BatchHandler) BatchStart(c *gin.Context) {
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	var req BatchStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
		return
	}

	result := BatchResult{
		Success: make([]string, 0),
		Failed:  make([]FailedItem, 0),
	}

	for _, vmID := range req.VMIDs {
		vm, err := h.vmRepo.FindByID(ctx, vmID)
		if err != nil {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Reason: t(c, "vm_not_found")})
			continue
		}

		if role != "admin" && vm.OwnerID != userUUID {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Name: vm.Name, Reason: t(c, "permission_denied")})
			continue
		}

		if vm.Status == "running" {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Name: vm.Name, Reason: t(c, "vm_already_running")})
			continue
		}

		if h.libvirt == nil {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Name: vm.Name, Reason: t(c, "libvirt_service_unavailable")})
			continue
		}

		started := false
		if vm.LibvirtDomainUUID != "" && vm.LibvirtDomainUUID != "new-uuid" && vm.LibvirtDomainUUID != "defined-uuid" {
			domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
			if err == nil {
				if err := domain.Create(); err != nil {
					log.Printf("[BATCH] Failed to start VM %s: %v", vmID, err)
					domain.Free()
					result.Failed = append(result.Failed, FailedItem{VMID: vmID, Name: vm.Name, Reason: err.Error()})
					continue
				}
				domain.Free()
				started = true
			}
		}

		if !started {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Name: vm.Name, Reason: t(c, "failed_to_start_vm")})
			continue
		}

		if err := h.vmRepo.UpdateStatus(ctx, vmID, "running"); err != nil {
			log.Printf("[BATCH] Failed to update VM status %s: %v", vmID, err)
		}

		result.Success = append(result.Success, vmID)
	}

	if h.auditService != nil {
		h.auditService.LogSuccess(c, "vm.batch_start", "virtual_machine", nil, map[string]interface{}{
			"vm_ids":  req.VMIDs,
			"success": len(result.Success),
			"failed":  len(result.Failed),
		})
	}

	c.JSON(http.StatusOK, errors.Success(result))
}

func (h *BatchHandler) BatchStop(c *gin.Context) {
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	var req BatchStopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
		return
	}

	result := BatchResult{
		Success: make([]string, 0),
		Failed:  make([]FailedItem, 0),
	}

	for _, vmID := range req.VMIDs {
		vm, err := h.vmRepo.FindByID(ctx, vmID)
		if err != nil {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Reason: t(c, "vm_not_found")})
			continue
		}

		if role != "admin" && vm.OwnerID != userUUID {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Name: vm.Name, Reason: t(c, "permission_denied")})
			continue
		}

		if vm.Status != "running" {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Name: vm.Name, Reason: t(c, "vm_not_running")})
			continue
		}

		if h.libvirt == nil {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Name: vm.Name, Reason: t(c, "libvirt_service_unavailable")})
			continue
		}

		stopped := false
		if vm.LibvirtDomainUUID != "" && vm.LibvirtDomainUUID != "new-uuid" && vm.LibvirtDomainUUID != "defined-uuid" {
			domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
			if err == nil {
				if req.Force {
					if err := domain.Destroy(); err != nil {
						log.Printf("[BATCH] Failed to force stop VM %s: %v", vmID, err)
						domain.Free()
						result.Failed = append(result.Failed, FailedItem{VMID: vmID, Name: vm.Name, Reason: err.Error()})
						continue
					}
				} else {
					if err := domain.Shutdown(); err != nil {
						log.Printf("[BATCH] Failed to stop VM %s: %v", vmID, err)
						domain.Free()
						result.Failed = append(result.Failed, FailedItem{VMID: vmID, Name: vm.Name, Reason: err.Error()})
						continue
					}
				}
				domain.Free()
				stopped = true
			}
		}

		if !stopped {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Name: vm.Name, Reason: t(c, "failed_to_stop_vm")})
			continue
		}

		if err := h.vmRepo.UpdateStatus(ctx, vmID, "stopped"); err != nil {
			log.Printf("[BATCH] Failed to update VM status %s: %v", vmID, err)
		}

		result.Success = append(result.Success, vmID)
	}

	if h.auditService != nil {
		action := "vm.batch_stop"
		if req.Force {
			action = "vm.batch_force_stop"
		}
		h.auditService.LogSuccess(c, action, "virtual_machine", nil, map[string]interface{}{
			"vm_ids":  req.VMIDs,
			"force":   req.Force,
			"success": len(result.Success),
			"failed":  len(result.Failed),
		})
	}

	c.JSON(http.StatusOK, errors.Success(result))
}

func (h *BatchHandler) BatchDelete(c *gin.Context) {
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	var req BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
		return
	}

	result := BatchResult{
		Success: make([]string, 0),
		Failed:  make([]FailedItem, 0),
	}

	for _, vmID := range req.VMIDs {
		vm, err := h.vmRepo.FindByID(ctx, vmID)
		if err != nil {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Reason: t(c, "vm_not_found")})
			continue
		}

		if role != "admin" && vm.OwnerID != userUUID {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Name: vm.Name, Reason: t(c, "permission_denied")})
			continue
		}

		if vm.Status == "running" || vm.Status == "paused" {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Name: vm.Name, Reason: t(c, "vm_running_delete")})
			continue
		}

		if h.libvirt != nil && vm.LibvirtDomainUUID != "" && vm.LibvirtDomainUUID != "new-uuid" && vm.LibvirtDomainUUID != "defined-uuid" {
			domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
			if err == nil {
				state, _, _ := domain.GetState()
				if state == 1 {
					domain.Destroy()
				}
				domain.Free()

				cmd := exec.Command("virsh", "undefine", "--nvram", "--uuid", vm.LibvirtDomainUUID)
				if output, err := cmd.CombinedOutput(); err != nil {
					log.Printf("[BATCH] Failed to undefine domain: %v, output: %s", err, string(output))
					if strings.Contains(string(output), "cannot undefine domain with nvram") {
						cmd = exec.Command("virsh", "undefine", "--uuid", vm.LibvirtDomainUUID)
						cmd.CombinedOutput()
					}
				}
			}
		}

		if vm.DiskPath != "" {
			if _, err := os.Stat(vm.DiskPath); err == nil {
				if err := os.Remove(vm.DiskPath); err != nil {
					log.Printf("[BATCH] Failed to delete disk file %s: %v", vm.DiskPath, err)
				}
			}
		}

		vmUUID, _ := uuid.Parse(vmID)
		if err := h.vmRepo.Delete(ctx, vmID); err != nil {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Name: vm.Name, Reason: err.Error()})
			continue
		}

		if h.auditService != nil {
			h.auditService.LogSuccess(c, "vm.delete", "virtual_machine", &vmUUID, map[string]interface{}{
				"name":   vm.Name,
				"status": vm.Status,
			})
		}

		result.Success = append(result.Success, vmID)
	}

	if h.auditService != nil {
		h.auditService.LogSuccess(c, "vm.batch_delete", "virtual_machine", nil, map[string]interface{}{
			"vm_ids":  req.VMIDs,
			"success": len(result.Success),
			"failed":  len(result.Failed),
		})
	}

	c.JSON(http.StatusOK, errors.Success(result))
}

func (h *BatchHandler) BatchOperation(c *gin.Context) {
	operation := c.Param("operation")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	var req struct {
		VMIDs []string `json:"vm_ids" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
		return
	}

	result := BatchResult{
		Success: make([]string, 0),
		Failed:  make([]FailedItem, 0),
	}

	for _, vmID := range req.VMIDs {
		vm, err := h.vmRepo.FindByID(ctx, vmID)
		if err != nil {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Reason: t(c, "vm_not_found")})
			continue
		}

		if role != "admin" && vm.OwnerID != userUUID {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Name: vm.Name, Reason: t(c, "permission_denied")})
			continue
		}

		var opErr error
		switch operation {
		case "start":
			opErr = h.performStart(vm)
		case "stop":
			opErr = h.performStop(vm, false)
		case "force-stop":
			opErr = h.performStop(vm, true)
		case "suspend":
			opErr = h.performSuspend(vm)
		case "resume":
			opErr = h.performResume(vm)
		default:
			opErr = fmt.Errorf("unknown operation: %s", operation)
		}

		if opErr != nil {
			result.Failed = append(result.Failed, FailedItem{VMID: vmID, Name: vm.Name, Reason: opErr.Error()})
			continue
		}

		newStatus := h.getNewStatus(operation)
		if newStatus != "" {
			h.vmRepo.UpdateStatus(ctx, vmID, newStatus)
		}

		result.Success = append(result.Success, vmID)
	}

	if h.auditService != nil {
		h.auditService.LogSuccess(c, fmt.Sprintf("vm.batch_%s", operation), "virtual_machine", nil, map[string]interface{}{
			"vm_ids":    req.VMIDs,
			"operation": operation,
			"success":   len(result.Success),
			"failed":    len(result.Failed),
		})
	}

	c.JSON(http.StatusOK, errors.Success(result))
}

func (h *BatchHandler) performStart(vm *models.VirtualMachine) error {
	if h.libvirt == nil {
		return fmt.Errorf("libvirt service unavailable")
	}

	if vm.LibvirtDomainUUID == "" || vm.LibvirtDomainUUID == "new-uuid" {
		return fmt.Errorf("domain not defined")
	}

	domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
	if err != nil {
		return err
	}
	defer domain.Free()

	return domain.Create()
}

func (h *BatchHandler) performStop(vm *models.VirtualMachine, force bool) error {
	if h.libvirt == nil {
		return fmt.Errorf("libvirt service unavailable")
	}

	if vm.LibvirtDomainUUID == "" || vm.LibvirtDomainUUID == "new-uuid" {
		return fmt.Errorf("domain not defined")
	}

	domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
	if err != nil {
		return err
	}
	defer domain.Free()

	if force {
		return domain.Destroy()
	}
	return domain.Shutdown()
}

func (h *BatchHandler) performSuspend(vm *models.VirtualMachine) error {
	if h.libvirt == nil {
		return fmt.Errorf("libvirt service unavailable")
	}

	if vm.LibvirtDomainUUID == "" || vm.LibvirtDomainUUID == "new-uuid" {
		return fmt.Errorf("domain not defined")
	}

	domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
	if err != nil {
		return err
	}
	defer domain.Free()

	return domain.Suspend()
}

func (h *BatchHandler) performResume(vm *models.VirtualMachine) error {
	if h.libvirt == nil {
		return fmt.Errorf("libvirt service unavailable")
	}

	if vm.LibvirtDomainUUID == "" || vm.LibvirtDomainUUID == "new-uuid" {
		return fmt.Errorf("domain not defined")
	}

	domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
	if err != nil {
		return err
	}
	defer domain.Free()

	return domain.Resume()
}

func (h *BatchHandler) getNewStatus(operation string) string {
	switch operation {
	case "start", "resume":
		return "running"
	case "stop", "force-stop":
		return "stopped"
	case "suspend":
		return "suspended"
	default:
		return ""
	}
}
