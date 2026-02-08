package repository

import (
	"context"
	"time"

	"vmmanager/internal/models"

	"gorm.io/gorm"
)

type AuditLogRepository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, log *models.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *AuditLogRepository) FindByID(ctx context.Context, id string) (*models.AuditLog, error) {
	var log models.AuditLog
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *AuditLogRepository) List(ctx context.Context, offset, limit int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	r.db.WithContext(ctx).Model(&models.AuditLog{}).Count(&total)

	err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&logs).Error

	return logs, total, err
}

func (r *AuditLogRepository) ListByUser(ctx context.Context, userID string, offset, limit int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&models.AuditLog{}).Where("user_id = ?", userID)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&logs).Error

	return logs, total, err
}

func (r *AuditLogRepository) ListByAction(ctx context.Context, action string, offset, limit int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&models.AuditLog{}).Where("action = ?", action)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&logs).Error

	return logs, total, err
}

func (r *AuditLogRepository) ListByResource(ctx context.Context, resourceType, resourceID string, offset, limit int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&models.AuditLog{}).
		Where("resource_type = ? AND resource_id = ?", resourceType, resourceID)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&logs).Error

	return logs, total, err
}

func (r *AuditLogRepository) ListByTimeRange(ctx context.Context, start, end time.Time, offset, limit int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&models.AuditLog{}).
		Where("created_at BETWEEN ? AND ?", start, end)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&logs).Error

	return logs, total, err
}

func (r *AuditLogRepository) DeleteOldLogs(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&models.AuditLog{}).Error
}

func (r *AuditLogRepository) CountByAction(ctx context.Context, action string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.AuditLog{}).
		Where("action = ?", action).
		Count(&count).Error
	return count, err
}
