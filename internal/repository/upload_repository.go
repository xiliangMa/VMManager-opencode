package repository

import (
	"context"
	"errors"

	"vmmanager/internal/models"

	"gorm.io/gorm"
)

var ErrUploadNotFound = errors.New("upload not found")

type TemplateUploadRepository struct {
	db *gorm.DB
}

func NewTemplateUploadRepository(db *gorm.DB) *TemplateUploadRepository {
	return &TemplateUploadRepository{db: db}
}

func (r *TemplateUploadRepository) Create(ctx context.Context, upload *models.TemplateUpload) error {
	return r.db.WithContext(ctx).Create(upload).Error
}

func (r *TemplateUploadRepository) FindByID(ctx context.Context, id string) (*models.TemplateUpload, error) {
	var upload models.TemplateUpload
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&upload).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUploadNotFound
	}
	return &upload, err
}

func (r *TemplateUploadRepository) Update(ctx context.Context, upload *models.TemplateUpload) error {
	return r.db.WithContext(ctx).Save(upload).Error
}

func (r *TemplateUploadRepository) UpdateProgress(ctx context.Context, id string, progress int) error {
	return r.db.WithContext(ctx).
		Model(&models.TemplateUpload{}).
		Where("id = ?", id).
		Update("progress", progress).Error
}

func (r *TemplateUploadRepository) UpdateStatus(ctx context.Context, id, status string) error {
	return r.db.WithContext(ctx).
		Model(&models.TemplateUpload{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *TemplateUploadRepository) UpdateStatusWithError(ctx context.Context, id, status, errorMsg string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if errorMsg != "" {
		updates["error_message"] = errorMsg
	}
	return r.db.WithContext(ctx).
		Model(&models.TemplateUpload{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *TemplateUploadRepository) Complete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&models.TemplateUpload{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      "completed",
			"progress":    100,
			"completedAt": gorm.Expr("NOW()"),
		}).Error
}

func (r *TemplateUploadRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.TemplateUpload{}).Error
}

func (r *TemplateUploadRepository) List(ctx context.Context, offset, limit int) ([]models.TemplateUpload, int64, error) {
	var uploads []models.TemplateUpload
	var total int64

	r.db.WithContext(ctx).Model(&models.TemplateUpload{}).Count(&total)

	err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&uploads).Error

	return uploads, total, err
}

func (r *TemplateUploadRepository) ListByUser(ctx context.Context, userID string, offset, limit int) ([]models.TemplateUpload, int64, error) {
	var uploads []models.TemplateUpload
	var total int64

	query := r.db.WithContext(ctx).Model(&models.TemplateUpload{}).Where("uploaded_by = ?", userID)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&uploads).Error

	return uploads, total, err
}

func (r *TemplateUploadRepository) ListByStatus(ctx context.Context, status string) ([]models.TemplateUpload, error) {
	var uploads []models.TemplateUpload
	err := r.db.WithContext(ctx).
		Where("status = ?", status).
		Find(&uploads).Error
	return uploads, err
}

func (r *TemplateUploadRepository) CountByUser(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.TemplateUpload{}).
		Where("uploaded_by = ?", userID).
		Count(&count).Error
	return count, err
}
