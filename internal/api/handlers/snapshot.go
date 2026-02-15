package handlers

import (
	"net/http"
	"time"

	"vmmanager/internal/api/errors"
	"vmmanager/internal/libvirt"
	"vmmanager/internal/models"
	"vmmanager/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SnapshotHandler struct {
	vmRepo       *repository.VMRepository
	snapshotRepo *repository.VMSnapshotRepository
	libvirt      *libvirt.Client
}

func NewSnapshotHandler(vmRepo *repository.VMRepository, snapshotRepo *repository.VMSnapshotRepository, libvirtClient *libvirt.Client) *SnapshotHandler {
	return &SnapshotHandler{
		vmRepo:       vmRepo,
		snapshotRepo: snapshotRepo,
		libvirt:      libvirtClient,
	}
}

type CreateSnapshotRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

func (h *SnapshotHandler) CreateSnapshot(c *gin.Context) {
	vmID := c.Param("id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, vmID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, t(c, "vm.vmNotFound"), vmID))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, t(c, "common.permissionDenied"), "not VM owner"))
		return
	}

	var req CreateSnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "common.validationError"), err.Error()))
		return
	}

	existing, _ := h.snapshotRepo.FindByVMAndName(ctx, vmID, req.Name)
	if existing != nil {
		c.JSON(http.StatusConflict, errors.FailWithCode(errors.ErrCodeConflict, t(c, "snapshot.nameExists")))
		return
	}

	if h.libvirt != nil && vm.LibvirtDomainUUID != "" {
		if err := h.libvirt.CreateSnapshot(vm.LibvirtDomainUUID, req.Name, req.Description); err != nil {
			c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "snapshot.failedToCreate"), err.Error()))
			return
		}
	}

	snapshot := &models.VMSnapshot{
		VMID:        uuid.MustParse(vmID),
		Name:        req.Name,
		Description: req.Description,
		Status:      "created",
		CreatedBy:   &userUUID,
	}

	if err := h.snapshotRepo.Create(ctx, snapshot); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "snapshot.failedToCreate"), err.Error()))
		return
	}

	c.JSON(http.StatusCreated, errors.Success(snapshot))
}

func (h *SnapshotHandler) ListSnapshots(c *gin.Context) {
	vmID := c.Param("id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, vmID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, t(c, "vm.vmNotFound"), vmID))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, t(c, "common.permissionDenied"), "not VM owner"))
		return
	}

	snapshots, err := h.snapshotRepo.ListByVM(ctx, vmID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "snapshot.failedToList"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(snapshots))
}

func (h *SnapshotHandler) GetSnapshot(c *gin.Context) {
	vmID := c.Param("id")
	snapshotID := c.Param("snapshot_id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, vmID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, t(c, "vm.vmNotFound"), vmID))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, t(c, "common.permissionDenied"), "not VM owner"))
		return
	}

	snapshot, err := h.snapshotRepo.FindByID(ctx, snapshotID)
	if err != nil {
		if err == repository.ErrSnapshotNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "snapshot.notFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "snapshot.failedToGet"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(snapshot))
}

func (h *SnapshotHandler) RestoreSnapshot(c *gin.Context) {
	vmID := c.Param("id")
	snapshotID := c.Param("snapshot_id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, vmID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, t(c, "vm.vmNotFound"), vmID))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, t(c, "common.permissionDenied"), "not VM owner"))
		return
	}

	snapshot, err := h.snapshotRepo.FindByID(ctx, snapshotID)
	if err != nil {
		if err == repository.ErrSnapshotNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "snapshot.notFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "snapshot.failedToGet"), err.Error()))
		return
	}

	if h.libvirt != nil && vm.LibvirtDomainUUID != "" {
		if err := h.libvirt.RevertToSnapshot(vm.LibvirtDomainUUID, snapshot.Name); err != nil {
			c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "snapshot.failedToRestore"), err.Error()))
			return
		}
	}

	if err := h.snapshotRepo.SetCurrentSnapshot(ctx, vmID, snapshotID); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "snapshot.failedToRestore"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(map[string]string{
		"message":    t(c, "snapshot.restoreSuccess"),
		"snapshotId": snapshotID,
	}))
}

func (h *SnapshotHandler) DeleteSnapshot(c *gin.Context) {
	vmID := c.Param("id")
	snapshotID := c.Param("snapshot_id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, vmID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, t(c, "vm.vmNotFound"), vmID))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, t(c, "common.permissionDenied"), "not VM owner"))
		return
	}

	snapshot, err := h.snapshotRepo.FindByID(ctx, snapshotID)
	if err != nil {
		if err == repository.ErrSnapshotNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "snapshot.notFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "snapshot.failedToGet"), err.Error()))
		return
	}

	if snapshot.IsCurrent {
		c.JSON(http.StatusBadRequest, errors.FailWithCode(errors.ErrCodeBadRequest, t(c, "snapshot.cannotDeleteCurrent")))
		return
	}

	if h.libvirt != nil && vm.LibvirtDomainUUID != "" {
		if err := h.libvirt.DeleteSnapshot(vm.LibvirtDomainUUID, snapshot.Name); err != nil {
			c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "snapshot.failedToDelete"), err.Error()))
			return
		}
	}

	if err := h.snapshotRepo.Delete(ctx, snapshotID); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "snapshot.failedToDelete"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(nil))
}

func (h *SnapshotHandler) SyncSnapshots(c *gin.Context) {
	vmID := c.Param("id")
	ctx := c.Request.Context()

	userID, _ := c.Get("user_id")
	role, _ := c.Get("role")
	userUUID, _ := uuid.Parse(userID.(string))

	vm, err := h.vmRepo.FindByID(ctx, vmID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeVMNotFound, t(c, "vm.vmNotFound"), vmID))
		return
	}

	if role != "admin" && vm.OwnerID != userUUID {
		c.JSON(http.StatusForbidden, errors.FailWithDetails(errors.ErrCodeForbidden, t(c, "common.permissionDenied"), "not VM owner"))
		return
	}

	if h.libvirt == nil || vm.LibvirtDomainUUID == "" {
		c.JSON(http.StatusOK, errors.Success([]string{}))
		return
	}

	libvirtSnapshots, err := h.libvirt.ListSnapshots(vm.LibvirtDomainUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "snapshot.failedToList"), err.Error()))
		return
	}

	dbSnapshots, _ := h.snapshotRepo.ListByVM(ctx, vmID)
	dbSnapshotMap := make(map[string]*models.VMSnapshot)
	for i := range dbSnapshots {
		dbSnapshotMap[dbSnapshots[i].Name] = &dbSnapshots[i]
	}

	for _, name := range libvirtSnapshots {
		if _, exists := dbSnapshotMap[name]; !exists {
			info, err := h.libvirt.GetSnapshotInfo(vm.LibvirtDomainUUID, name)
			if err != nil {
				continue
			}

			snapshot := &models.VMSnapshot{
				VMID:        uuid.MustParse(vmID),
				Name:        name,
				Description: info.Description,
				Status:      "created",
				IsCurrent:   info.IsCurrent,
				CreatedBy:   &userUUID,
				CreatedAt:   time.Now(),
			}

			h.snapshotRepo.Create(ctx, snapshot)
		}
	}

	c.JSON(http.StatusOK, errors.Success(map[string]int{
		"synced": len(libvirtSnapshots),
	}))
}
