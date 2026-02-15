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

type VirtualNetworkHandler struct {
	repo    *repository.VirtualNetworkRepository
	libvirt *libvirt.Client
}

func NewVirtualNetworkHandler(repo *repository.VirtualNetworkRepository, libvirtClient *libvirt.Client) *VirtualNetworkHandler {
	return &VirtualNetworkHandler{
		repo:    repo,
		libvirt: libvirtClient,
	}
}

type CreateNetworkRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	NetworkType string `json:"networkType"`
	BridgeName  string `json:"bridgeName"`
	Subnet      string `json:"subnet" binding:"required"`
	Gateway     string `json:"gateway" binding:"required"`
	DHCPStart   string `json:"dhcpStart"`
	DHCPEnd     string `json:"dhcpEnd"`
	DHCPEnabled *bool  `json:"dhcpEnabled"`
	Autostart   *bool  `json:"autostart"`
}

type UpdateNetworkRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	BridgeName  string `json:"bridgeName"`
	Subnet      string `json:"subnet"`
	Gateway     string `json:"gateway"`
	DHCPStart   string `json:"dhcpStart"`
	DHCPEnd     string `json:"dhcpEnd"`
	DHCPEnabled *bool  `json:"dhcpEnabled"`
	Autostart   *bool  `json:"autostart"`
}

func (h *VirtualNetworkHandler) List(c *gin.Context) {
	ctx := c.Request.Context()

	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil {
			page = val
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if val, err := strconv.Atoi(ps); err == nil {
			pageSize = val
		}
	}

	offset := (page - 1) * pageSize
	networks, total, err := h.repo.List(ctx, offset, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "network.failedToList"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.SuccessWithPage(networks, total, page, pageSize))
}

func (h *VirtualNetworkHandler) Get(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	network, err := h.repo.FindByID(ctx, id)
	if err != nil {
		if err == repository.ErrNetworkNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "network.notFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "network.failedToGet"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(network))
}

func (h *VirtualNetworkHandler) Create(c *gin.Context) {
	ctx := c.Request.Context()
	var req CreateNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
		return
	}

	existing, _ := h.repo.FindByName(ctx, req.Name)
	if existing != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithCode(errors.ErrCodeBadRequest, t(c, "network.nameExists")))
		return
	}

	networkType := req.NetworkType
	if networkType == "" {
		networkType = "nat"
	}

	dhcpEnabled := true
	if req.DHCPEnabled != nil {
		dhcpEnabled = *req.DHCPEnabled
	}

	autostart := true
	if req.Autostart != nil {
		autostart = *req.Autostart
	}

	userID, _ := c.Get("user_id")
	userUUID, _ := uuid.Parse(userID.(string))

	network := &models.VirtualNetwork{
		Name:        req.Name,
		Description: req.Description,
		NetworkType: networkType,
		BridgeName:  req.BridgeName,
		Subnet:      req.Subnet,
		Gateway:     req.Gateway,
		DHCPStart:   req.DHCPStart,
		DHCPEnd:     req.DHCPEnd,
		DHCPEnabled: dhcpEnabled,
		Autostart:   autostart,
		Active:      false,
		CreatedBy:   &userUUID,
	}

	xmlDef := h.generateNetworkXML(network)
	network.XMLDef = xmlDef

	if h.libvirt != nil {
		if err := h.libvirt.NetworkDefineXML(xmlDef); err != nil {
			c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "network.failedToCreateLibvirt"), err.Error()))
			return
		}

		if autostart {
			if err := h.libvirt.NetworkSetAutostart(req.Name, true); err != nil {
				log.Printf("[NETWORK] Warning: Failed to set autostart for network %s: %v", req.Name, err)
			}
		}

		if err := h.libvirt.NetworkCreate(req.Name); err != nil {
			log.Printf("[NETWORK] Warning: Failed to start network %s: %v", req.Name, err)
		} else {
			network.Active = true
		}
	}

	if err := h.repo.Create(ctx, network); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "network.failedToCreate"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(network))
}

func (h *VirtualNetworkHandler) Update(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	network, err := h.repo.FindByID(ctx, id)
	if err != nil {
		if err == repository.ErrNetworkNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "network.notFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "network.failedToGet"), err.Error()))
		return
	}

	var req UpdateNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
		return
	}

	if req.Name != "" && req.Name != network.Name {
		existing, _ := h.repo.FindByName(ctx, req.Name)
		if existing != nil {
			c.JSON(http.StatusBadRequest, errors.FailWithCode(errors.ErrCodeBadRequest, t(c, "network.nameExists")))
			return
		}
		network.Name = req.Name
	}

	if req.Description != "" {
		network.Description = req.Description
	}
	if req.BridgeName != "" {
		network.BridgeName = req.BridgeName
	}
	if req.Subnet != "" {
		network.Subnet = req.Subnet
	}
	if req.Gateway != "" {
		network.Gateway = req.Gateway
	}
	if req.DHCPStart != "" {
		network.DHCPStart = req.DHCPStart
	}
	if req.DHCPEnd != "" {
		network.DHCPEnd = req.DHCPEnd
	}
	if req.DHCPEnabled != nil {
		network.DHCPEnabled = *req.DHCPEnabled
	}
	if req.Autostart != nil {
		network.Autostart = *req.Autostart
	}

	network.XMLDef = h.generateNetworkXML(network)

	if err := h.repo.Update(ctx, network); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "network.failedToUpdate"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(network))
}

func (h *VirtualNetworkHandler) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	network, err := h.repo.FindByID(ctx, id)
	if err != nil {
		if err == repository.ErrNetworkNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "network.notFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "network.failedToGet"), err.Error()))
		return
	}

	if h.libvirt != nil {
		if network.Active {
			h.libvirt.NetworkDestroy(network.Name)
		}
		h.libvirt.NetworkUndefine(network.Name)
	}

	if err := h.repo.Delete(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "network.failedToDelete"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(nil))
}

func (h *VirtualNetworkHandler) Start(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	network, err := h.repo.FindByID(ctx, id)
	if err != nil {
		if err == repository.ErrNetworkNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "network.notFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "network.failedToGet"), err.Error()))
		return
	}

	if h.libvirt == nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithCode(errors.ErrCodeLibvirt, t(c, "libvirt_service_unavailable")))
		return
	}

	if err := h.libvirt.NetworkCreate(network.Name); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "network.failedToStart"), err.Error()))
		return
	}

	h.repo.SetActive(ctx, id, true)
	network.Active = true

	c.JSON(http.StatusOK, errors.Success(network))
}

func (h *VirtualNetworkHandler) Stop(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	network, err := h.repo.FindByID(ctx, id)
	if err != nil {
		if err == repository.ErrNetworkNotFound {
			c.JSON(http.StatusNotFound, errors.FailWithCode(errors.ErrCodeNotFound, t(c, "network.notFound")))
			return
		}
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "network.failedToGet"), err.Error()))
		return
	}

	if h.libvirt == nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithCode(errors.ErrCodeLibvirt, t(c, "libvirt_service_unavailable")))
		return
	}

	if err := h.libvirt.NetworkDestroy(network.Name); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeLibvirt, t(c, "network.failedToStop"), err.Error()))
		return
	}

	h.repo.SetActive(ctx, id, false)
	network.Active = false

	c.JSON(http.StatusOK, errors.Success(network))
}

func (h *VirtualNetworkHandler) generateNetworkXML(network *models.VirtualNetwork) string {
	xml := fmt.Sprintf(`<network>
  <name>%s</name>
  <forward mode='%s'/>`, network.Name, network.NetworkType)

	if network.BridgeName != "" {
		xml += fmt.Sprintf(`
  <bridge name='%s' stp='on' delay='0'/>`, network.BridgeName)
	}

	xml += fmt.Sprintf(`
  <ip address='%s' netmask='%s'>`, network.Gateway, network.Subnet)

	if network.DHCPEnabled && network.DHCPStart != "" && network.DHCPEnd != "" {
		xml += fmt.Sprintf(`
    <dhcp>
      <range start='%s' end='%s'/>
    </dhcp>`, network.DHCPStart, network.DHCPEnd)
	}

	xml += `
  </ip>
</network>`

	return xml
}
