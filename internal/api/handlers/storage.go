package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"vmmanager/internal/api/errors"
	"vmmanager/internal/libvirt"
	"vmmanager/internal/models"
	"vmmanager/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type StorageHandler struct {
	repo       *repository.Repositories
	libvirt    *libvirt.Client
}

func NewStorageHandler(repo *repository.Repositories, libvirtClient *libvirt.Client) *StorageHandler {
	return &StorageHandler{
		repo:    repo,
		libvirt: libvirtClient,
	}
}

type CreatePoolRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	PoolType    string `json:"poolType"`
	TargetPath  string `json:"targetPath"`
	SourcePath  string `json:"sourcePath"`
	Autostart   *bool  `json:"autostart"`
}

func (h *StorageHandler) CreatePool(c *gin.Context) {
	ctx := c.Request.Context()
	var req CreatePoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
		return
	}

	existing, _ := h.repo.StoragePool.FindByName(ctx, req.Name)
	if existing != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithCode(errors.ErrCodeBadRequest, t(c, "storage.poolNameExists")))
		return
	}

	poolType := req.PoolType
	if poolType == "" {
		poolType = "dir"
	}

	autostart := true
	if req.Autostart != nil {
		autostart = *req.Autostart
	}

	userID, _ := c.Get("user_id")
	userUUID, _ := uuid.Parse(userID.(string))

	pool := &models.StoragePool{
		Name:        req.Name,
		Description: req.Description,
		PoolType:    poolType,
		TargetPath:  req.TargetPath,
		SourcePath:  req.SourcePath,
		Autostart:   autostart,
		Active:      false,
		CreatedBy:   &userUUID,
	}

	xmlDef := h.generatePoolXML(pool)
	pool.XMLDef = xmlDef

	if h.libvirt != nil {
		if err := h.libvirt.StoragePoolDefineXML(xmlDef); err != nil {
			c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "storage.failedToCreateLibvirt"), err.Error()))
			return
		}

		if autostart {
			if err := h.libvirt.StoragePoolSetAutostart(req.Name, true); err != nil {
				log.Printf("[STORAGE] Warning: Failed to set autostart for pool %s: %v", req.Name, err)
			}
		}

		if err := h.libvirt.StoragePoolCreate(req.Name); err != nil {
			log.Printf("[STORAGE] Warning: Failed to start pool %s: %v", req.Name, err)
		} else {
			pool.Active = true
		}

		if info, err := h.libvirt.StoragePoolGetInfo(req.Name); err == nil {
			pool.Capacity = int64(info.Capacity)
			pool.Available = int64(info.Available)
			pool.Used = int64(info.Used)
		}
	}

	if err := h.repo.StoragePool.Create(ctx, pool); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "storage.failedToCreatePool"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(pool))
}

func (h *StorageHandler) generatePoolXML(pool *models.StoragePool) string {
	switch pool.PoolType {
	case "dir":
		return fmt.Sprintf(`<pool type='dir'>
  <name>%s</name>
  <target>
    <path>%s</path>
  </target>
</pool>`, pool.Name, pool.TargetPath)
	case "fs":
		return fmt.Sprintf(`<pool type='fs'>
  <name>%s</name>
  <source>
    <dir path='%s'/>
  </source>
  <target>
    <path>%s</path>
  </target>
</pool>`, pool.Name, pool.SourcePath, pool.TargetPath)
	case "logical":
		return fmt.Sprintf(`<pool type='logical'>
  <name>%s</name>
  <target>
    <path>/dev/%s</path>
  </target>
</pool>`, pool.Name, pool.Name)
	default:
		return fmt.Sprintf(`<pool type='dir'>
  <name>%s</name>
  <target>
    <path>%s</path>
  </target>
</pool>`, pool.Name, pool.TargetPath)
	}
}

func (h *StorageHandler) ListPools(c *gin.Context) {
	ctx := c.Request.Context()
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	pools, total, err := h.repo.StoragePool.List(ctx, (page-1)*pageSize, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "storage.failedToListPools"), err.Error()))
		return
	}

	for i := range pools {
		if h.libvirt != nil {
			if info, err := h.libvirt.StoragePoolGetInfo(pools[i].Name); err == nil {
				pools[i].Capacity = int64(info.Capacity)
				pools[i].Available = int64(info.Available)
				pools[i].Used = int64(info.Used)
				pools[i].Active = info.Active
			}
		}
	}

	c.JSON(http.StatusOK, errors.SuccessWithPage(pools, total, page, pageSize))
}

func (h *StorageHandler) GetPool(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	pool, err := h.repo.StoragePool.FindByID(ctx, id)
	if err != nil {
		if err == repository.ErrStoragePoolNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "storage.poolNotFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "storage.failedToGetPool"), err.Error()))
		return
	}

	if h.libvirt != nil {
		if info, err := h.libvirt.StoragePoolGetInfo(pool.Name); err == nil {
			pool.Capacity = int64(info.Capacity)
			pool.Available = int64(info.Available)
			pool.Used = int64(info.Used)
			pool.Active = info.Active
		}
	}

	c.JSON(http.StatusOK, errors.Success(pool))
}

func (h *StorageHandler) UpdatePool(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	var req struct {
		Description string `json:"description"`
		Autostart   *bool  `json:"autostart"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
		return
	}

	pool, err := h.repo.StoragePool.FindByID(ctx, id)
	if err != nil {
		if err == repository.ErrStoragePoolNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "storage.poolNotFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "storage.failedToGetPool"), err.Error()))
		return
	}

	if req.Description != "" {
		pool.Description = req.Description
	}

	if req.Autostart != nil {
		pool.Autostart = *req.Autostart
		if h.libvirt != nil {
			if err := h.libvirt.StoragePoolSetAutostart(pool.Name, pool.Autostart); err != nil {
				log.Printf("[STORAGE] Warning: Failed to set autostart for pool %s: %v", pool.Name, err)
			}
		}
	}

	if err := h.repo.StoragePool.Update(ctx, pool); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "storage.failedToUpdatePool"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(pool))
}

func (h *StorageHandler) DeletePool(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	pool, err := h.repo.StoragePool.FindByID(ctx, id)
	if err != nil {
		if err == repository.ErrStoragePoolNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "storage.poolNotFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "storage.failedToGetPool"), err.Error()))
		return
	}

	if h.libvirt != nil {
		if pool.Active {
			if err := h.libvirt.StoragePoolDestroy(pool.Name); err != nil {
				log.Printf("[STORAGE] Warning: Failed to destroy pool %s: %v", pool.Name, err)
			}
		}
		if err := h.libvirt.StoragePoolUndefine(pool.Name); err != nil {
			log.Printf("[STORAGE] Warning: Failed to undefine pool %s: %v", pool.Name, err)
		}
	}

	if err := h.repo.StoragePool.Delete(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "storage.failedToDeletePool"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(nil))
}

func (h *StorageHandler) StartPool(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	pool, err := h.repo.StoragePool.FindByID(ctx, id)
	if err != nil {
		if err == repository.ErrStoragePoolNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "storage.poolNotFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "storage.failedToGetPool"), err.Error()))
		return
	}

	if h.libvirt != nil {
		if err := h.libvirt.StoragePoolCreate(pool.Name); err != nil {
			c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "storage.failedToStartPool"), err.Error()))
			return
		}

		pool.Active = true
		if err := h.repo.StoragePool.SetActive(ctx, id, true); err != nil {
			log.Printf("[STORAGE] Warning: Failed to update pool active status: %v", err)
		}
	}

	c.JSON(http.StatusOK, errors.Success(pool))
}

func (h *StorageHandler) StopPool(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	pool, err := h.repo.StoragePool.FindByID(ctx, id)
	if err != nil {
		if err == repository.ErrStoragePoolNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "storage.poolNotFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "storage.failedToGetPool"), err.Error()))
		return
	}

	if h.libvirt != nil {
		if err := h.libvirt.StoragePoolDestroy(pool.Name); err != nil {
			c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "storage.failedToStopPool"), err.Error()))
			return
		}

		pool.Active = false
		if err := h.repo.StoragePool.SetActive(ctx, id, false); err != nil {
			log.Printf("[STORAGE] Warning: Failed to update pool active status: %v", err)
		}
	}

	c.JSON(http.StatusOK, errors.Success(pool))
}

func (h *StorageHandler) RefreshPool(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	pool, err := h.repo.StoragePool.FindByID(ctx, id)
	if err != nil {
		if err == repository.ErrStoragePoolNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "storage.poolNotFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "storage.failedToGetPool"), err.Error()))
		return
	}

	if h.libvirt != nil {
		if err := h.libvirt.StoragePoolRefresh(pool.Name); err != nil {
			c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "storage.failedToRefreshPool"), err.Error()))
			return
		}

		if info, err := h.libvirt.StoragePoolGetInfo(pool.Name); err == nil {
			pool.Capacity = int64(info.Capacity)
			pool.Available = int64(info.Available)
			pool.Used = int64(info.Used)
			if err := h.repo.StoragePool.UpdateCapacity(ctx, id, pool.Capacity, pool.Available, pool.Used); err != nil {
				log.Printf("[STORAGE] Warning: Failed to update pool capacity: %v", err)
			}
		}
	}

	c.JSON(http.StatusOK, errors.Success(pool))
}

func (h *StorageHandler) ListVolumes(c *gin.Context) {
	ctx := c.Request.Context()
	poolID := c.Param("id")

	pool, err := h.repo.StoragePool.FindByID(ctx, poolID)
	if err != nil {
		if err == repository.ErrStoragePoolNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "storage.poolNotFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "storage.failedToGetPool"), err.Error()))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	volumes, total, err := h.repo.StorageVolume.ListByPool(ctx, poolID, (page-1)*pageSize, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "storage.failedToListVolumes"), err.Error()))
		return
	}

	if h.libvirt != nil && pool.Active {
		libvirtVols, err := h.libvirt.StorageVolumeList(pool.Name)
		if err == nil {
			for _, lv := range libvirtVols {
				found := false
				for i, v := range volumes {
					if v.Name == lv.Name {
						volumes[i].Capacity = int64(lv.Capacity)
						volumes[i].Allocation = int64(lv.Allocation)
						volumes[i].Path = lv.Path
						found = true
						break
					}
				}
				if !found {
					volumes = append(volumes, models.StorageVolume{
						PoolID:     pool.ID,
						Name:       lv.Name,
						Capacity:   int64(lv.Capacity),
						Allocation: int64(lv.Allocation),
						Path:       lv.Path,
					})
				}
			}
		}
	}

	c.JSON(http.StatusOK, errors.SuccessWithPage(volumes, total, page, pageSize))
}

type CreateVolumeRequest struct {
	Name     string `json:"name" binding:"required"`
	Capacity int64  `json:"capacity" binding:"required"`
	Format   string `json:"format"`
}

func (h *StorageHandler) CreateVolume(c *gin.Context) {
	ctx := c.Request.Context()
	poolID := c.Param("id")

	pool, err := h.repo.StoragePool.FindByID(ctx, poolID)
	if err != nil {
		if err == repository.ErrStoragePoolNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "storage.poolNotFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "storage.failedToGetPool"), err.Error()))
		return
	}

	var req CreateVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
		return
	}

	format := req.Format
	if format == "" {
		format = "qcow2"
	}

	if h.libvirt != nil && pool.Active {
		if err := h.libvirt.StorageVolumeCreate(pool.Name, req.Name, req.Capacity, format); err != nil {
			c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "storage.failedToCreateVolume"), err.Error()))
			return
		}
	}

	volume := &models.StorageVolume{
		PoolID:   pool.ID,
		Name:     req.Name,
		Capacity: req.Capacity,
		Format:   format,
	}

	if h.libvirt != nil && pool.Active {
		if vols, err := h.libvirt.StorageVolumeList(pool.Name); err == nil {
			for _, v := range vols {
				if v.Name == req.Name {
					volume.Allocation = int64(v.Allocation)
					volume.Path = v.Path
					break
				}
			}
		}
	}

	if err := h.repo.StorageVolume.Create(ctx, volume); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "storage.failedToCreateVolume"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(volume))
}

func (h *StorageHandler) DeleteVolume(c *gin.Context) {
	ctx := c.Request.Context()
	poolID := c.Param("id")
	volumeID := c.Param("volume_id")

	pool, err := h.repo.StoragePool.FindByID(ctx, poolID)
	if err != nil {
		if err == repository.ErrStoragePoolNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "storage.poolNotFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "storage.failedToGetPool"), err.Error()))
		return
	}

	volume, err := h.repo.StorageVolume.FindByID(ctx, volumeID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "storage.volumeNotFound")))
		return
	}

	if h.libvirt != nil && pool.Active {
		if err := h.libvirt.StorageVolumeDelete(pool.Name, volume.Name); err != nil {
			c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "storage.failedToDeleteVolume"), err.Error()))
			return
		}
	}

	if err := h.repo.StorageVolume.Delete(ctx, volumeID); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "storage.failedToDeleteVolume"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(nil))
}
