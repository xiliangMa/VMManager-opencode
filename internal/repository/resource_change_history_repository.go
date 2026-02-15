package repository

import (
	"context"
	"time"

	"vmmanager/internal/models"

	"gorm.io/gorm"
)

type ResourceChangeHistoryRepository struct {
	db *gorm.DB
}

func NewResourceChangeHistoryRepository(db *gorm.DB) *ResourceChangeHistoryRepository {
	return &ResourceChangeHistoryRepository{db: db}
}

func (r *ResourceChangeHistoryRepository) Create(ctx context.Context, history *models.ResourceChangeHistory) error {
	return r.db.WithContext(ctx).Create(history).Error
}

func (r *ResourceChangeHistoryRepository) FindByID(ctx context.Context, id string) (*models.ResourceChangeHistory, error) {
	var history models.ResourceChangeHistory
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&history).Error
	if err != nil {
		return nil, err
	}
	return &history, nil
}

func (r *ResourceChangeHistoryRepository) List(ctx context.Context, offset, limit int) ([]models.ResourceChangeHistory, int64, error) {
	var histories []models.ResourceChangeHistory
	var total int64

	r.db.WithContext(ctx).Model(&models.ResourceChangeHistory{}).Count(&total)

	err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&histories).Error

	return histories, total, err
}

func (r *ResourceChangeHistoryRepository) ListByResource(ctx context.Context, resourceType, resourceID string, offset, limit int) ([]models.ResourceChangeHistory, int64, error) {
	var histories []models.ResourceChangeHistory
	var total int64

	query := r.db.WithContext(ctx).Model(&models.ResourceChangeHistory{}).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&histories).Error

	return histories, total, err
}

func (r *ResourceChangeHistoryRepository) ListByType(ctx context.Context, resourceType string, offset, limit int) ([]models.ResourceChangeHistory, int64, error) {
	var histories []models.ResourceChangeHistory
	var total int64

	query := r.db.WithContext(ctx).Model(&models.ResourceChangeHistory{}).
		Where("resource_type = ?", resourceType)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&histories).Error

	return histories, total, err
}

func (r *ResourceChangeHistoryRepository) ListByUser(ctx context.Context, userID string, offset, limit int) ([]models.ResourceChangeHistory, int64, error) {
	var histories []models.ResourceChangeHistory
	var total int64

	query := r.db.WithContext(ctx).Model(&models.ResourceChangeHistory{}).
		Where("changed_by = ?", userID)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&histories).Error

	return histories, total, err
}

func (r *ResourceChangeHistoryRepository) ListByAction(ctx context.Context, action string, offset, limit int) ([]models.ResourceChangeHistory, int64, error) {
	var histories []models.ResourceChangeHistory
	var total int64

	query := r.db.WithContext(ctx).Model(&models.ResourceChangeHistory{}).
		Where("action = ?", action)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&histories).Error

	return histories, total, err
}

func (r *ResourceChangeHistoryRepository) ListByTimeRange(ctx context.Context, start, end time.Time, offset, limit int) ([]models.ResourceChangeHistory, int64, error) {
	var histories []models.ResourceChangeHistory
	var total int64

	query := r.db.WithContext(ctx).Model(&models.ResourceChangeHistory{}).
		Where("created_at BETWEEN ? AND ?", start, end)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&histories).Error

	return histories, total, err
}

func (r *ResourceChangeHistoryRepository) DeleteOldHistories(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&models.ResourceChangeHistory{}).Error
}

type ResourceChangeHistoryWithUser struct {
	models.ResourceChangeHistory
	Username string `json:"username"`
}

func (r *ResourceChangeHistoryRepository) ListWithUser(ctx context.Context, offset, limit int) ([]ResourceChangeHistoryWithUser, int64, error) {
	var histories []ResourceChangeHistoryWithUser
	var total int64

	r.db.WithContext(ctx).Model(&models.ResourceChangeHistory{}).Count(&total)

	err := r.db.WithContext(ctx).
		Table("resource_change_histories").
		Select("resource_change_histories.*, users.username").
		Joins("LEFT JOIN users ON users.id = resource_change_histories.changed_by").
		Order("resource_change_histories.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&histories).Error

	return histories, total, err
}

func (r *ResourceChangeHistoryRepository) ListByResourceWithUser(ctx context.Context, resourceType, resourceID string, offset, limit int) ([]ResourceChangeHistoryWithUser, int64, error) {
	var histories []ResourceChangeHistoryWithUser
	var total int64

	r.db.WithContext(ctx).Model(&models.ResourceChangeHistory{}).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
		Count(&total)

	err := r.db.WithContext(ctx).
		Table("resource_change_histories").
		Select("resource_change_histories.*, users.username").
		Joins("LEFT JOIN users ON users.id = resource_change_histories.changed_by").
		Where("resource_change_histories.resource_type = ? AND resource_change_histories.resource_id = ?", resourceType, resourceID).
		Order("resource_change_histories.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&histories).Error

	return histories, total, err
}
