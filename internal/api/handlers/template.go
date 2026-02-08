package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"vmmanager/internal/api/errors"
	"vmmanager/internal/models"
	"vmmanager/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TemplateHandler struct {
	templateRepo *repository.TemplateRepository
	uploadRepo   *repository.TemplateUploadRepository
	config       *UploadConfig
}

type UploadConfig struct {
	UploadPath     string
	MaxPartSize    int64
	AllowedFormats map[string]bool
}

func NewTemplateHandler(
	templateRepo *repository.TemplateRepository,
	uploadRepo *repository.TemplateUploadRepository,
) *TemplateHandler {
	return &TemplateHandler{
		templateRepo: templateRepo,
		uploadRepo:   uploadRepo,
		config: &UploadConfig{
			UploadPath:  "./uploads",
			MaxPartSize: 100 * 1024 * 1024,
			AllowedFormats: map[string]bool{
				"qcow2":  true,
				"vmdk":   true,
				"ova":    true,
				"raw":    true,
				"qcow":   true,
				"vdi":    true,
				"qcow2c": true,
			},
		},
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
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to fetch templates", err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.SuccessWithMeta(templates, gin.H{
		"page":        page,
		"per_page":    pageSize,
		"total":       total,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	}))
}

func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	template, err := h.templateRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeTemplateNotFound, "template not found", id))
		return
	}

	c.JSON(http.StatusOK, errors.Success(template))
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
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, "validation error", err.Error()))
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
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to create template", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, errors.Success(template))
}

func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	template, err := h.templateRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeTemplateNotFound, "template not found", id))
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
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, "validation error", err.Error()))
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
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to update template", err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(template))
}

func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	_, err := h.templateRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeTemplateNotFound, "template not found", id))
		return
	}

	if err := h.templateRepo.Delete(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to delete template", err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(nil))
}

type InitUploadRequest struct {
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	FileName     string `json:"file_name" binding:"required"`
	FileSize     int64  `json:"file_size" binding:"required,min=1"`
	Format       string `json:"format" binding:"required"`
	Architecture string `json:"architecture"`
	ChunkSize    int64  `json:"chunk_size" binding:"required,min=1,max=104857600"`
}

type InitUploadResponse struct {
	UploadID    string `json:"upload_id"`
	UploadPath  string `json:"upload_path"`
	ChunkSize   int64  `json:"chunk_size"`
	TotalChunks int    `json:"total_chunks"`
}

type UploadPartRequest struct {
	ChunkIndex  int `form:"chunk_index" binding:"required,min=0"`
	TotalChunks int `form:"total_chunks" binding:"required,min=1"`
}

type CompleteUploadRequest struct {
	TotalChunks int    `json:"total_chunks" binding:"required,min=1"`
	Checksum    string `json:"checksum"`
}

func (h *TemplateHandler) InitTemplateUpload(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userUUID, _ := uuid.Parse(userID.(string))

	var req InitUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, "validation error", err.Error()))
		return
	}

	if !h.config.AllowedFormats[strings.ToLower(req.Format)] {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, "invalid format", fmt.Sprintf("allowed: %v", getAllowedFormats())))
		return
	}

	uploadUUID := uuid.New()
	uploadDir := filepath.Join(h.config.UploadPath, "templates", uploadUUID.String())
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, "failed to create upload directory", err.Error()))
		return
	}

	totalChunks := int((req.FileSize + req.ChunkSize - 1) / req.ChunkSize)

	upload := &models.TemplateUpload{
		ID:           uploadUUID,
		Name:         req.Name,
		Description:  req.Description,
		FileName:     req.FileName,
		FileSize:     req.FileSize,
		Format:       strings.ToLower(req.Format),
		Architecture: req.Architecture,
		UploadPath:   uploadDir,
		TempPath:     "",
		Status:       "uploading",
		Progress:     0,
		UploadedBy:   &userUUID,
	}

	ctx := c.Request.Context()
	if err := h.uploadRepo.Create(ctx, upload); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to create upload record", err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(InitUploadResponse{
		UploadID:    uploadUUID.String(),
		UploadPath:  fmt.Sprintf("/api/v1/templates/upload/part?upload_id=%s", uploadUUID.String()),
		ChunkSize:   req.ChunkSize,
		TotalChunks: totalChunks,
	}))
}

func getAllowedFormats() []string {
	return []string{"qcow2", "vmdk", "ova", "raw", "qcow", "vdi", "qcow2c"}
}

func (h *TemplateHandler) UploadTemplatePart(c *gin.Context) {
	uploadID := c.Query("upload_id")
	if uploadID == "" {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, "upload_id is required", ""))
		return
	}

	var req UploadPartRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, "validation error", err.Error()))
		return
	}

	ctx := c.Request.Context()
	upload, err := h.uploadRepo.FindByID(ctx, uploadID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeNotFound, "upload not found", uploadID))
		return
	}

	if upload.Status != "uploading" {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeConflict, "upload is not in uploading status", upload.Status))
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, "failed to read file", err.Error()))
		return
	}
	defer file.Close()

	if header.Size > h.config.MaxPartSize {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, "file chunk too large", fmt.Sprintf("max: %d", h.config.MaxPartSize)))
		return
	}

	chunkPath := filepath.Join(upload.UploadPath, fmt.Sprintf("chunk_%06d", req.ChunkIndex))
	dst, err := os.Create(chunkPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, "failed to create chunk file", err.Error()))
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, "failed to write chunk", err.Error()))
		return
	}

	progress := int((int64(req.ChunkIndex+1) * 100) / int64(req.TotalChunks))
	h.uploadRepo.UpdateProgress(ctx, uploadID, progress)

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"upload_id":   uploadID,
		"chunk_index": req.ChunkIndex,
		"chunk_size":  written,
		"progress":    progress,
	}))
}

func (h *TemplateHandler) CompleteTemplateUpload(c *gin.Context) {
	uploadID := c.Param("id")
	ctx := c.Request.Context()

	upload, err := h.uploadRepo.FindByID(ctx, uploadID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeNotFound, "upload not found", uploadID))
		return
	}

	if upload.Status != "uploading" {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeConflict, "upload is not in uploading status", upload.Status))
		return
	}

	var req CompleteUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, "validation error", err.Error()))
		return
	}

	uploadDir := upload.UploadPath
	finalPath := filepath.Join(uploadDir, upload.FileName)

	if err := h.mergeChunks(uploadDir, finalPath, req.TotalChunks); err != nil {
		h.uploadRepo.UpdateStatusWithError(ctx, uploadID, "failed", err.Error())
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, "failed to merge chunks", err.Error()))
		return
	}

	fileInfo, err := os.Stat(finalPath)
	if err != nil || fileInfo.Size() != upload.FileSize {
		h.uploadRepo.UpdateStatusWithError(ctx, uploadID, "failed", "file size mismatch")
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeConflict, "file size verification failed", fmt.Sprintf("expected: %d, got: %d", upload.FileSize, fileInfo.Size())))
		return
	}

	if err := h.uploadRepo.Complete(ctx, uploadID); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to complete upload", err.Error()))
		return
	}

	template := &models.VMTemplate{
		Name:         upload.Name,
		Description:  upload.Description,
		OSType:       "linux",
		OSVersion:    "",
		Architecture: upload.Architecture,
		Format:       upload.Format,
		CPUMin:       1,
		CPUMax:       4,
		MemoryMin:    1024,
		MemoryMax:    8192,
		DiskMin:      20,
		DiskMax:      500,
		TemplatePath: finalPath,
		IconURL:      "",
		DiskSize:     upload.FileSize,
		IsPublic:     true,
		IsActive:     true,
		Downloads:    0,
		CreatedBy:    upload.UploadedBy,
	}

	if err := h.templateRepo.Create(ctx, template); err != nil {
		h.uploadRepo.UpdateStatusWithError(ctx, uploadID, "failed", err.Error())
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to create template", err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"upload_id":   uploadID,
		"template_id": template.ID,
		"file_path":   finalPath,
		"file_size":   upload.FileSize,
	}))
}

func (h *TemplateHandler) mergeChunks(uploadDir, finalPath string, totalChunks int) error {
	finalFile, err := os.Create(finalPath)
	if err != nil {
		return fmt.Errorf("failed to create final file: %w", err)
	}
	defer finalFile.Close()

	for i := 0; i < totalChunks; i++ {
		chunkPath := filepath.Join(uploadDir, fmt.Sprintf("chunk_%06d", i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return fmt.Errorf("failed to open chunk %d: %w", i, err)
		}

		if _, err := io.Copy(finalFile, chunkFile); err != nil {
			chunkFile.Close()
			return fmt.Errorf("failed to write chunk %d: %w", i, err)
		}
		chunkFile.Close()

		os.Remove(chunkPath)
	}

	return nil
}

func (h *TemplateHandler) AbortUpload(c *gin.Context) {
	uploadID := c.Param("id")
	ctx := c.Request.Context()

	upload, err := h.uploadRepo.FindByID(ctx, uploadID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeNotFound, "upload not found", uploadID))
		return
	}

	if err := h.uploadRepo.UpdateStatus(ctx, uploadID, "aborted"); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, "failed to abort upload", err.Error()))
		return
	}

	os.RemoveAll(upload.UploadPath)

	c.JSON(http.StatusOK, errors.Success(nil))
}

func (h *TemplateHandler) GetUploadStatus(c *gin.Context) {
	uploadID := c.Param("id")
	ctx := c.Request.Context()

	upload, err := h.uploadRepo.FindByID(ctx, uploadID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeNotFound, "upload not found", uploadID))
		return
	}

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"upload_id":    upload.ID,
		"name":         upload.Name,
		"file_name":    upload.FileName,
		"file_size":    upload.FileSize,
		"format":       upload.Format,
		"status":       upload.Status,
		"progress":     upload.Progress,
		"error":        upload.ErrorMessage,
		"created_at":   upload.CreatedAt,
		"completed_at": upload.CompletedAt,
	}))
}

func ValidateTemplateFormat(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".qcow2":
		return "qcow2", nil
	case ".vmdk":
		return "vmdk", nil
	case ".ova", ".ovf":
		return "ova", nil
	case ".raw":
		return "raw", nil
	case ".qcow":
		return "qcow", nil
	case ".vdi":
		return "vdi", nil
	default:
		return "", fmt.Errorf("unsupported format: %s", ext)
	}
}

func CalculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := uuid.New().String()[:32]
	return hash, nil
}
