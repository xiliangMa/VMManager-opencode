package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"vmmanager/internal/api/errors"
	"vmmanager/internal/libvirt"
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
	libvirt      *libvirt.Client
}

func NewVMHandler(
	vmRepo *repository.VMRepository,
	userRepo *repository.UserRepository,
	templateRepo *repository.TemplateRepository,
	statsRepo *repository.VMStatsRepository,
	libvirtClient *libvirt.Client,
) *VMHandler {
	return &VMHandler{
		vmRepo:       vmRepo,
		userRepo:     userRepo,
		templateRepo: templateRepo,
		statsRepo:    statsRepo,
		libvirt:      libvirtClient,
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
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_fetch_vms"), err.Error()))
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
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, t(c, "vm_not_found_id"), id))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, t(c, "permission_denied_not_vm_owner"), "not VM owner"))
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
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
		return
	}

	ctx := c.Request.Context()

	user, err := h.userRepo.FindByID(ctx, userUUID.String())
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeUserNotFound, t(c, "user_not_found"), userUUID.String()))
		return
	}

	vmCount, _ := h.vmRepo.CountByOwner(ctx, userUUID.String())
	if user.QuotaVMCount > 0 && int(vmCount) >= user.QuotaVMCount {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeQuotaExceeded, t(c, "vm_quota_exceeded"), fmt.Sprintf("current: %d, limit: %d", vmCount, user.QuotaVMCount)))
		return
	}

	if req.CPUAllocated > user.QuotaCPU {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeQuotaExceeded, t(c, "cpu_quota_exceeded"), fmt.Sprintf("requested: %d, limit: %d", req.CPUAllocated, user.QuotaCPU)))
		return
	}

	if req.MemoryAllocated > user.QuotaMemory {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeQuotaExceeded, t(c, "memory_quota_exceeded"), fmt.Sprintf("requested: %d, limit: %d", req.MemoryAllocated, user.QuotaMemory)))
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
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_create_vm"), err.Error()))
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
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, t(c, "vm_not_found_id"), id))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, t(c, "permission_denied_not_vm_owner"), "not VM owner"))
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
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
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
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_update_vm"), err.Error()))
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
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, t(c, "vm_not_found_id"), id))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, t(c, "permission_denied_not_vm_owner"), "not VM owner"))
		return
	}

	if err := h.vmRepo.Delete(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "vm_deleted"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(nil))
}

func (h *VMHandler) StartVM(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, t(c, "vm_not_found_id"), id))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, t(c, "permission_denied_not_vm_owner"), "not VM owner"))
		return
	}

	if vm.Status == "running" {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "vm_already_running"), ""))
		return
	}

	log.Printf("[VM] Starting VM: %s, LibvirtDomainUUID: %s", id, vm.LibvirtDomainUUID)

	if h.libvirt == nil {
		log.Printf("[VM] libvirt client is nil, updating DB status only")
		if err := h.vmRepo.UpdateStatus(ctx, id, "running"); err != nil {
			c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_update_vm_status"), err.Error()))
			return
		}
		c.JSON(http.StatusOK, errors.Success(gin.H{
			"id":     vm.ID,
			"status": "running",
		}))
		return
	}

	if vm.LibvirtDomainUUID == "" {
		log.Printf("[VM] LibvirtDomainUUID is empty, creating domain in libvirt")

		domain, err := h.libvirt.DomainCreateXML(generateDomainXML(*vm), 0)
		if err != nil {
			log.Printf("[VM] Failed to create domain: %v", err)
			c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_create_vm_domain"), err.Error()))
			return
		}

		domainUUID, _ := domain.GetUUIDString()
		vm.LibvirtDomainUUID = domainUUID

		if err := h.vmRepo.UpdateLibvirtDomainUUID(ctx, id, domainUUID); err != nil {
			log.Printf("[VM] Failed to update LibvirtDomainUUID: %v", err)
		}

		log.Printf("[VM] Domain created: %s", domainUUID)
	}

	domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
	if err != nil {
		log.Printf("[VM] Failed to lookup domain: %v", err)
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_lookup_vm_domain"), err.Error()))
		return
	}

	state, _, _ := domain.GetState()
	log.Printf("[VM] Domain state before start: %d", state)

	if err := domain.Create(); err != nil {
		log.Printf("[VM] Failed to start domain: %v", err)
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_start_vm"), err.Error()))
		return
	}

	log.Printf("[VM] Domain started successfully")

	if err := h.vmRepo.UpdateStatus(ctx, id, "running"); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_update_vm_status"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"id":     vm.ID,
		"status": "running",
	}))
}

func generateDomainXML(vm models.VirtualMachine) string {
	return fmt.Sprintf(`<domain type='qemu'>
  <name>%s</name>
  <uuid>%s</uuid>
  <memory unit='MiB'>%d</memory>
  <vcpu placement='static'>%d</vcpu>
  <os>
    <type arch='x86_64' machine='pc'>hvm</type>
    <boot dev='hd'/>
  </os>
  <features>
    <acpi/>
    <apic/>
  </features>
  <cpu mode='host-model'>
    <model fallback='allow'/>
  </cpu>
  <clock offset='utc'>
    <timer name='rtc' tickpolicy='catchup'/>
    <timer name='hpet' present='no'/>
  </clock>
  <on_poweroff>destroy</on_poweroff>
  <on_reboot>restart</on_reboot>
  <on_crash>restart</on_crash>
  <devices>
    <emulator>/usr/bin/qemu-system-x86_64</emulator>
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source file='%s'/>
      <target dev='vda' bus='virtio'/>
    </disk>
    <interface type='network'>
      <source network='default'/>
      <model type='virtio'/>
    </interface>
    <graphics type='vnc' port='-1' autoport='yes' listen='0.0.0.0'/>
  </devices>
</domain>`, vm.Name, vm.ID.String(), vm.MemoryAllocated, vm.CPUAllocated, vm.DiskPath)
}

func (h *VMHandler) StopVM(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, t(c, "vm_not_found_id"), id))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, t(c, "permission_denied_not_vm_owner"), "not VM owner"))
		return
	}

	if vm.Status != "running" {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "vm_not_running"), ""))
		return
	}

	if h.libvirt != nil && vm.LibvirtDomainUUID != "" {
		domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
		if err == nil {
			if err := domain.Shutdown(); err != nil {
				c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_stop_vm"), err.Error()))
				return
			}
		}
	}

	if err := h.vmRepo.UpdateStatus(ctx, id, "stopped"); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_update_vm_status"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"id":     vm.ID,
		"status": "stopped",
	}))
}

func (h *VMHandler) ForceStopVM(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, t(c, "vm_not_found_id"), id))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, t(c, "permission_denied_not_vm_owner"), "not VM owner"))
		return
	}

	if h.libvirt != nil && vm.LibvirtDomainUUID != "" {
		domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
		if err == nil {
			if err := domain.Destroy(); err != nil {
				c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_force_stop_vm"), err.Error()))
				return
			}
		}
	}

	if err := h.vmRepo.UpdateStatus(ctx, id, "stopped"); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_update_vm_status"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"id":     vm.ID,
		"status": "stopped",
	}))
}

func (h *VMHandler) RebootVM(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, t(c, "vm_not_found_id"), id))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, t(c, "permission_denied_not_vm_owner"), "not VM owner"))
		return
	}

	if vm.Status != "running" {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "vm_not_running_reboot"), ""))
		return
	}

	if h.libvirt != nil && vm.LibvirtDomainUUID != "" {
		domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
		if err == nil {
			if err := domain.Shutdown(); err != nil {
				c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_reboot_vm"), err.Error()))
				return
			}
			if err := domain.Create(); err != nil {
				c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_reboot_vm"), err.Error()))
				return
			}
		}
	}

	if err := h.vmRepo.UpdateStatus(ctx, id, "running"); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_update_vm_status"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"id":     vm.ID,
		"status": "running",
	}))
}

func (h *VMHandler) SuspendVM(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, t(c, "vm_not_found_id"), id))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, t(c, "permission_denied_not_vm_owner"), "not VM owner"))
		return
	}

	if vm.Status != "running" {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "vm_not_running_suspend"), ""))
		return
	}

	if h.libvirt != nil && vm.LibvirtDomainUUID != "" {
		domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
		if err == nil {
			if err := domain.Suspend(); err != nil {
				c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_suspend_vm"), err.Error()))
				return
			}
		}
	}

	if err := h.vmRepo.UpdateStatus(ctx, id, "suspended"); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_update_vm_status"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"id":     vm.ID,
		"status": "suspended",
	}))
}

func (h *VMHandler) ResumeVM(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, t(c, "vm_not_found_id"), id))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, t(c, "permission_denied_not_vm_owner"), "not VM owner"))
		return
	}

	if vm.Status != "suspended" {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "vm_not_suspended"), ""))
		return
	}

	if h.libvirt != nil && vm.LibvirtDomainUUID != "" {
		domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
		if err == nil {
			if err := domain.Resume(); err != nil {
				c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_resume_vm"), err.Error()))
				return
			}
		}
	}

	if err := h.vmRepo.UpdateStatus(ctx, id, "running"); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_update_vm_status"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"id":     vm.ID,
		"status": "running",
	}))
}

func (h *VMHandler) updateVMStatus(id string, c *gin.Context, status string) {
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, t(c, "vm_not_found_id"), id))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, t(c, "permission_denied_not_vm_owner"), "not VM owner"))
		return
	}

	if err := h.vmRepo.UpdateStatus(ctx, id, status); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_update_vm_status"), err.Error()))
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
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, t(c, "vm_not_found_id"), id))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, t(c, "permission_denied_not_vm_owner"), "not VM owner"))
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
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "invalid_vm_id"), err.Error()))
		return
	}

	stats, err := h.statsRepo.FindByVMID(ctx, vmUUID.String(), 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_fetch_vm_stats"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(stats))
}
