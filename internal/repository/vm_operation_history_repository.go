package repository

import (
	"context"
	"time"

	"vmmanager/internal/models"

	"gorm.io/gorm"
)

type VMOperationHistoryRepository struct {
	db *gorm.DB
}

func NewVMOperationHistoryRepository(db *gorm.DB) *VMOperationHistoryRepository {
	return &VMOperationHistoryRepository{db: db}
}

func (r *VMOperationHistoryRepository) Create(ctx context.Context, history *models.VMOperationHistory) error {
	return r.db.WithContext(ctx).Create(history).Error
}

func (r *VMOperationHistoryRepository) FindByID(ctx context.Context, id string) (*models.VMOperationHistory, error) {
	var history models.VMOperationHistory
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&history).Error
	if err != nil {
		return nil, err
	}
	return &history, nil
}

func (r *VMOperationHistoryRepository) List(ctx context.Context, offset, limit int) ([]models.VMOperationHistory, int64, error) {
	var histories []models.VMOperationHistory
	var total int64

	r.db.WithContext(ctx).Model(&models.VMOperationHistory{}).Count(&total)

	err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("started_at DESC").
		Find(&histories).Error

	return histories, total, err
}

func (r *VMOperationHistoryRepository) ListByVM(ctx context.Context, vmID string, offset, limit int) ([]models.VMOperationHistory, int64, error) {
	var histories []models.VMOperationHistory
	var total int64

	query := r.db.WithContext(ctx).Model(&models.VMOperationHistory{}).
		Where("vm_id = ?", vmID)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("started_at DESC").
		Find(&histories).Error

	return histories, total, err
}

func (r *VMOperationHistoryRepository) ListByUser(ctx context.Context, userID string, offset, limit int) ([]models.VMOperationHistory, int64, error) {
	var histories []models.VMOperationHistory
	var total int64

	query := r.db.WithContext(ctx).Model(&models.VMOperationHistory{}).
		Where("triggered_by = ?", userID)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("started_at DESC").
		Find(&histories).Error

	return histories, total, err
}

func (r *VMOperationHistoryRepository) ListByOperation(ctx context.Context, operation string, offset, limit int) ([]models.VMOperationHistory, int64, error) {
	var histories []models.VMOperationHistory
	var total int64

	query := r.db.WithContext(ctx).Model(&models.VMOperationHistory{}).
		Where("operation = ?", operation)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("started_at DESC").
		Find(&histories).Error

	return histories, total, err
}

func (r *VMOperationHistoryRepository) ListByStatus(ctx context.Context, status string, offset, limit int) ([]models.VMOperationHistory, int64, error) {
	var histories []models.VMOperationHistory
	var total int64

	query := r.db.WithContext(ctx).Model(&models.VMOperationHistory{}).
		Where("status = ?", status)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("started_at DESC").
		Find(&histories).Error

	return histories, total, err
}

func (r *VMOperationHistoryRepository) ListByTimeRange(ctx context.Context, start, end time.Time, offset, limit int) ([]models.VMOperationHistory, int64, error) {
	var histories []models.VMOperationHistory
	var total int64

	query := r.db.WithContext(ctx).Model(&models.VMOperationHistory{}).
		Where("started_at BETWEEN ? AND ?", start, end)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("started_at DESC").
		Find(&histories).Error

	return histories, total, err
}

func (r *VMOperationHistoryRepository) UpdateStatus(ctx context.Context, id string, status string, completedAt time.Time, duration int, responseData, errorMessage string) error {
	updates := map[string]interface{}{
		"status":       status,
		"completed_at": completedAt,
		"duration":     duration,
	}
	if responseData != "" {
		updates["response_data"] = responseData
	}
	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}

	return r.db.WithContext(ctx).
		Model(&models.VMOperationHistory{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *VMOperationHistoryRepository) GetLatestByVM(ctx context.Context, vmID string) (*models.VMOperationHistory, error) {
	var history models.VMOperationHistory
	err := r.db.WithContext(ctx).
		Where("vm_id = ?", vmID).
		Order("started_at DESC").
		First(&history).Error
	if err != nil {
		return nil, err
	}
	return &history, nil
}

func (r *VMOperationHistoryRepository) GetPendingOperations(ctx context.Context) ([]models.VMOperationHistory, error) {
	var histories []models.VMOperationHistory
	err := r.db.WithContext(ctx).
		Where("status = ?", "pending").
		Order("started_at ASC").
		Find(&histories).Error
	return histories, err
}

func (r *VMOperationHistoryRepository) CountByVM(ctx context.Context, vmID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.VMOperationHistory{}).
		Where("vm_id = ?", vmID).
		Count(&count).Error
	return count, err
}

func (r *VMOperationHistoryRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.VMOperationHistory{}).
		Where("status = ?", status).
		Count(&count).Error
	return count, err
}

func (r *VMOperationHistoryRepository) DeleteOldHistories(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).
		Where("started_at < ?", before).
		Delete(&models.VMOperationHistory{}).Error
}

type VMOperationHistoryWithUser struct {
	models.VMOperationHistory
	Username string `json:"username"`
	VMName   string `json:"vmName"`
}

func (r *VMOperationHistoryRepository) ListWithUser(ctx context.Context, offset, limit int) ([]VMOperationHistoryWithUser, int64, error) {
	var histories []VMOperationHistoryWithUser
	var total int64

	r.db.WithContext(ctx).Model(&models.VMOperationHistory{}).Count(&total)

	err := r.db.WithContext(ctx).
		Table("vm_operation_histories").
		Select("vm_operation_histories.*, users.username, virtual_machines.name as vm_name").
		Joins("LEFT JOIN users ON users.id = vm_operation_histories.triggered_by").
		Joins("LEFT JOIN virtual_machines ON virtual_machines.id = vm_operation_histories.vm_id").
		Order("vm_operation_histories.started_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&histories).Error

	return histories, total, err
}

func (r *VMOperationHistoryRepository) ListByVMWithUser(ctx context.Context, vmID string, offset, limit int) ([]VMOperationHistoryWithUser, int64, error) {
	var histories []VMOperationHistoryWithUser
	var total int64

	r.db.WithContext(ctx).Model(&models.VMOperationHistory{}).
		Where("vm_id = ?", vmID).
		Count(&total)

	err := r.db.WithContext(ctx).
		Table("vm_operation_histories").
		Select("vm_operation_histories.*, users.username, virtual_machines.name as vm_name").
		Joins("LEFT JOIN users ON users.id = vm_operation_histories.triggered_by").
		Joins("LEFT JOIN virtual_machines ON virtual_machines.id = vm_operation_histories.vm_id").
		Where("vm_operation_histories.vm_id = ?", vmID).
		Order("vm_operation_histories.started_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&histories).Error

	return histories, total, err
}
