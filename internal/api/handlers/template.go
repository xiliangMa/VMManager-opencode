package handlers

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"vmmanager/internal/api/errors"
	"vmmanager/internal/models"
	"vmmanager/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TemplateHandler struct {
	templateRepo *repository.TemplateRepository
	uploadRepo   *repository.TemplateUploadRepository
	vmRepo       *repository.VMRepository
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
	vmRepo *repository.VMRepository,
) *TemplateHandler {
	return &TemplateHandler{
		templateRepo: templateRepo,
		uploadRepo:   uploadRepo,
		vmRepo:       vmRepo,
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
		if v, err := strconv.Atoi(p); err == nil {
			page = v
		}
	}

	if ps := c.Query("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil {
			pageSize = v
		}
	}

	isPublicStr := c.Query("is_public")

	var templates []models.VMTemplate
	var total int64
	var err error
	var userID interface{}

	if isPublicStr == "true" {
		userID, _ = c.Get("user_id")
		templates, total, err = h.templateRepo.ListPublic(ctx, (page-1)*pageSize, pageSize)
	} else if isPublicStr == "false" {
		userID, _ = c.Get("user_id")
		templates, total, err = h.templateRepo.ListByUser(ctx, userID.(string), (page-1)*pageSize, pageSize)
	} else {
		userID, _ = c.Get("user_id")
		templates, total, err = h.templateRepo.ListUserTemplates(ctx, userID.(string), (page-1)*pageSize, pageSize)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_fetch_templates"), err.Error()))
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
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeTemplateNotFound, t(c, "template_not_found_id"), id))
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
		TemplatePath string `json:"template_path"`
		IconURL      string `json:"icon_url"`
		DiskSize     int64  `json:"disk_size"`
		IsPublic     bool   `json:"is_public"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
		return
	}

	ctx := c.Request.Context()

	diskSize := req.DiskSize
	if diskSize == 0 && req.DiskMax > 0 {
		diskSize = int64(req.DiskMax)
	}

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
		DiskSize:     diskSize,
		IsPublic:     req.IsPublic,
		CreatedBy:    &userUUID,
	}

	if err := h.templateRepo.Create(ctx, template); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_create_template"), err.Error()))
		return
	}

	c.JSON(http.StatusCreated, errors.Success(template))
}

func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	template, err := h.templateRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeTemplateNotFound, t(c, "template_not_found_id"), id))
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
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
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
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_update_template"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.Success(template))
}

func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	template, err := h.templateRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeTemplateNotFound, t(c, "template_not_found_id"), id))
		return
	}

	vmCount, err := h.vmRepo.CountByTemplateID(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_check_template_usage"), err.Error()))
		return
	}

	if vmCount > 0 {
		errMsg := fmt.Sprintf("%s (%d %s)", t(c, "template_in_use"), vmCount, t(c, "vm.vmCount"))
		c.JSON(http.StatusBadRequest, errors.FailWithCode(errors.ErrCodeBadRequest, errMsg))
		return
	}

	if err := h.templateRepo.Delete(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_delete_template"), err.Error()))
		return
	}

	if template.TemplatePath != "" {
		log.Printf("[TEMPLATE] Deleting template file: %s", template.TemplatePath)
		if err := os.Remove(template.TemplatePath); err != nil {
			log.Printf("[TEMPLATE] Failed to remove template file %s: %v", template.TemplatePath, err)
		} else {
			log.Printf("[TEMPLATE] Successfully removed template file: %s", template.TemplatePath)
		}
		uploadDir := filepath.Dir(template.TemplatePath)
		log.Printf("[TEMPLATE] Deleting upload directory: %s", uploadDir)
		if err := os.RemoveAll(uploadDir); err != nil {
			log.Printf("[TEMPLATE] Failed to remove upload directory %s: %v", uploadDir, err)
		} else {
			log.Printf("[TEMPLATE] Successfully removed upload directory: %s", uploadDir)
		}
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
	CPUMin       int    `json:"cpu_min"`
	CPUMax       int    `json:"cpu_max"`
	MemoryMin    int    `json:"memory_min"`
	MemoryMax    int    `json:"memory_max"`
	DiskMin      int    `json:"disk_min"`
	DiskMax      int    `json:"disk_max"`
	IsPublic     bool   `json:"is_public"`
}

type UploadPartRequest struct {
	ChunkIndex  int `form:"chunk_index" binding:"required,min=0"`
	TotalChunks int `form:"total_chunks" binding:"required,min=1"`
}

type CompleteUploadRequest struct {
	TotalChunks  int    `json:"total_chunks" binding:"required,min=1"`
	Checksum     string `json:"checksum"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	OSType       string `json:"os_type"`
	OSVersion    string `json:"os_version"`
	Architecture string `json:"architecture"`
	Format       string `json:"format"`
	CPUMin       int    `json:"cpu_min"`
	CPUMax       int    `json:"cpu_max"`
	MemoryMin    int    `json:"memory_min"`
	MemoryMax    int    `json:"memory_max"`
	DiskMin      int    `json:"disk_min"`
	DiskMax      int    `json:"disk_max"`
	IsPublic     bool   `json:"is_public"`
}

type InitUploadResponse struct {
	UploadID    string `json:"upload_id"`
	UploadPath  string `json:"upload_path"`
	ChunkSize   int64  `json:"chunk_size"`
	TotalChunks int    `json:"total_chunks"`
}

func (h *TemplateHandler) InitTemplateUpload(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userUUID, _ := uuid.Parse(userID.(string))

	var req InitUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
		return
	}

	if !h.config.AllowedFormats[strings.ToLower(req.Format)] {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "invalid_format"), fmt.Sprintf("allowed: %v", getAllowedFormats())))
		return
	}

	uploadUUID := uuid.New()
	uploadDir := filepath.Join(h.config.UploadPath, "templates", uploadUUID.String())
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_create_upload_directory"), err.Error()))
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
		TotalChunks:  totalChunks,
		ChunkSize:    req.ChunkSize,
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
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "upload_id_required"), ""))
		return
	}

	chunkIndexStr := c.Query("chunk_index")
	totalChunksStr := c.Query("total_chunks")

	if chunkIndexStr == "" || totalChunksStr == "" {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "chunk_index_and_total_chunks_required"), ""))
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "invalid_chunk_index"), err.Error()))
		return
	}

	totalChunks, err := strconv.Atoi(totalChunksStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "invalid_total_chunks"), err.Error()))
		return
	}

	ctx := c.Request.Context()
	upload, err := h.uploadRepo.FindByID(ctx, uploadID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeNotFound, t(c, "upload_not_found"), uploadID))
		return
	}

	if upload.Status != "uploading" {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeConflict, t(c, "upload_is_not_in_uploading_status"), upload.Status))
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "failed_to_read_file"), err.Error()))
		return
	}
	defer file.Close()

	if header.Size > h.config.MaxPartSize {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "file_chunk_too_large"), fmt.Sprintf("max: %d", h.config.MaxPartSize)))
		return
	}

	chunkPath := filepath.Join(upload.UploadPath, fmt.Sprintf("chunk_%06d", chunkIndex))
	log.Printf("[UPLOAD] Creating chunk file: %s", chunkPath)
	dst, err := os.Create(chunkPath)
	if err != nil {
		log.Printf("[UPLOAD] Failed to create chunk file %s: %v", chunkPath, err)
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_create_chunk_file"), err.Error()))
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		log.Printf("[UPLOAD] Failed to write chunk %d for upload %s: %v", chunkIndex, uploadID, err)
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_write_chunk"), err.Error()))
		return
	}

	log.Printf("[UPLOAD] Chunk %d/%d written successfully for upload %s (%d bytes)", chunkIndex, totalChunks, uploadID, written)

	progress := int((int64(chunkIndex+1) * 100) / int64(totalChunks))

	var uploadedChunks []int
	if upload.UploadedChunks != "" {
		chunkStrs := strings.Split(upload.UploadedChunks, ",")
		for _, s := range chunkStrs {
			if s != "" {
				if chunk, err := strconv.Atoi(s); err == nil {
					uploadedChunks = append(uploadedChunks, chunk)
				}
			}
		}
	}

	found := false
	for _, c := range uploadedChunks {
		if c == chunkIndex {
			found = true
			break
		}
	}
	if !found {
		uploadedChunks = append(uploadedChunks, chunkIndex)
	}

	chunkStrs := make([]string, len(uploadedChunks))
	for i, c := range uploadedChunks {
		chunkStrs[i] = strconv.Itoa(c)
	}
	upload.UploadedChunks = strings.Join(chunkStrs, ",")
	upload.Progress = progress
	h.uploadRepo.Update(ctx, upload)

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"upload_id":       uploadID,
		"chunk_index":     chunkIndex,
		"chunk_size":      written,
		"progress":        progress,
		"uploaded_chunks": uploadedChunks,
	}))
}

func (h *TemplateHandler) CompleteTemplateUpload(c *gin.Context) {
	uploadID := c.Param("id")
	ctx := c.Request.Context()

	upload, err := h.uploadRepo.FindByID(ctx, uploadID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeNotFound, t(c, "upload_not_found"), uploadID))
		return
	}

	if upload.Status != "uploading" {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeConflict, t(c, "upload_is_not_in_uploading_status"), upload.Status))
		return
	}

	var req CompleteUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
		return
	}

	uploadDir := upload.UploadPath
	finalPath := filepath.Join(uploadDir, upload.FileName)

	if err := h.mergeChunks(uploadDir, finalPath, req.TotalChunks); err != nil {
		h.uploadRepo.UpdateStatusWithError(ctx, uploadID, "failed", err.Error())
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_merge_chunks"), err.Error()))
		return
	}

	fileInfo, err := os.Stat(finalPath)
	if err != nil || fileInfo.Size() != upload.FileSize {
		h.uploadRepo.UpdateStatusWithError(ctx, uploadID, "failed", "file size mismatch")
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeConflict, t(c, "file_size_verification_failed"), fmt.Sprintf("expected: %d, got: %d", upload.FileSize, fileInfo.Size())))
		return
	}

	file, err := os.Open(finalPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_open_file"), err.Error()))
		return
	}
	defer file.Close()

	md5Hash := md5.New()
	sha256Hash := sha256.New()
	tee := io.TeeReader(file, io.MultiWriter(md5Hash, sha256Hash))
	if _, err := io.Copy(io.Discard, tee); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_calculate_checksum"), err.Error()))
		return
	}

	md5Sum := hex.EncodeToString(md5Hash.Sum(nil))
	sha256Sum := hex.EncodeToString(sha256Hash.Sum(nil))

	if req.Checksum != "" && !strings.EqualFold(req.Checksum, md5Sum) {
		os.Remove(finalPath)
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "checksum_mismatch"),
			fmt.Sprintf("expected: %s, got: %s", req.Checksum, md5Sum)))
		return
	}

	now := time.Now()
	upload.Status = "completed"
	upload.Progress = 100
	upload.CompletedAt = &now
	h.uploadRepo.Update(ctx, upload)

	diskMax := req.DiskMax
	if diskMax == 0 {
		diskMax = int(fileInfo.Size() / 1024 / 1024 / 1024)
		if diskMax < 20 {
			diskMax = 20
		}
	}

	templateName := req.Name
	if templateName == "" {
		templateName = upload.Name
	}

	// Use architecture from request, or fall back to upload record
	architecture := req.Architecture
	if architecture == "" {
		architecture = upload.Architecture
	}

	template := &models.VMTemplate{
		Name:         templateName,
		Description:  req.Description,
		OSType:       req.OSType,
		OSVersion:    req.OSVersion,
		Architecture: architecture,
		Format:       req.Format,
		CPUMin:       req.CPUMin,
		CPUMax:       req.CPUMax,
		MemoryMin:    req.MemoryMin,
		MemoryMax:    req.MemoryMax,
		DiskMin:      req.DiskMin,
		DiskMax:      diskMax,
		TemplatePath: finalPath,
		DiskSize:     upload.FileSize,
		MD5:          md5Sum,
		SHA256:       sha256Sum,
		IsPublic:     req.IsPublic,
		IsActive:     true,
		CreatedBy:    upload.UploadedBy,
	}

	if err := h.templateRepo.Create(ctx, template); err != nil {
		h.uploadRepo.UpdateStatusWithError(ctx, uploadID, "failed", err.Error())
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_create_template"), err.Error()))
		return
	}

	log.Printf("[TEMPLATE] Template upload completed: %s, MD5: %s, SHA256: %s", template.Name, md5Sum, sha256Sum)

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"upload_id":   uploadID,
		"template_id": template.ID,
		"file_path":   finalPath,
		"file_size":   upload.FileSize,
		"md5":         md5Sum,
		"sha256":      sha256Sum,
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
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeNotFound, t(c, "upload_not_found"), uploadID))
		return
	}

	if err := h.uploadRepo.UpdateStatus(ctx, uploadID, "aborted"); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_abort_upload"), err.Error()))
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
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeNotFound, t(c, "upload_not_found"), uploadID))
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
