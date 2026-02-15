package repository

import (
	"context"
	"errors"

	"vmmanager/internal/models"

	"gorm.io/gorm"
)

var ErrISONotFound = errors.New("iso not found")

type ISORepository struct {
	db *gorm.DB
}

func NewISORepository(db *gorm.DB) *ISORepository {
	return &ISORepository{db: db}
}

func (r *ISORepository) Create(ctx context.Context, iso *models.ISO) error {
	return r.db.WithContext(ctx).Create(iso).Error
}

func (r *ISORepository) FindByID(ctx context.Context, id string) (*models.ISO, error) {
	var iso models.ISO
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&iso).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrISONotFound
	}
	return &iso, err
}

func (r *ISORepository) FindByMD5(ctx context.Context, md5 string) (*models.ISO, error) {
	var iso models.ISO
	err := r.db.WithContext(ctx).Where("md5 = ?", md5).First(&iso).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrISONotFound
	}
	return &iso, err
}

func (r *ISORepository) Update(ctx context.Context, iso *models.ISO) error {
	return r.db.WithContext(ctx).Save(iso).Error
}

func (r *ISORepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.ISO{}).Error
}

func (r *ISORepository) List(ctx context.Context, offset, limit int) ([]models.ISO, int64, error) {
	var isos []models.ISO
	var total int64

	r.db.WithContext(ctx).Model(&models.ISO{}).Where("status = ?", "active").Count(&total)

	query := r.db.WithContext(ctx).
		Where("status = ?", "active").
		Offset(offset).
		Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&isos).Error

	return isos, total, err
}

func (r *ISORepository) ListByArchitecture(ctx context.Context, architecture string, offset, limit int) ([]models.ISO, int64, error) {
	var isos []models.ISO
	var total int64

	query := r.db.WithContext(ctx).Model(&models.ISO{}).
		Where("status = ? AND architecture = ?", "active", architecture)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&isos).Error

	return isos, total, err
}

func (r *ISORepository) Search(ctx context.Context, keyword string, offset, limit int) ([]models.ISO, int64, error) {
	var isos []models.ISO
	var total int64

	query := r.db.WithContext(ctx).Model(&models.ISO{}).
		Where("status = ? AND (name ILIKE ? OR description ILIKE ? OR os_type ILIKE ?)", 
			"active", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&isos).Error

	return isos, total, err
}

type ISOUploadRepository struct {
	db *gorm.DB
}

func NewISOUploadRepository(db *gorm.DB) *ISOUploadRepository {
	return &ISOUploadRepository{db: db}
}

func (r *ISOUploadRepository) Create(ctx context.Context, upload *models.ISOUpload) error {
	return r.db.WithContext(ctx).Create(upload).Error
}

func (r *ISOUploadRepository) FindByID(ctx context.Context, id string) (*models.ISOUpload, error) {
	var upload models.ISOUpload
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&upload).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("upload not found")
	}
	return &upload, err
}

func (r *ISOUploadRepository) Update(ctx context.Context, upload *models.ISOUpload) error {
	return r.db.WithContext(ctx).Save(upload).Error
}

func (r *ISOUploadRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.ISOUpload{}).Error
}
