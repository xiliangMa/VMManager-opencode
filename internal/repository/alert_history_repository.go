package repository

import (
	"context"
	"time"

	"vmmanager/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AlertHistoryRepository struct {
	db *gorm.DB
}

func NewAlertHistoryRepository(db *gorm.DB) *AlertHistoryRepository {
	return &AlertHistoryRepository{db: db}
}

func (r *AlertHistoryRepository) Create(ctx context.Context, history *models.AlertHistory) error {
	return r.db.WithContext(ctx).Create(history).Error
}

func (r *AlertHistoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.AlertHistory, error) {
	var history models.AlertHistory
	err := r.db.WithContext(ctx).First(&history, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &history, nil
}

func (r *AlertHistoryRepository) GetByVM(ctx context.Context, vmID string, limit int) ([]models.AlertHistory, error) {
	var histories []models.AlertHistory
	query := r.db.WithContext(ctx).Where("vm_id = ?", vmID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&histories).Error
	return histories, err
}

func (r *AlertHistoryRepository) GetActive(ctx context.Context) ([]models.AlertHistory, error) {
	var histories []models.AlertHistory
	err := r.db.WithContext(ctx).Where("status = ?", "triggered").Order("created_at DESC").Find(&histories).Error
	return histories, err
}

func (r *AlertHistoryRepository) Resolve(ctx context.Context, id string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.AlertHistory{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      "resolved",
			"resolved_at": now,
		}).Error
}

func (r *AlertHistoryRepository) GetStats(ctx context.Context) (total, critical, warning, info int64, err error) {
	err = r.db.WithContext(ctx).Model(&models.AlertHistory{}).
		Where("status = ?", "triggered").
		Count(&total).Error

	if err != nil {
		return
	}

	r.db.WithContext(ctx).Model(&models.AlertHistory{}).
		Where("status = ? AND severity = ?", "triggered", "critical").
		Count(&critical)

	r.db.WithContext(ctx).Model(&models.AlertHistory{}).
		Where("status = ? AND severity = ?", "triggered", "warning").
		Count(&warning)

	r.db.WithContext(ctx).Model(&models.AlertHistory{}).
		Where("status = ? AND severity = ?", "triggered", "info").
		Count(&info)

	return
}

func (r *AlertHistoryRepository) DeleteOld(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).
		Where("created_at < ? AND status = ?", before, "resolved").
		Delete(&models.AlertHistory{}).Error
}
