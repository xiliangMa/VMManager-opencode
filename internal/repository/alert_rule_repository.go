package repository

import (
	"context"
	"errors"

	"vmmanager/internal/models"

	"gorm.io/gorm"
)

var ErrAlertRuleNotFound = errors.New("alert rule not found")

type AlertRuleRepository struct {
	db *gorm.DB
}

func NewAlertRuleRepository(db *gorm.DB) *AlertRuleRepository {
	return &AlertRuleRepository{db: db}
}

func (r *AlertRuleRepository) Create(ctx context.Context, rule *models.AlertRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *AlertRuleRepository) FindByID(ctx context.Context, id string) (*models.AlertRule, error) {
	var rule models.AlertRule
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&rule).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAlertRuleNotFound
	}
	return &rule, err
}

func (r *AlertRuleRepository) Update(ctx context.Context, rule *models.AlertRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

func (r *AlertRuleRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.AlertRule{}).Error
}

func (r *AlertRuleRepository) List(ctx context.Context, offset, limit int) ([]models.AlertRule, int64, error) {
	var rules []models.AlertRule
	var total int64

	r.db.WithContext(ctx).Model(&models.AlertRule{}).Count(&total)

	err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&rules).Error

	return rules, total, err
}

func (r *AlertRuleRepository) ListEnabled(ctx context.Context) ([]models.AlertRule, error) {
	var rules []models.AlertRule
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Find(&rules).Error
	return rules, err
}

func (r *AlertRuleRepository) CountBySeverity(ctx context.Context, severity string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.AlertRule{}).
		Where("severity = ?", severity).
		Count(&count).Error
	return count, err
}

func (r *AlertRuleRepository) FindByVMAndMetric(ctx context.Context, vmID, metric string) ([]models.AlertRule, error) {
	var rules []models.AlertRule

	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Where("is_global = ? OR (vm_ids::text LIKE ?)", true, "%"+vmID+"%").
		Where("metric = ?", metric).
		Find(&rules).Error

	return rules, err
}
