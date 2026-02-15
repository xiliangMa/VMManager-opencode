package repository

import (
	"context"
	"errors"
	"time"

	"vmmanager/internal/models"

	"gorm.io/gorm"
)

var ErrBackupNotFound = errors.New("backup not found")
var ErrScheduleNotFound = errors.New("backup schedule not found")

type VMBackupRepository struct {
	db *gorm.DB
}

func NewVMBackupRepository(db *gorm.DB) *VMBackupRepository {
	return &VMBackupRepository{db: db}
}

func (r *VMBackupRepository) Create(ctx context.Context, backup *models.VMBackup) error {
	return r.db.WithContext(ctx).Create(backup).Error
}

func (r *VMBackupRepository) FindByID(ctx context.Context, id string) (*models.VMBackup, error) {
	var backup models.VMBackup
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&backup).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBackupNotFound
		}
		return nil, err
	}
	return &backup, nil
}

func (r *VMBackupRepository) Update(ctx context.Context, backup *models.VMBackup) error {
	return r.db.WithContext(ctx).Save(backup).Error
}

func (r *VMBackupRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.VMBackup{}).Error
}

func (r *VMBackupRepository) ListByVM(ctx context.Context, vmID string, offset, limit int) ([]models.VMBackup, int64, error) {
	var backups []models.VMBackup
	var total int64

	r.db.WithContext(ctx).Model(&models.VMBackup{}).Where("vm_id = ?", vmID).Count(&total)

	query := r.db.WithContext(ctx).Where("vm_id = ?", vmID).Order("created_at DESC")
	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}

	err := query.Find(&backups).Error
	return backups, total, err
}

func (r *VMBackupRepository) ListPending(ctx context.Context) ([]models.VMBackup, error) {
	var backups []models.VMBackup
	err := r.db.WithContext(ctx).Where("status = ?", "pending").Find(&backups).Error
	return backups, err
}

func (r *VMBackupRepository) ListScheduled(ctx context.Context, before time.Time) ([]models.VMBackup, error) {
	var backups []models.VMBackup
	err := r.db.WithContext(ctx).
		Where("status = ? AND scheduled_at IS NOT NULL AND scheduled_at <= ?", "pending", before).
		Find(&backups).Error
	return backups, err
}

func (r *VMBackupRepository) ListExpired(ctx context.Context) ([]models.VMBackup, error) {
	var backups []models.VMBackup
	now := time.Now()
	err := r.db.WithContext(ctx).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at < ?", "completed", now).
		Find(&backups).Error
	return backups, err
}

func (r *VMBackupRepository) UpdateStatus(ctx context.Context, id string, status string, progress int, errorMsg string) error {
	updates := map[string]interface{}{
		"status":   status,
		"progress": progress,
	}
	if errorMsg != "" {
		updates["error_msg"] = errorMsg
	}
	if status == "running" {
		updates["started_at"] = time.Now()
	} else if status == "completed" || status == "failed" {
		updates["completed_at"] = time.Now()
	}
	return r.db.WithContext(ctx).Model(&models.VMBackup{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *VMBackupRepository) CountByVM(ctx context.Context, vmID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.VMBackup{}).Where("vm_id = ?", vmID).Count(&count).Error
	return count, err
}

type BackupScheduleRepository struct {
	db *gorm.DB
}

func NewBackupScheduleRepository(db *gorm.DB) *BackupScheduleRepository {
	return &BackupScheduleRepository{db: db}
}

func (r *BackupScheduleRepository) Create(ctx context.Context, schedule *models.BackupSchedule) error {
	return r.db.WithContext(ctx).Create(schedule).Error
}

func (r *BackupScheduleRepository) FindByID(ctx context.Context, id string) (*models.BackupSchedule, error) {
	var schedule models.BackupSchedule
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&schedule).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScheduleNotFound
		}
		return nil, err
	}
	return &schedule, nil
}

func (r *BackupScheduleRepository) Update(ctx context.Context, schedule *models.BackupSchedule) error {
	return r.db.WithContext(ctx).Save(schedule).Error
}

func (r *BackupScheduleRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.BackupSchedule{}).Error
}

func (r *BackupScheduleRepository) ListByVM(ctx context.Context, vmID string) ([]models.BackupSchedule, error) {
	var schedules []models.BackupSchedule
	err := r.db.WithContext(ctx).Where("vm_id = ?", vmID).Order("created_at DESC").Find(&schedules).Error
	return schedules, err
}

func (r *BackupScheduleRepository) ListEnabled(ctx context.Context) ([]models.BackupSchedule, error) {
	var schedules []models.BackupSchedule
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&schedules).Error
	return schedules, err
}

func (r *BackupScheduleRepository) UpdateLastRun(ctx context.Context, id string, lastRun, nextRun time.Time) error {
	return r.db.WithContext(ctx).Model(&models.BackupSchedule{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_run_at": lastRun,
			"next_run_at": nextRun,
		}).Error
}

func (r *BackupScheduleRepository) SetEnabled(ctx context.Context, id string, enabled bool) error {
	return r.db.WithContext(ctx).Model(&models.BackupSchedule{}).
		Where("id = ?", id).
		Update("enabled", enabled).Error
}
