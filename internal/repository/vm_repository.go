package repository

import (
	"context"
	"errors"
	"net"

	"vmmanager/internal/models"

	"gorm.io/gorm"
)

var ErrVMNotFound = errors.New("virtual machine not found")

type VMRepository struct {
	db *gorm.DB
}

func NewVMRepository(db *gorm.DB) *VMRepository {
	return &VMRepository{db: db}
}

func (r *VMRepository) Create(ctx context.Context, vm *models.VirtualMachine) error {
	return r.db.WithContext(ctx).Create(vm).Error
}

func (r *VMRepository) FindByID(ctx context.Context, id string) (*models.VirtualMachine, error) {
	var vm models.VirtualMachine
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&vm).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrVMNotFound
	}
	return &vm, err
}

func (r *VMRepository) FindByName(ctx context.Context, name string) (*models.VirtualMachine, error) {
	var vm models.VirtualMachine
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&vm).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrVMNotFound
	}
	return &vm, err
}

func (r *VMRepository) FindByOwner(ctx context.Context, ownerID string, offset, limit int) ([]models.VirtualMachine, int64, error) {
	var vms []models.VirtualMachine
	var total int64

	query := r.db.WithContext(ctx).Model(&models.VirtualMachine{}).Where("owner_id = ?", ownerID)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&vms).Error

	return vms, total, err
}

func (r *VMRepository) Update(ctx context.Context, vm *models.VirtualMachine) error {
	return r.db.WithContext(ctx).Save(vm).Error
}

func (r *VMRepository) UpdateStatus(ctx context.Context, id, status string) error {
	return r.db.WithContext(ctx).
		Model(&models.VirtualMachine{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *VMRepository) UpdateIPAddress(ctx context.Context, id string, ip net.IP) error {
	return r.db.WithContext(ctx).
		Model(&models.VirtualMachine{}).
		Where("id = ?", id).
		Update("ip_address", ip).Error
}

func (r *VMRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.VirtualMachine{}).Error
}

func (r *VMRepository) List(ctx context.Context, offset, limit int) ([]models.VirtualMachine, int64, error) {
	var vms []models.VirtualMachine
	var total int64

	r.db.WithContext(ctx).Model(&models.VirtualMachine{}).Count(&total)

	err := r.db.WithContext(ctx).
		Preload("Owner").
		Preload("Template").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&vms).Error

	return vms, total, err
}

func (r *VMRepository) ListByStatus(ctx context.Context, status string) ([]models.VirtualMachine, error) {
	var vms []models.VirtualMachine
	err := r.db.WithContext(ctx).
		Where("status = ?", status).
		Find(&vms).Error
	return vms, err
}

func (r *VMRepository) CountByOwner(ctx context.Context, ownerID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.VirtualMachine{}).
		Where("owner_id = ?", ownerID).
		Count(&count).Error
	return count, err
}

func (r *VMRepository) GetVNCPort(ctx context.Context) (int, error) {
	var port int
	err := r.db.WithContext(ctx).
		Model(&models.VirtualMachine{}).
		Order("vnc_port DESC").
		First(&port, "vnc_port > 0").Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return 5900, nil
	}
	if err != nil {
		return 0, err
	}
	return port + 1, nil
}

func (r *VMRepository) FindByMAC(ctx context.Context, mac string) (*models.VirtualMachine, error) {
	var vm models.VirtualMachine
	err := r.db.WithContext(ctx).Where("mac_address = ?", mac).First(&vm).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrVMNotFound
	}
	return &vm, err
}

func (r *VMRepository) FindByLibvirtUUID(ctx context.Context, uuid string) (*models.VirtualMachine, error) {
	var vm models.VirtualMachine
	err := r.db.WithContext(ctx).Where("libvirt_domain_uuid = ?", uuid).First(&vm).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrVMNotFound
	}
	return &vm, err
}
