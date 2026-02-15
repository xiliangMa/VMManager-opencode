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

type ISOHandler struct {
	isoRepo    *repository.ISORepository
	uploadRepo *repository.ISOUploadRepository
	config     *ISOUploadConfig
}

type ISOUploadConfig struct {
	UploadPath     string
	MaxPartSize    int64
	AllowedFormats map[string]bool
}

func NewISOHandler(
	isoRepo *repository.ISORepository,
	uploadRepo *repository.ISOUploadRepository,
) *ISOHandler {
	return &ISOHandler{
		isoRepo:    isoRepo,
		uploadRepo: uploadRepo,
		config: &ISOUploadConfig{
			UploadPath:  "./uploads/isos",
			MaxPartSize: 100 * 1024 * 1024,
			AllowedFormats: map[string]bool{
				"iso": true,
			},
		},
	}
}

func (h *ISOHandler) ListISOs(c *gin.Context) {
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

	keyword := c.Query("search")
	architecture := c.Query("architecture")

	var isos []models.ISO
	var total int64
	var err error

	if keyword != "" {
		isos, total, err = h.isoRepo.Search(ctx, keyword, (page-1)*pageSize, pageSize)
	} else if architecture != "" {
		isos, total, err = h.isoRepo.ListByArchitecture(ctx, architecture, (page-1)*pageSize, pageSize)
	} else {
		isos, total, err = h.isoRepo.List(ctx, (page-1)*pageSize, pageSize)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_fetch_isos"), err.Error()))
		return
	}

	c.JSON(http.StatusOK, errors.SuccessWithMeta(isos, gin.H{
		"page":        page,
		"per_page":    pageSize,
		"total":       total,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	}))
}

func (h *ISOHandler) GetISO(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	iso, err := h.isoRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeNotFound, t(c, "iso_not_found_id"), id))
		return
	}

	c.JSON(http.StatusOK, errors.Success(iso))
}

func (h *ISOHandler) DeleteISO(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	iso, err := h.isoRepo.FindByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeNotFound, t(c, "iso_not_found_id"), id))
		return
	}

	if err := h.isoRepo.Delete(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_delete_iso"), err.Error()))
		return
	}

	if iso.ISOPath != "" {
		log.Printf("[ISO] Deleting ISO file: %s", iso.ISOPath)
		if err := os.Remove(iso.ISOPath); err != nil {
			log.Printf("[ISO] Failed to remove ISO file %s: %v", iso.ISOPath, err)
		} else {
			log.Printf("[ISO] Successfully removed ISO file: %s", iso.ISOPath)
		}
		uploadDir := filepath.Dir(iso.ISOPath)
		log.Printf("[ISO] Deleting upload directory: %s", uploadDir)
		if err := os.RemoveAll(uploadDir); err != nil {
			log.Printf("[ISO] Failed to remove upload directory %s: %v", uploadDir, err)
		} else {
			log.Printf("[ISO] Successfully removed upload directory: %s", uploadDir)
		}
	}

	c.JSON(http.StatusOK, errors.Success(nil))
}

type InitISOUploadRequest struct {
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	FileName     string `json:"file_name" binding:"required"`
	FileSize     int64  `json:"file_size" binding:"required,min=1"`
	Architecture string `json:"architecture"`
	OSType       string `json:"os_type"`
	OSVersion    string `json:"os_version"`
	ChunkSize    int64  `json:"chunk_size" binding:"required,min=1,max=104857600"`
}

type InitISOUploadResponse struct {
	UploadID    string `json:"upload_id"`
	UploadPath  string `json:"upload_path"`
	ChunkSize   int64  `json:"chunk_size"`
	TotalChunks int    `json:"total_chunks"`
}

func (h *ISOHandler) InitISOUpload(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userUUID, _ := uuid.Parse(userID.(string))

	var req InitISOUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
		return
	}

	ext := strings.ToLower(filepath.Ext(req.FileName))
	if ext != ".iso" {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "invalid_format"), "only .iso files are allowed"))
		return
	}

	uploadUUID := uuid.New()
	uploadDir := filepath.Join(h.config.UploadPath, uploadUUID.String())
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_create_upload_directory"), err.Error()))
		return
	}

	totalChunks := int((req.FileSize + req.ChunkSize - 1) / req.ChunkSize)

	upload := &models.ISOUpload{
		ID:           uploadUUID,
		Name:         req.Name,
		Description:  req.Description,
		FileName:     req.FileName,
		FileSize:     req.FileSize,
		Architecture: req.Architecture,
		OSType:       req.OSType,
		OSVersion:    req.OSVersion,
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

	c.JSON(http.StatusOK, errors.Success(InitISOUploadResponse{
		UploadID:    uploadUUID.String(),
		UploadPath:  fmt.Sprintf("/api/v1/isos/upload/part?upload_id=%s", uploadUUID.String()),
		ChunkSize:   req.ChunkSize,
		TotalChunks: totalChunks,
	}))
}

func (h *ISOHandler) UploadISOPart(c *gin.Context) {
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

	tempFilePath := filepath.Join(upload.UploadPath, fmt.Sprintf("%s.part", upload.FileName))
	tempFile, err := os.OpenFile(tempFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_create_temp_file"), err.Error()))
		return
	}
	defer tempFile.Close()

	if _, err := tempFile.Seek(int64(chunkIndex)*h.config.MaxPartSize, 0); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_seek_file"), err.Error()))
		return
	}

	if _, err := io.Copy(tempFile, file); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_write_file"), err.Error()))
		return
	}

	progress := int(float64(chunkIndex+1) / float64(totalChunks) * 100)
	upload.Progress = progress

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

	h.uploadRepo.Update(ctx, upload)

	log.Printf("[ISO] Uploaded chunk %d/%d for upload %s, file: %s, size: %d",
		chunkIndex+1, totalChunks, uploadID, header.Filename, header.Size)

	c.JSON(http.StatusOK, errors.Success(gin.H{
		"chunk_index":     chunkIndex,
		"total_chunks":    totalChunks,
		"progress":        progress,
		"uploaded_chunks": uploadedChunks,
	}))
}

type CompleteISOUploadRequest struct {
	TotalChunks int    `json:"total_chunks" binding:"required,min=1"`
	Checksum    string `json:"checksum"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OSType      string `json:"os_type"`
	OSVersion   string `json:"os_version"`
}

func (h *ISOHandler) CompleteISOUpload(c *gin.Context) {
	uploadID := c.Query("upload_id")
	if uploadID == "" {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "upload_id_required"), ""))
		return
	}

	var req CompleteISOUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeValidation, t(c, "validation_error"), err.Error()))
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

	tempFilePath := filepath.Join(upload.UploadPath, fmt.Sprintf("%s.part", upload.FileName))
	finalFilePath := filepath.Join(upload.UploadPath, upload.FileName)

	if err := os.Rename(tempFilePath, finalFilePath); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeInternalError, t(c, "failed_to_rename_file"), err.Error()))
		return
	}

	file, err := os.Open(finalFilePath)
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
		os.Remove(finalFilePath)
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "checksum_mismatch"),
			fmt.Sprintf("expected: %s, got: %s", req.Checksum, md5Sum)))
		return
	}

	name := upload.Name
	if req.Name != "" {
		name = req.Name
	}
	description := upload.Description
	if req.Description != "" {
		description = req.Description
	}
	osType := upload.OSType
	if req.OSType != "" {
		osType = req.OSType
	}
	osVersion := upload.OSVersion
	if req.OSVersion != "" {
		osVersion = req.OSVersion
	}

	iso := &models.ISO{
		Name:         name,
		Description:  description,
		FileName:     upload.FileName,
		FileSize:     upload.FileSize,
		ISOPath:      finalFilePath,
		MD5:          md5Sum,
		SHA256:       sha256Sum,
		OSType:       osType,
		OSVersion:    osVersion,
		Architecture: upload.Architecture,
		Status:       "active",
		UploadedBy:   upload.UploadedBy,
	}

	if err := h.isoRepo.Create(ctx, iso); err != nil {
		c.JSON(http.StatusInternalServerError, errors.FailWithDetails(errors.ErrCodeDatabase, t(c, "failed_to_create_iso"), err.Error()))
		return
	}

	now := time.Now()
	upload.Status = "completed"
	upload.Progress = 100
	upload.CompletedAt = &now
	h.uploadRepo.Update(ctx, upload)

	log.Printf("[ISO] ISO upload completed: %s, MD5: %s, SHA256: %s", iso.Name, md5Sum, sha256Sum)

	c.JSON(http.StatusOK, errors.Success(iso))
}

func (h *ISOHandler) GetUploadStatus(c *gin.Context) {
	uploadID := c.Query("upload_id")
	if uploadID == "" {
		c.JSON(http.StatusBadRequest, errors.FailWithDetails(errors.ErrCodeBadRequest, t(c, "upload_id_required"), ""))
		return
	}

	ctx := c.Request.Context()
	upload, err := h.uploadRepo.FindByID(ctx, uploadID)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.FailWithDetails(errors.ErrCodeNotFound, t(c, "upload_not_found"), uploadID))
		return
	}

	c.JSON(http.StatusOK, errors.Success(upload))
}
