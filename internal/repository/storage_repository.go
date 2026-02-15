package repository

import (
	"context"
	"errors"

	"vmmanager/internal/models"

	"gorm.io/gorm"
)

var ErrStoragePoolNotFound = errors.New("storage pool not found")

type StoragePoolRepository struct {
	db *gorm.DB
}

func NewStoragePoolRepository(db *gorm.DB) *StoragePoolRepository {
	return &StoragePoolRepository{db: db}
}

func (r *StoragePoolRepository) Create(ctx context.Context, pool *models.StoragePool) error {
	return r.db.WithContext(ctx).Create(pool).Error
}

func (r *StoragePoolRepository) FindByID(ctx context.Context, id string) (*models.StoragePool, error) {
	var pool models.StoragePool
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&pool).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStoragePoolNotFound
		}
		return nil, err
	}
	return &pool, nil
}

func (r *StoragePoolRepository) FindByName(ctx context.Context, name string) (*models.StoragePool, error) {
	var pool models.StoragePool
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&pool).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStoragePoolNotFound
		}
		return nil, err
	}
	return &pool, nil
}

func (r *StoragePoolRepository) Update(ctx context.Context, pool *models.StoragePool) error {
	return r.db.WithContext(ctx).Save(pool).Error
}

func (r *StoragePoolRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.StoragePool{}).Error
}

func (r *StoragePoolRepository) List(ctx context.Context, offset, limit int) ([]models.StoragePool, int64, error) {
	var pools []models.StoragePool
	var total int64

	r.db.WithContext(ctx).Model(&models.StoragePool{}).Count(&total)

	query := r.db.WithContext(ctx).Order("created_at DESC")
	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}

	err := query.Find(&pools).Error
	return pools, total, err
}

func (r *StoragePoolRepository) ListActive(ctx context.Context) ([]models.StoragePool, error) {
	var pools []models.StoragePool
	err := r.db.WithContext(ctx).Where("active = ?", true).Find(&pools).Error
	return pools, err
}

func (r *StoragePoolRepository) SetActive(ctx context.Context, id string, active bool) error {
	return r.db.WithContext(ctx).Model(&models.StoragePool{}).
		Where("id = ?", id).
		Update("active", active).Error
}

func (r *StoragePoolRepository) UpdateCapacity(ctx context.Context, id string, capacity, available, used int64) error {
	return r.db.WithContext(ctx).Model(&models.StoragePool{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"capacity":  capacity,
			"available": available,
			"used":      used,
		}).Error
}

type StorageVolumeRepository struct {
	db *gorm.DB
}

func NewStorageVolumeRepository(db *gorm.DB) *StorageVolumeRepository {
	return &StorageVolumeRepository{db: db}
}

func (r *StorageVolumeRepository) Create(ctx context.Context, volume *models.StorageVolume) error {
	return r.db.WithContext(ctx).Create(volume).Error
}

func (r *StorageVolumeRepository) FindByID(ctx context.Context, id string) (*models.StorageVolume, error) {
	var volume models.StorageVolume
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&volume).Error
	if err != nil {
		return nil, err
	}
	return &volume, nil
}

func (r *StorageVolumeRepository) FindByNameAndPool(ctx context.Context, poolID, name string) (*models.StorageVolume, error) {
	var volume models.StorageVolume
	err := r.db.WithContext(ctx).Where("pool_id = ? AND name = ?", poolID, name).First(&volume).Error
	if err != nil {
		return nil, err
	}
	return &volume, nil
}

func (r *StorageVolumeRepository) Update(ctx context.Context, volume *models.StorageVolume) error {
	return r.db.WithContext(ctx).Save(volume).Error
}

func (r *StorageVolumeRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.StorageVolume{}).Error
}

func (r *StorageVolumeRepository) ListByPool(ctx context.Context, poolID string, offset, limit int) ([]models.StorageVolume, int64, error) {
	var volumes []models.StorageVolume
	var total int64

	r.db.WithContext(ctx).Model(&models.StorageVolume{}).Where("pool_id = ?", poolID).Count(&total)

	query := r.db.WithContext(ctx).Where("pool_id = ?", poolID).Order("created_at DESC")
	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}

	err := query.Find(&volumes).Error
	return volumes, total, err
}

func (r *StorageVolumeRepository) ListByVM(ctx context.Context, vmID string) ([]models.StorageVolume, error) {
	var volumes []models.StorageVolume
	err := r.db.WithContext(ctx).Where("vm_id = ?", vmID).Find(&volumes).Error
	return volumes, err
}

func (r *StorageVolumeRepository) SetVMID(ctx context.Context, id string, vmID *string) error {
	return r.db.WithContext(ctx).Model(&models.StorageVolume{}).
		Where("id = ?", id).
		Update("vm_id", vmID).Error
}
