package repository

import (
	"context"
	"errors"

	"vmmanager/internal/models"

	"gorm.io/gorm"
)

var ErrNetworkNotFound = errors.New("network not found")

type VirtualNetworkRepository struct {
	db *gorm.DB
}

func NewVirtualNetworkRepository(db *gorm.DB) *VirtualNetworkRepository {
	return &VirtualNetworkRepository{db: db}
}

func (r *VirtualNetworkRepository) Create(ctx context.Context, network *models.VirtualNetwork) error {
	return r.db.WithContext(ctx).Create(network).Error
}

func (r *VirtualNetworkRepository) FindByID(ctx context.Context, id string) (*models.VirtualNetwork, error) {
	var network models.VirtualNetwork
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&network).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNetworkNotFound
		}
		return nil, err
	}
	return &network, nil
}

func (r *VirtualNetworkRepository) FindByName(ctx context.Context, name string) (*models.VirtualNetwork, error) {
	var network models.VirtualNetwork
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&network).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNetworkNotFound
		}
		return nil, err
	}
	return &network, nil
}

func (r *VirtualNetworkRepository) Update(ctx context.Context, network *models.VirtualNetwork) error {
	return r.db.WithContext(ctx).Save(network).Error
}

func (r *VirtualNetworkRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.VirtualNetwork{}).Error
}

func (r *VirtualNetworkRepository) List(ctx context.Context, offset, limit int) ([]models.VirtualNetwork, int64, error) {
	var networks []models.VirtualNetwork
	var total int64

	r.db.WithContext(ctx).Model(&models.VirtualNetwork{}).Count(&total)

	query := r.db.WithContext(ctx).Order("created_at DESC")
	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}

	err := query.Find(&networks).Error
	return networks, total, err
}

func (r *VirtualNetworkRepository) ListActive(ctx context.Context) ([]models.VirtualNetwork, error) {
	var networks []models.VirtualNetwork
	err := r.db.WithContext(ctx).Where("active = ?", true).Find(&networks).Error
	return networks, err
}

func (r *VirtualNetworkRepository) SetActive(ctx context.Context, id string, active bool) error {
	return r.db.WithContext(ctx).Model(&models.VirtualNetwork{}).
		Where("id = ?", id).
		Update("active", active).Error
}
