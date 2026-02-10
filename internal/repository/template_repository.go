package repository

import (
	"context"
	"errors"

	"vmmanager/internal/models"

	"gorm.io/gorm"
)

var ErrTemplateNotFound = errors.New("template not found")

type TemplateRepository struct {
	db *gorm.DB
}

func NewTemplateRepository(db *gorm.DB) *TemplateRepository {
	return &TemplateRepository{db: db}
}

func (r *TemplateRepository) Create(ctx context.Context, tmpl *models.VMTemplate) error {
	return r.db.WithContext(ctx).Create(tmpl).Error
}

func (r *TemplateRepository) FindByID(ctx context.Context, id string) (*models.VMTemplate, error) {
	var tmpl models.VMTemplate
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&tmpl).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTemplateNotFound
	}
	return &tmpl, err
}

func (r *TemplateRepository) FindByName(ctx context.Context, name string) (*models.VMTemplate, error) {
	var tmpl models.VMTemplate
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&tmpl).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTemplateNotFound
	}
	return &tmpl, err
}

func (r *TemplateRepository) Update(ctx context.Context, tmpl *models.VMTemplate) error {
	return r.db.WithContext(ctx).Save(tmpl).Error
}

func (r *TemplateRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.VMTemplate{}).Error
}

func (r *TemplateRepository) List(ctx context.Context, offset, limit int) ([]models.VMTemplate, int64, error) {
	var templates []models.VMTemplate
	var total int64

	r.db.WithContext(ctx).Model(&models.VMTemplate{}).Count(&total)

	query := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Offset(offset).
		Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&templates).Error

	return templates, total, err
}

func (r *TemplateRepository) ListPublic(ctx context.Context, offset, limit int) ([]models.VMTemplate, int64, error) {
	var templates []models.VMTemplate
	var total int64

	query := r.db.WithContext(ctx).Model(&models.VMTemplate{}).
		Where("is_active = ? AND is_public = ?", true, true)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&templates).Error

	return templates, total, err
}

func (r *TemplateRepository) ListByUser(ctx context.Context, userID string, offset, limit int) ([]models.VMTemplate, int64, error) {
	var templates []models.VMTemplate
	var total int64

	query := r.db.WithContext(ctx).Model(&models.VMTemplate{}).
		Where("is_active = ? AND (is_public = ? OR created_by = ?)", true, true, userID)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&templates).Error

	return templates, total, err
}

func (r *TemplateRepository) ListUserTemplates(ctx context.Context, userID string, offset, limit int) ([]models.VMTemplate, int64, error) {
	var templates []models.VMTemplate
	var total int64

	query := r.db.WithContext(ctx).Model(&models.VMTemplate{}).
		Where("is_active = ? OR created_by = ?", true, userID)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&templates).Error

	return templates, total, err
}

func (r *TemplateRepository) ListByOS(ctx context.Context, osType string, offset, limit int) ([]models.VMTemplate, int64, error) {
	var templates []models.VMTemplate
	var total int64

	query := r.db.WithContext(ctx).Model(&models.VMTemplate{}).
		Where("is_active = ? AND os_type = ?", true, osType)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&templates).Error

	return templates, total, err
}

func (r *TemplateRepository) IncrementDownloads(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&models.VMTemplate{}).
		Where("id = ?", id).
		UpdateColumn("downloads", gorm.Expr("downloads + 1")).Error
}

func (r *TemplateRepository) CountByUser(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.VMTemplate{}).
		Where("created_by = ? AND is_active = ?", userID, true).
		Count(&count).Error
	return count, err
}
