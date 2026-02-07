package handlers

import (
	"net/http"

	"vmmanager/internal/models"
	"vmmanager/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TemplateHandler struct {
	templateRepo *repository.TemplateRepository
	uploadRepo   *repository.TemplateUploadRepository
}

func NewTemplateHandler(
	templateRepo *repository.TemplateRepository,
	uploadRepo *repository.TemplateUploadRepository,
) *TemplateHandler {
	return &TemplateHandler{
		templateRepo: templateRepo,
		uploadRepo:   uploadRepo,
	}
}

func (h *TemplateHandler) ListTemplates(c *gin.Context) {
	ctx := c.Request.Context()

	page := 1
	pageSize := 20

	if p := c.Query("page"); p != "" {
		_, _ = c.GetQuery("page")
	}

	if ps := c.Query("page_size"); ps != "" {
		_, _ = c.GetQuery("page_size")
	}

	templates, total, err := h.templateRepo.ListPublic(ctx, (page-1)*pageSize, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "failed to fetch templates"})
		return
	}

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
	ctx := c.Request.Context()

	template, err := h.templateRepo.FindByID(ctx, id)
	if err != nil {
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

	ctx := c.Request.Context()

	template := &models.VMTemplate{
		Name:         req.Name,
		Description:  req.Description,
		OSType:       req.OSType,
		OSVersion:    req.OSVersion,
		Architecture: req.Architecture,
		Format:       req.Format,
		CPUMin:       req.CPUMin,
		CPUMax:       req.CPUMax,
		MemoryMin:    req.MemoryMin,
		MemoryMax:    req.MemoryMax,
		DiskMin:      req.DiskMin,
		DiskMax:      req.DiskMax,
		TemplatePath: req.TemplatePath,
		IconURL:      req.IconURL,
		DiskSize:     req.DiskSize,
		IsPublic:     req.IsPublic,
		CreatedBy:    &userUUID,
	}

	if err := h.templateRepo.Create(ctx, template); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "failed to create template"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "success",
		"data":    template,
	})
}

func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	template, err := h.templateRepo.FindByID(ctx, id)
	if err != nil {
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

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 4001, "message": err.Error()})
		return
	}

	if req.Name != "" {
		template.Name = req.Name
	}
	template.Description = req.Description
	template.IconURL = req.IconURL
	template.IsPublic = req.IsPublic
	template.IsActive = req.IsActive

	if err := h.templateRepo.Update(ctx, template); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "failed to update template"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    template,
	})
}

func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	_, err := h.templateRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 4004, "message": "template not found"})
		return
	}

	if err := h.templateRepo.Delete(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "failed to delete template"})
		return
	}

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

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 4001, "message": err.Error()})
		return
	}

	ctx := c.Request.Context()

	upload := &models.TemplateUpload{
		Name:         req.Name,
		Description:  req.Description,
		FileName:     req.FileName,
		FileSize:     req.FileSize,
		Format:       req.Format,
		Architecture: req.Architecture,
		Status:       "uploading",
		UploadedBy:   &userUUID,
	}

	if err := h.uploadRepo.Create(ctx, upload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5001, "message": "failed to create upload record"})
		return
	}

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
