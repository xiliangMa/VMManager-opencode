package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
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
	isoRepo      *repository.ISORepository
	libvirt      *libvirt.Client
	storagePath  string
}

func NewVMHandler(
	vmRepo *repository.VMRepository,
	userRepo *repository.UserRepository,
	templateRepo *repository.TemplateRepository,
	statsRepo *repository.VMStatsRepository,
	isoRepo *repository.ISORepository,
	libvirtClient *libvirt.Client,
	storagePath string,
) *VMHandler {
	return &VMHandler{
		vmRepo:       vmRepo,
		userRepo:     userRepo,
		templateRepo: templateRepo,
		statsRepo:    statsRepo,
		isoRepo:      isoRepo,
		libvirt:      libvirtClient,
		storagePath:  storagePath,
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

	if h.libvirt != nil {
		existingDomain, err := h.libvirt.LookupByName(req.Name)
		if err == nil {
			existingDomain.Free()
			c.JSON(http.StatusConflict, errors.FailWithDetails(errors.ErrCodeVMConflict, t(c, "vm_name_exists"), fmt.Sprintf("VM with name '%s' already exists in libvirt", req.Name)))
			return
		}
	}

	macAddress, _ := models.GenerateMACAddress()
	vncPassword, _ := models.GenerateVNCPassword(8)

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
		DiskPath:        fmt.Sprintf("%s/%s.qcow2", h.storagePath, uuid.New().String()),
		BootOrder:       req.BootOrder,
		Autostart:       req.Autostart,
		Tags:            req.Tags,
	}

	if req.TemplateID != nil {
		templateUUID, _ := uuid.Parse(*req.TemplateID)
		vm.TemplateID = &templateUUID

		template, err := h.templateRepo.FindByID(ctx, *req.TemplateID)
		if err == nil && template != nil && template.Architecture != "" {
			vm.Architecture = template.Architecture
		}
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

	if vm.Status == "running" || vm.Status == "paused" || vm.Status == "suspended" {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "vm_running_delete"), ""))
		return
	}

	deleted := false
	tryDelete := func() bool {
		domainDeleted := false

		if vm.LibvirtDomainUUID != "" && vm.LibvirtDomainUUID != "new-uuid" && vm.LibvirtDomainUUID != "defined-uuid" {
			log.Printf("[VM] Deleting libvirt domain by UUID: %s", vm.LibvirtDomainUUID)
			domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
			if err == nil {
				state, _, _ := domain.GetState()
				if state == 1 {
					log.Printf("[VM] Force destroying running VM: %s", id)
					domain.Destroy()
				}
				domain.Free()

				cmd := exec.Command("virsh", "undefine", "--nvram", vm.LibvirtDomainUUID)
				if output, err := cmd.CombinedOutput(); err != nil {
					log.Printf("[VM] Failed to undefine domain by UUID: %v, output: %s", err, string(output))
					if strings.Contains(string(output), "cannot undefine domain with nvram") {
						cmd = exec.Command("virsh", "undefine", vm.LibvirtDomainUUID)
						if output, err := cmd.CombinedOutput(); err != nil {
							log.Printf("[VM] Failed to undefine domain by UUID (fallback): %v, output: %s", err, string(output))
						} else {
							log.Printf("[VM] Libvirt domain undefined by UUID: %s", vm.LibvirtDomainUUID)
							domainDeleted = true
						}
					}
				} else {
					log.Printf("[VM] Libvirt domain undefined by UUID: %s", vm.LibvirtDomainUUID)
					domainDeleted = true
				}
			} else {
				log.Printf("[VM] Domain not found by UUID: %v", err)
			}
		}

		if !domainDeleted && vm.Name != "" {
			log.Printf("[VM] Deleting libvirt domain by name: %s", vm.Name)
			domain, err := h.libvirt.LookupByName(vm.Name)
			if err == nil {
				state, _, _ := domain.GetState()
				if state == 1 {
					log.Printf("[VM] Force destroying running VM by name: %s", vm.Name)
					domain.Destroy()
				}
				domain.Free()

				cmd := exec.Command("virsh", "undefine", "--nvram", vm.Name)
				if output, err := cmd.CombinedOutput(); err != nil {
					log.Printf("[VM] Failed to undefine domain by name: %v, output: %s", err, string(output))
					if strings.Contains(string(output), "cannot undefine domain with nvram") {
						cmd = exec.Command("virsh", "undefine", vm.Name)
						if output, err := cmd.CombinedOutput(); err != nil {
							log.Printf("[VM] Failed to undefine domain by name (fallback): %v, output: %s", err, string(output))
						} else {
							log.Printf("[VM] Libvirt domain undefined by name: %s", vm.Name)
							domainDeleted = true
						}
					}
				} else {
					log.Printf("[VM] Libvirt domain undefined by name: %s", vm.Name)
					domainDeleted = true
				}
			} else {
				log.Printf("[VM] Domain not found by name: %v", err)
			}
		}
		return domainDeleted
	}

	if vm.Name != "" {
		nvramPath := fmt.Sprintf("/var/lib/libvirt/qemu/nvram/%s_VARS.fd", vm.Name)
		if exists(nvramPath) {
			log.Printf("[VM] Deleting nvram file before undefine: %s", nvramPath)
			if err := os.Remove(nvramPath); err != nil {
				log.Printf("[VM] Failed to delete nvram file: %v", err)
			} else {
				log.Printf("[VM] Nvram file deleted: %s", nvramPath)
			}
		}
	}

	if h.libvirt != nil {
		deleted = tryDelete()
		if !deleted {
			log.Printf("[VM] Warning: Failed to delete libvirt domain for VM: %s (uuid: %s, name: %s)", id, vm.LibvirtDomainUUID, vm.Name)
			c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, "failed_to_delete_vm", "Failed to delete libvirt domain"))
			return
		}
	} else {
		log.Printf("[VM] Error: libvirt client is nil, cannot delete domain for VM: %s", id)
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, "failed_to_delete_vm", "libvirt client is not initialized"))
		return
	}

	if vm.DiskPath != "" && exists(vm.DiskPath) {
		log.Printf("[VM] Deleting disk file: %s", vm.DiskPath)
		if err := os.Remove(vm.DiskPath); err != nil {
			log.Printf("[VM] Failed to delete disk file: %v", err)
		} else {
			log.Printf("[VM] Disk file deleted: %s", vm.DiskPath)
		}
	}

	if err := h.vmRepo.Delete(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "vm_deleted"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(nil))
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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
		log.Printf("[VM] libvirt client is nil, cannot start VM")
		c.JSON(http.StatusServiceUnavailable, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "libvirt_service_unavailable"), "libvirt client is not initialized"))
		return
	}

	domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
	if err != nil {
		log.Printf("[VM] Domain not found in libvirt: %v", err)
		log.Printf("[VM] LibvirtDomainUUID: %s, creating new domain", vm.LibvirtDomainUUID)

		diskPath := vm.DiskPath
		if diskPath == "" {
			diskPath = fmt.Sprintf("%s/%s.qcow2", h.storagePath, vm.ID.String())
		}

		templatePath := ""
		isoPath := ""
		if vm.TemplateID != nil {
			template, err := h.templateRepo.FindByID(ctx, vm.TemplateID.String())
			if err == nil && template.TemplatePath != "" {
				templatePath = template.TemplatePath
				if !strings.HasPrefix(templatePath, "/") {
					templatePath = "./" + templatePath
				}
				log.Printf("[VM] Template path: %s", templatePath)

				if strings.HasSuffix(strings.ToLower(templatePath), ".iso") {
					isoPath = templatePath
					log.Printf("[VM] Detected ISO file: %s", isoPath)
					templatePath = ""
				}
			}
		}

		if templatePath != "" && exists(templatePath) {
			log.Printf("[VM] Copying template disk to: %s", diskPath)
			cmd := exec.Command("cp", templatePath, diskPath)
			if err := cmd.Run(); err != nil {
				log.Printf("[VM] Failed to copy template disk: %v, creating empty disk", err)
				cmd = exec.Command("qemu-img", "create", "-f", "qcow2", "-o", "preallocation=off", diskPath, fmt.Sprintf("%dG", vm.DiskAllocated))
				if err := cmd.Run(); err != nil {
					log.Printf("[VM] Failed to create disk image: %v", err)
				}
			}
		} else {
			log.Printf("[VM] Creating empty disk image: %s", diskPath)
			cmd := exec.Command("qemu-img", "create", "-f", "qcow2", "-o", "preallocation=off", diskPath, fmt.Sprintf("%dG", vm.DiskAllocated))
			if err := cmd.Run(); err != nil {
				log.Printf("[VM] Failed to create disk image: %v", err)
			}
		}

		log.Printf("[VM] Disk prepared: %s", diskPath)

		nvramPath := fmt.Sprintf("/var/lib/libvirt/qemu/nvram/%s_VARS.fd", vm.Name)
		arch := vm.Architecture
		if arch == "" {
			arch = "x86_64"
		}
		var nvramTemplate string
		switch arch {
		case "arm64", "aarch64":
			nvramTemplate = "/usr/share/AAVMF/AAVMF_VARS.fd"
		default:
			nvramTemplate = "/usr/share/OVMF/OVMF_VARS.fd"
		}

		if exists(nvramTemplate) {
			cmd := exec.Command("cp", nvramTemplate, nvramPath)
			if err := cmd.Run(); err != nil {
				log.Printf("[VM] Failed to create nvram file: %v", err)
			} else {
				log.Printf("[VM] NVRAM file created: %s", nvramPath)
			}
		} else {
			log.Printf("[VM] NVRAM template not found: %s", nvramTemplate)
		}

		domainXML := generateDomainXML(*vm, diskPath, isoPath)
		log.Printf("[VM] Generated domain XML:\n%s", domainXML)

		domain, err = h.libvirt.DefineXML(domainXML)
		if err != nil {
			log.Printf("[VM] Failed to define domain: %v", err)
			c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_create_vm_domain"), err.Error()))
			return
		}

		log.Printf("[VM] Domain defined successfully: %s", domain.UUID)

		if vm.LibvirtDomainUUID == "" || vm.LibvirtDomainUUID == "new-uuid" || vm.LibvirtDomainUUID == "defined-uuid" {
			if err := h.vmRepo.UpdateLibvirtDomainUUID(ctx, id, domain.UUID); err != nil {
				log.Printf("[VM] Failed to update LibvirtDomainUUID: %v", err)
			}
			log.Printf("[VM] LibvirtDomainUUID updated: %s", domain.UUID)
		}
	}

	state, _, _ := domain.GetState()
	log.Printf("[VM] Domain state before start: %d", state)

	if state == 1 {
		log.Printf("[VM] Domain is already running")
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

	// 立即更新状态为 starting，避免前端状态回跳
	if err := h.vmRepo.UpdateStatus(ctx, id, "starting"); err != nil {
		log.Printf("[VM] Failed to update status to starting: %v", err)
	}

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

// extractVNCPasswordFromXML extracts the VNC password from domain XML
func extractVNCPasswordFromXML(xmlDesc string) string {
	for _, line := range strings.Split(xmlDesc, "\n") {
		if strings.Contains(line, "<graphics") && strings.Contains(line, "type='vnc'") {
			for _, part := range strings.Fields(line) {
				if strings.HasPrefix(part, "passwd='") {
					passwd := strings.TrimPrefix(part, "passwd='")
					passwd = strings.TrimSuffix(passwd, "'")
					return passwd
				}
			}
		}
	}
	return ""
}

// extractSPICEPort extracts the SPICE port from domain XML
func extractSPICEPort(xmlDesc string) int {
	for _, line := range strings.Split(xmlDesc, "\n") {
		if strings.Contains(line, "<graphics") && strings.Contains(line, "type='spice'") {
			for _, part := range strings.Fields(line) {
				if strings.HasPrefix(part, "port='") {
					portStr := strings.TrimPrefix(part, "port='")
					portStr = strings.TrimSuffix(portStr, "'")
					var port int
					fmt.Sscanf(portStr, "%d", &port)
					return port
				}
			}
		}
	}
	return 0
}

// extractSPICEPasswordFromXML extracts the SPICE password from domain XML
func extractSPICEPasswordFromXML(xmlDesc string) string {
	for _, line := range strings.Split(xmlDesc, "\n") {
		if strings.Contains(line, "<graphics") && strings.Contains(line, "type='spice'") {
			for _, part := range strings.Fields(line) {
				if strings.HasPrefix(part, "passwd='") {
					passwd := strings.TrimPrefix(part, "passwd='")
					passwd = strings.TrimSuffix(passwd, "'")
					return passwd
				}
			}
		}
	}
	return ""
}

// generateBootOrder generates boot elements based on boot order string
// Format: "hd,cdrom,network" or "cdrom,hd,network" etc.
func generateBootOrder(bootOrder string) string {
	if bootOrder == "" {
		bootOrder = "hd,cdrom,network"
	}

	parts := strings.Split(bootOrder, ",")
	var bootXML string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		switch part {
		case "hd":
			bootXML += "    <boot dev='hd'/>\n"
		case "cdrom":
			bootXML += "    <boot dev='cdrom'/>\n"
		case "network":
			bootXML += "    <boot dev='network'/>\n"
		}
	}
	return bootXML
}

// generateISOConfig generates CD-ROM device configuration for ISO file
func generateISOConfig(isoPath string) string {
	if isoPath == "" || !exists(isoPath) {
		return ""
	}
	return fmt.Sprintf(`    <disk type='file' device='cdrom'>
      <driver name='qemu' type='raw'/>
      <source file='%s'/>
      <target dev='sda' bus='sata'/>
      <readonly/>
    </disk>`, isoPath)
}

func generateDomainXML(vm models.VirtualMachine, diskPath, isoPath string) string {
	// Note: VNC password is disabled for WebSocket proxy compatibility
	// The proxy forwards raw bytes and doesn't handle VNC auth protocol

	arch := vm.Architecture
	if arch == "" {
		arch = "x86_64"
	}

	var archConfig string
	switch arch {
	case "arm64", "aarch64":
		// ARM64 架构配置 - 在 x86 主机上使用 TCG 进行二进制翻译
		archConfig = fmt.Sprintf(`<domain type='qemu'>
  <name>%s</name>
  <uuid>%s</uuid>
  <memory unit='MiB'>%d</memory>
  <vcpu placement='static'>%d</vcpu>
  <os>
    <type arch='aarch64' machine='virt'>hvm</type>
    <loader readonly='yes' type='pflash'>/usr/share/AAVMF/AAVMF_CODE.fd</loader>
    <nvram template='/usr/share/AAVMF/AAVMF_VARS.fd'>/var/lib/libvirt/qemu/nvram/%s_VARS.fd</nvram>
%s  </os>
  <serial type='pty'>
    <target port='0'/>
  </serial>
  <console type='pty'>
    <target type='serial' port='0'/>
  </console>
  <features>
    <acpi/>
    <apic/>
    <gic version='3'/>
  </features>
  <cpu mode='custom' match='exact'>
    <model>cortex-a72</model>
  </cpu>
  <clock offset='utc'>
    <timer name='rtc' tickpolicy='catchup'/>
  </clock>
  <on_poweroff>destroy</on_poweroff>
  <on_reboot>restart</on_reboot>
  <on_crash>restart</on_crash>
  <devices>
    <emulator>/usr/bin/qemu-system-aarch64</emulator>
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2'/>
      <source file='%s'/>
      <target dev='vda' bus='virtio'/>
    </disk>
%s    <interface type='network'>
      <source network='default'/>
      <model type='virtio'/>
    </interface>
    <graphics type='spice' port='-1' autoport='yes' listen='0.0.0.0'>
      <listen type='address' address='0.0.0.0'/>
    </graphics>
    <video>
      <model type='qxl' ram='65536' vram='65536'/>
    </video>
    <controller type='usb' model='ehci'>
    </controller>
    <controller type='virtio-serial'>
    </controller>
    <channel type='spicevmc'>
      <target type='virtio' name='com.redhat.spice.0'/>
    </channel>
    <input type='tablet' bus='usb'>
    </input>
    <input type='mouse' bus='usb'>
    </input>
  </devices>
</domain>`, vm.Name, vm.ID.String(), vm.MemoryAllocated, vm.CPUAllocated, vm.ID.String(), generateBootOrder(vm.BootOrder), diskPath, generateISOConfig(isoPath))
	case "x86_64":
		archConfig = fmt.Sprintf(`<domain type='qemu'>
  <name>%s</name>
  <uuid>%s</uuid>
  <memory unit='MiB'>%d</memory>
  <vcpu placement='static'>%d</vcpu>
  <os>
    <type arch='x86_64' machine='q35'>hvm</type>
    <loader readonly='yes' type='pflash'>/usr/share/OVMF/OVMF_CODE.fd</loader>
    <nvram template='/usr/share/OVMF/OVMF_VARS.fd'>/var/lib/libvirt/qemu/nvram/%s_VARS.fd</nvram>
%s  </os>
  <serial type='pty'>
    <target port='0'/>
  </serial>
  <console type='pty'>
    <target type='serial' port='0'/>
  </console>
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
%s    <interface type='network'>
      <source network='default'/>
      <model type='virtio'/>
    </interface>
    <graphics type='spice' port='-1' autoport='yes' listen='0.0.0.0'>
      <listen type='address' address='0.0.0.0'/>
      <mouse mode='server'/>
    </graphics>
    <video>
      <model type='qxl' ram='65536' vram='65536'/>
    </video>
    <controller type='usb' model='qemu-xhci'/>
    <controller type='virtio-serial'/>
    <channel type='spicevmc'>
      <target type='virtio' name='com.redhat.spice.0'/>
    </channel>
    <input type='tablet' bus='virtio'>
    </input>
    <input type='mouse' bus='virtio'>
    </input>
  </devices>
</domain>`, vm.Name, vm.ID.String(), vm.MemoryAllocated, vm.CPUAllocated, vm.ID.String(), generateBootOrder(vm.BootOrder), diskPath, generateISOConfig(isoPath))
	}

	return archConfig
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

	if h.libvirt == nil || vm.LibvirtDomainUUID == "" {
		log.Printf("[VM] libvirt client is nil or LibvirtDomainUUID is empty, cannot stop VM")
		c.JSON(http.StatusServiceUnavailable, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "libvirt_service_unavailable"), "libvirt client is not initialized or domain not configured"))
		return
	}

	domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
	if err != nil {
		log.Printf("[VM] Domain not found in libvirt: %v", err)
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "vm_domain_not_found"), err.Error()))
		return
	}

	log.Printf("[VM] Attempting to shutdown VM: %s (libvirt: %s)", id, vm.LibvirtDomainUUID)

	// 立即更新状态为 stopping，避免前端状态回跳
	if err := h.vmRepo.UpdateStatus(ctx, id, "stopping"); err != nil {
		log.Printf("[VM] Failed to update status to stopping: %v", err)
	}

	if err := domain.Shutdown(); err != nil {
		log.Printf("[VM] Shutdown failed, using destroy: %v", err)
		domain.Free()
		domain, err = h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
		if err != nil {
			log.Printf("[VM] Domain not found: %v", err)
			h.vmRepo.UpdateStatus(ctx, id, "stopped")
			c.JSON(http.StatusOK, errors.Success(gin.H{"id": vm.ID, "status": "stopped"}))
			return
		}
		if err := domain.Destroy(); err != nil {
			log.Printf("[VM] Failed to destroy VM: %v", err)
			c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_stop_vm"), err.Error()))
			return
		}
		log.Printf("[VM] VM destroyed: %s", id)
		domain.Free()
	} else {
		log.Printf("[VM] Shutdown signal sent: %s", id)
		domain.Free()

		domain, err = h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
		if err != nil {
			log.Printf("[VM] Domain not found after shutdown: %v", err)
			h.vmRepo.UpdateStatus(ctx, id, "stopped")
			c.JSON(http.StatusOK, errors.Success(gin.H{"id": vm.ID, "status": "stopped"}))
			return
		}

		waitStart := time.Now()
		for time.Since(waitStart) < 30*time.Second {
			state, _, err := domain.GetState()
			if err != nil {
				log.Printf("[VM] Failed to get VM state: %v", err)
				break
			}
			if state == 0 {
				log.Printf("[VM] VM is now stopped: %s", id)
				domain.Free()
				h.vmRepo.UpdateStatus(ctx, id, "stopped")
				c.JSON(http.StatusOK, errors.Success(gin.H{"id": vm.ID, "status": "stopped"}))
				return
			}
			domain.Free()
			time.Sleep(1 * time.Second)
			domain, err = h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
			if err != nil {
				log.Printf("[VM] Domain not found during wait: %v", err)
				h.vmRepo.UpdateStatus(ctx, id, "stopped")
				c.JSON(http.StatusOK, errors.Success(gin.H{"id": vm.ID, "status": "stopped"}))
				return
			}
		}

		log.Printf("[VM] Shutdown timeout, using destroy: %s", id)
		if err := domain.Destroy(); err != nil {
			log.Printf("[VM] Failed to destroy VM after timeout: %v", err)
			state, _, _ := domain.GetState()
			domain.Free()
			if state == 5 {
				log.Printf("[VM] VM already stopped (state=%d), considering shutdown successful", state)
			} else {
				c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_stop_vm"), err.Error()))
				return
			}
		} else {
			domain.Free()
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

	if h.libvirt == nil || vm.LibvirtDomainUUID == "" {
		log.Printf("[VM] libvirt client is nil or LibvirtDomainUUID is empty, cannot force stop VM")
		c.JSON(http.StatusServiceUnavailable, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "libvirt_service_unavailable"), "libvirt client is not initialized or domain not configured"))
		return
	}

	domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
	if err != nil {
		log.Printf("[VM] Domain not found in libvirt: %v", err)
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "vm_domain_not_found"), err.Error()))
		return
	}

	if err := domain.Destroy(); err != nil {
		log.Printf("[VM] Failed to destroy domain: %v", err)
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_force_stop_vm"), err.Error()))
		return
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

	if h.libvirt == nil || vm.LibvirtDomainUUID == "" {
		log.Printf("[VM] libvirt client is nil or LibvirtDomainUUID is empty, cannot reboot VM")
		c.JSON(http.StatusServiceUnavailable, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "libvirt_service_unavailable"), "libvirt client is not initialized or domain not configured"))
		return
	}

	domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
	if err != nil {
		log.Printf("[VM] Domain not found in libvirt: %v", err)
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "vm_domain_not_found"), err.Error()))
		return
	}

	log.Printf("[VM] Rebooting VM: %s (libvirt: %s)", id, vm.LibvirtDomainUUID)

	if err := domain.Shutdown(); err != nil {
		log.Printf("[VM] Shutdown failed, using destroy: %v", err)
		domain.Free()
		domain, err = h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
		if err != nil {
			log.Printf("[VM] Domain not found: %v", err)
		} else {
			if err := domain.Destroy(); err != nil {
				log.Printf("[VM] Failed to destroy VM for reboot: %v", err)
				c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_reboot_vm"), err.Error()))
				return
			}
			domain.Free()
			log.Printf("[VM] VM destroyed for reboot: %s", id)
		}
	} else {
		log.Printf("[VM] Shutdown signal sent, waiting for VM to stop...")
		domain.Free()

		waitStart := time.Now()
		stopped := false
		for time.Since(waitStart) < 30*time.Second {
			domain, err = h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
			if err != nil {
				log.Printf("[VM] Domain not found during wait, treating as stopped: %v", err)
				stopped = true
				break
			}
			state, _, err := domain.GetState()
			domain.Free()
			if err != nil {
				log.Printf("[VM] Failed to get VM state: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}
			if state == 0 {
				stopped = true
				break
			}
			time.Sleep(1 * time.Second)
		}

		if !stopped {
			log.Printf("[VM] Shutdown timeout, using destroy: %s", id)
			domain, err = h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
			if err == nil {
				if err := domain.Destroy(); err != nil {
					state, _, _ := domain.GetState()
					if state == 5 {
						log.Printf("[VM] VM already stopped (state=%d), considering shutdown successful", state)
					} else {
						log.Printf("[VM] Failed to destroy VM after timeout: %v", err)
					}
				} else {
					log.Printf("[VM] VM destroyed after timeout: %s", id)
				}
				domain.Free()
			}
		}
	}

	domain, err = h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
	if err != nil {
		log.Printf("[VM] Domain not found when trying to start after reboot: %v", err)
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "vm_domain_not_found"), err.Error()))
		return
	}

	log.Printf("[VM] Starting VM again...")
	if err := domain.Create(); err != nil {
		log.Printf("[VM] Failed to start domain after reboot: %v", err)
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_reboot_vm"), err.Error()))
		return
	}
	domain.Free()

	log.Printf("[VM] VM rebooted successfully: %s", id)

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

	if h.libvirt == nil || vm.LibvirtDomainUUID == "" {
		log.Printf("[VM] libvirt client is nil or LibvirtDomainUUID is empty, cannot suspend VM")
		c.JSON(http.StatusServiceUnavailable, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "libvirt_service_unavailable"), "libvirt client is not initialized or domain not configured"))
		return
	}

	domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
	if err != nil {
		log.Printf("[VM] Domain not found in libvirt: %v", err)
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "vm_domain_not_found"), err.Error()))
		return
	}

	if err := domain.Suspend(); err != nil {
		log.Printf("[VM] Failed to suspend domain: %v", err)
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_suspend_vm"), err.Error()))
		return
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

	if h.libvirt == nil || vm.LibvirtDomainUUID == "" {
		log.Printf("[VM] libvirt client is nil or LibvirtDomainUUID is empty, cannot resume VM")
		c.JSON(http.StatusServiceUnavailable, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "libvirt_service_unavailable"), "libvirt client is not initialized or domain not configured"))
		return
	}

	domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
	if err != nil {
		log.Printf("[VM] Domain not found in libvirt: %v", err)
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "vm_domain_not_found"), err.Error()))
		return
	}

	if err := domain.Resume(); err != nil {
		log.Printf("[VM] Failed to resume domain: %v", err)
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_resume_vm"), err.Error()))
		return
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

	// Get SPICE port from libvirt domain XML
	spicePort := 0
	spicePassword := ""
	if domain, err := h.libvirt.LookupByUUID(vm.ID.String()); err == nil {
		if xmlDesc, err := domain.GetXMLDesc(); err == nil {
			spicePort = extractSPICEPort(xmlDesc)
			spicePassword = extractSPICEPasswordFromXML(xmlDesc)
		}
	}

	scheme := "ws"
	if c.Request.TLS != nil || c.Request.URL.Scheme == "https" || c.Request.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "wss"
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"type":          "spice",
		"host":          "127.0.0.1",
		"port":          spicePort,
		"password":      spicePassword,
		"websocket_url": fmt.Sprintf("%s://%s/ws/spice/%s", scheme, c.Request.Host, vm.ID),
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

func (h *VMHandler) StartInstallation(c *gin.Context) {
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

	if h.libvirt == nil {
		c.JSON(http.StatusServiceUnavailable, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "libvirt_service_unavailable"), "libvirt client is not initialized"))
		return
	}

	template, err := h.templateRepo.FindByID(ctx, vm.TemplateID.String())
	if err != nil || template.ISOPath == "" {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "no_iso_attached"), "template has no ISO path"))
		return
	}

	vm.InstallStatus = "installing"
	vm.InstallProgress = 0
	vm.BootOrder = "cdrom,hd,network"
	if err := h.vmRepo.Update(ctx, vm); err != nil {
		log.Printf("[Installation] Failed to update VM status: %v", err)
	}

	domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
	if err == nil {
		domain.Destroy()
		domain.Free()
	}

	diskPath := vm.DiskPath
	if diskPath == "" {
		diskPath = fmt.Sprintf("%s/%s.qcow2", h.storagePath, vm.ID.String())
	}

	log.Printf("[Installation] Starting VM %s in installation mode with ISO: %s", vm.Name, template.ISOPath)

	domainXML := generateDomainXML(*vm, diskPath, template.ISOPath)
	log.Printf("[Installation] Domain XML:\n%s", domainXML)

	domain, err = h.libvirt.DefineXML(domainXML)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "failed_to_define_domain"), err.Error()))
		return
	}
	defer domain.Free()

	if err := domain.Create(); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "failed_to_start_vm"), err.Error()))
		return
	}

	vmUUIDStr, _ := domain.GetUUIDString()
	vm.LibvirtDomainUUID = vmUUIDStr
	vm.Status = "running"
	vm.InstallStatus = "installing"

	if err := h.vmRepo.Update(ctx, vm); err != nil {
		log.Printf("[Installation] Failed to update VM: %v", err)
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"message":        "VM started in installation mode",
		"status":         "running",
		"install_status": "installing",
		"boot_order":     vm.BootOrder,
	}))
}

func (h *VMHandler) FinishInstallation(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "vm_not_running"), "VM is not running"))
		return
	}

	if h.libvirt == nil {
		c.JSON(http.StatusServiceUnavailable, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "libvirt_service_unavailable"), "libvirt client is not initialized"))
		return
	}

	domain, err := h.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "domain_not_found"), err.Error()))
		return
	}
	defer domain.Free()

	if err := domain.Destroy(); err != nil {
		log.Printf("[Installation] Warning: failed to destroy domain: %v", err)
	}

	vm.BootOrder = "hd,cdrom,network"
	vm.IsInstalled = true
	vm.InstallStatus = "completed"
	vm.InstallProgress = 100

	domainXML := generateDomainXML(*vm, vm.DiskPath, "")
	log.Printf("[Installation] Domain XML after install:\n%s", domainXML)

	domain, err = h.libvirt.DefineXML(domainXML)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "failed_to_define_domain"), err.Error()))
		return
	}
	defer domain.Free()

	if err := domain.Create(); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "failed_to_start_vm"), err.Error()))
		return
	}

	vmUUIDStr, _ := domain.GetUUIDString()
	vm.LibvirtDomainUUID = vmUUIDStr
	vm.Status = "running"

	if err := h.vmRepo.Update(ctx, vm); err != nil {
		log.Printf("[Installation] Failed to update VM: %v", err)
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"message":      "Installation completed, VM started from hard disk",
		"status":       "running",
		"is_installed": true,
		"boot_order":   vm.BootOrder,
	}))
}

func (h *VMHandler) InstallAgent(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	var req struct {
		AgentType string `json:"agent_type"`
		Script    string `json:"script"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		req.AgentType = "spice-vdagent"
	}

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
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "vm_not_running"), "VM is not running"))
		return
	}

	if h.libvirt == nil {
		c.JSON(http.StatusServiceUnavailable, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "libvirt_service_unavailable"), "libvirt client is not initialized"))
		return
	}

	var script string
	if req.Script != "" {
		script = req.Script
	} else {
		switch req.AgentType {
		case "spice-vdagent":
			script = `#!/bin/bash
apt-get update && apt-get install -y spice-vdagent 2>/dev/null || \
yum install -y spice-vdagent 2>/dev/null || \
zypper install -y spice-vdagent 2>/dev/null
systemctl enable spice-vdagent 2>/dev/null || true
systemctl start spice-vdagent 2>/dev/null || true
echo "SPICE vdagent installation completed"
`
		default:
			script = req.Script
		}
	}

	scriptPath := fmt.Sprintf("/tmp/install_agent_%s.sh", vm.ID.String())
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_write_script"), err.Error()))
		return
	}
	defer os.Remove(scriptPath)

	cmd := exec.Command("bash", scriptPath)
	cmd.Run()

	vm.AgentInstalled = true
	if err := h.vmRepo.Update(ctx, vm); err != nil {
		log.Printf("[Agent] Failed to update VM: %v", err)
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"message":    "Agent installation script prepared",
		"agent_type": req.AgentType,
		"script":     script,
		"note":       "Please run the script inside the VM manually via console",
	}))
}

func (h *VMHandler) GetInstallationStatus(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	vm, err := h.vmRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, t(c, "vm_not_found_id"), id))
		return
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"is_installed":     vm.IsInstalled,
		"install_status":   vm.InstallStatus,
		"install_progress": vm.InstallProgress,
		"agent_installed":  vm.AgentInstalled,
		"boot_order":       vm.BootOrder,
		"current_status":   vm.Status,
	}))
}

func (h *VMHandler) MountISO(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	var req struct {
		ISOID string `json:"isoId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
		return
	}

	vm, err := h.vmRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, t(c, "vm_not_found_id"), id))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, t(c, "permission_denied_not_vm_owner"), "not VM owner"))
		return
	}

	if vm.LibvirtDomainUUID == "" {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "vm_domain_not_found"), "VM has no libvirt domain"))
		return
	}

	iso, err := h.isoRepo.FindByID(ctx, req.ISOID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeNotFound, t(c, "iso_not_found"), req.ISOID))
		return
	}

	if h.libvirt == nil {
		c.JSON(http.StatusServiceUnavailable, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "libvirt_service_unavailable"), "libvirt client is not initialized"))
		return
	}

	if err := h.libvirt.AttachISO(vm.LibvirtDomainUUID, iso.ISOPath); err != nil {
		log.Printf("[VM] Failed to mount ISO: %v", err)
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_mount_iso"), err.Error()))
		return
	}

	log.Printf("[VM] ISO mounted successfully: VM=%s, ISO=%s", id, iso.Name)

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"vmId":    id,
		"isoId":   iso.ID,
		"isoName": iso.Name,
		"isoPath": iso.ISOPath,
	}))
}

func (h *VMHandler) UnmountISO(c *gin.Context) {
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

	if vm.LibvirtDomainUUID == "" {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "vm_domain_not_found"), "VM has no libvirt domain"))
		return
	}

	if h.libvirt == nil {
		c.JSON(http.StatusServiceUnavailable, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "libvirt_service_unavailable"), "libvirt client is not initialized"))
		return
	}

	if err := h.libvirt.DetachISO(vm.LibvirtDomainUUID); err != nil {
		log.Printf("[VM] Failed to unmount ISO: %v", err)
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_unmount_iso"), err.Error()))
		return
	}

	log.Printf("[VM] ISO unmounted successfully: VM=%s", id)

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"vmId": id,
	}))
}

func (h *VMHandler) GetMountedISO(c *gin.Context) {
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

	if vm.LibvirtDomainUUID == "" {
		c.JSON(http.StatusOK, errors.Success(gin.H{
			"vmId":    id,
			"mounted": false,
			"isoPath": "",
			"isoId":   "",
			"isoName": "",
		}))
		return
	}

	if h.libvirt == nil {
		c.JSON(http.StatusOK, errors.Success(gin.H{
			"vmId":    id,
			"mounted": false,
			"isoPath": "",
			"isoId":   "",
			"isoName": "",
		}))
		return
	}

	isoPath, err := h.libvirt.GetMountedISO(vm.LibvirtDomainUUID)
	if err != nil {
		log.Printf("[VM] Failed to get mounted ISO: %v", err)
		c.JSON(http.StatusOK, errors.Success(gin.H{
			"vmId":    id,
			"mounted": false,
			"isoPath": "",
			"isoId":   "",
			"isoName": "",
		}))
		return
	}

	if isoPath == "" {
		c.JSON(http.StatusOK, errors.Success(gin.H{
			"vmId":    id,
			"mounted": false,
			"isoPath": "",
			"isoId":   "",
			"isoName": "",
		}))
		return
	}

	iso, err := h.isoRepo.FindByPath(ctx, isoPath)
	if err != nil {
		c.JSON(http.StatusOK, errors.Success(gin.H{
			"vmId":    id,
			"mounted": true,
			"isoPath": isoPath,
			"isoId":   "",
			"isoName": "",
		}))
		return
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"vmId":    id,
		"mounted": true,
		"isoPath": isoPath,
		"isoId":   iso.ID,
		"isoName": iso.Name,
	}))
}
