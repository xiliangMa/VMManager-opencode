package handlers

import (
	"net/http"

	"vmmanager/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TemplateHandler struct {
	db *gorm.DB
}

func NewTemplateHandler(db *gorm.DB) *TemplateHandler {
	return &TemplateHandler{db: db}
}

func (h *TemplateHandler) ListTemplates(c *gin.Context) {
	var templates []models.VMTemplate
	query := h.db.Where("is_active = ? AND is_public = ?", true, true)

	var page, pageSize int
	page = 1
	pageSize = 20

	if p := c.Query("page"); p != "" {
		_, _ = c.GetQuery("page")
	}

	if ps := c.Query("page_size"); ps != "" {
		_, _ = c.GetQuery("page_size")
	}

	var total int64
	query.Model(&models.VMTemplate{}).Count(&total)

	offset := (page - 1) * pageSize
	query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&templates)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    templates,
		"meta": gin.H{
			"page":        page,
			"per_page":    pageSize,
			"total":       total,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	id := c.Param("id")
	templateUUID, _ := uuid.Parse(id)

	var template models.VMTemplate
	if err := h.db.First(&template, "id = ?", templateUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "template not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    template,
	})
}

func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userUUID, _ := uuid.Parse(userID.(string))

	var req struct {
		Name         string `json:"name" binding:"required"`
		Description  string `json:"description"`
		OSType       string `json:"os_type" binding:"required"`
		OSVersion    string `json:"os_version"`
		Architecture string `json:"architecture"`
		Format       string `json:"format"`
		CPUMin       int    `json:"cpu_min"`
		CPUMax       int    `json:"cpu_max"`
		MemoryMin    int    `json:"memory_min"`
		MemoryMax    int    `json:"memory_max"`
		DiskMin      int    `json:"disk_min"`
		DiskMax      int    `json:"disk_max"`
		TemplatePath string `json:"template_path" binding:"required"`
		IconURL      string `json:"icon_url"`
		DiskSize     int64  `json:"disk_size" binding:"required"`
		IsPublic     bool   `json:"is_public"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 4001, "message": err.Error()})
		return
	}

	template := models.VMTemplate{
		Name:         req.Name,
		Description:  req.Description,
		OSType:       req.OSType,
		OSVersion:    req.OSVersion,
		Architecture: req.Architecture,
		Format:       req.Format,
		TemplatePath: req.TemplatePath,
		IconURL:      req.IconURL,
		DiskSize:     req.DiskSize,
		IsPublic:     req.IsPublic,
		CreatedBy:    &userUUID,
	}

	h.db.Create(&template)

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "success",
		"data":    template,
	})
}

func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	id := c.Param("id")
	templateUUID, _ := uuid.Parse(id)

	var template models.VMTemplate
	if err := h.db.First(&template, "id = ?", templateUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "template not found"})
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IconURL     string `json:"icon_url"`
		IsPublic    bool   `json:"is_public"`
		IsActive    bool   `json:"is_active"`
	}

	c.ShouldBindJSON(&req)

	if req.Name != "" {
		template.Name = req.Name
	}
	template.Description = req.Description
	template.IconURL = req.IconURL
	template.IsPublic = req.IsPublic
	template.IsActive = req.IsActive

	h.db.Save(&template)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    template,
	})
}

func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	id := c.Param("id")
	templateUUID, _ := uuid.Parse(id)

	var template models.VMTemplate
	if err := h.db.First(&template, "id = ?", templateUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "template not found"})
		return
	}

	h.db.Delete(&template)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
}

func (h *TemplateHandler) InitTemplateUpload(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userUUID, _ := uuid.Parse(userID.(string))

	var req struct {
		Name         string `json:"name" binding:"required"`
		Description  string `json:"description"`
		FileName     string `json:"file_name" binding:"required"`
		FileSize     int64  `json:"file_size" binding:"required"`
		Format       string `json:"format"`
		Architecture string `json:"architecture"`
	}

	c.ShouldBindJSON(&req)

	upload := models.TemplateUpload{
		Name:         req.Name,
		Description:  req.Description,
		FileName:     req.FileName,
		FileSize:     req.FileSize,
		Format:       req.Format,
		Architecture: req.Architecture,
		Status:       "uploading",
		UploadedBy:   &userUUID,
	}

	h.db.Create(&upload)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"upload_id":  upload.ID,
			"upload_url": "/api/v1/templates/upload/part",
		},
	})
}

func (h *TemplateHandler) UploadTemplatePart(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
}

func (h *TemplateHandler) CompleteTemplateUpload(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
}
