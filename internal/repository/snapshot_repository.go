package repository

import (
	"context"
	"errors"

	"vmmanager/internal/models"

	"gorm.io/gorm"
)

var ErrSnapshotNotFound = errors.New("snapshot not found")

type VMSnapshotRepository struct {
	db *gorm.DB
}

func NewVMSnapshotRepository(db *gorm.DB) *VMSnapshotRepository {
	return &VMSnapshotRepository{db: db}
}

func (r *VMSnapshotRepository) Create(ctx context.Context, snapshot *models.VMSnapshot) error {
	return r.db.WithContext(ctx).Create(snapshot).Error
}

func (r *VMSnapshotRepository) FindByID(ctx context.Context, id string) (*models.VMSnapshot, error) {
	var snapshot models.VMSnapshot
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&snapshot).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSnapshotNotFound
		}
		return nil, err
	}
	return &snapshot, nil
}

func (r *VMSnapshotRepository) FindByVMAndName(ctx context.Context, vmID string, name string) (*models.VMSnapshot, error) {
	var snapshot models.VMSnapshot
	err := r.db.WithContext(ctx).Where("vm_id = ? AND name = ?", vmID, name).First(&snapshot).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSnapshotNotFound
		}
		return nil, err
	}
	return &snapshot, nil
}

func (r *VMSnapshotRepository) Update(ctx context.Context, snapshot *models.VMSnapshot) error {
	return r.db.WithContext(ctx).Save(snapshot).Error
}

func (r *VMSnapshotRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.VMSnapshot{}).Error
}

func (r *VMSnapshotRepository) DeleteByVMAndName(ctx context.Context, vmID string, name string) error {
	return r.db.WithContext(ctx).Where("vm_id = ? AND name = ?", vmID, name).Delete(&models.VMSnapshot{}).Error
}

func (r *VMSnapshotRepository) ListByVM(ctx context.Context, vmID string) ([]models.VMSnapshot, error) {
	var snapshots []models.VMSnapshot
	err := r.db.WithContext(ctx).Where("vm_id = ?", vmID).Order("created_at DESC").Find(&snapshots).Error
	return snapshots, err
}

func (r *VMSnapshotRepository) CountByVM(ctx context.Context, vmID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.VMSnapshot{}).Where("vm_id = ?", vmID).Count(&count).Error
	return count, err
}

func (r *VMSnapshotRepository) SetCurrentSnapshot(ctx context.Context, vmID string, snapshotID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.VMSnapshot{}).
			Where("vm_id = ?", vmID).
			Update("is_current", false).Error; err != nil {
			return err
		}

		return tx.Model(&models.VMSnapshot{}).
			Where("id = ?", snapshotID).
			Update("is_current", true).Error
	})
}

func (r *VMSnapshotRepository) GetCurrentSnapshot(ctx context.Context, vmID string) (*models.VMSnapshot, error) {
	var snapshot models.VMSnapshot
	err := r.db.WithContext(ctx).Where("vm_id = ? AND is_current = ?", vmID, true).First(&snapshot).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSnapshotNotFound
		}
		return nil, err
	}
	return &snapshot, nil
}

func (r *VMSnapshotRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	return r.db.WithContext(ctx).Model(&models.VMSnapshot{}).
		Where("id = ?", id).
		Update("status", status).Error
}
