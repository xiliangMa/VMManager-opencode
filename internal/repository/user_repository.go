package repository

import (
	"context"
	"errors"

	"vmmanager/internal/models"

	"gorm.io/gorm"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserAlreadyExists = errors.New("user already exists")

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	return &user, err
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	return &user, err
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	return &user, err
}

func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.User{}).Error
}

func (r *UserRepository) List(ctx context.Context, offset, limit int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	r.db.WithContext(ctx).Model(&models.User{}).Count(&total)

	query := r.db.WithContext(ctx).
		Offset(offset).
		Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&users).Error

	return users, total, err
}

func (r *UserRepository) UpdateLastLogin(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", id).
		Update("last_login_at", gorm.Expr("NOW()")).Error
}

func (r *UserRepository) UpdateQuota(ctx context.Context, id string, cpu, memory, disk, vmCount int) error {
	updates := map[string]interface{}{}
	if cpu > 0 {
		updates["quota_cpu"] = cpu
	}
	if memory > 0 {
		updates["quota_memory"] = memory
	}
	if disk > 0 {
		updates["quota_disk"] = disk
	}
	if vmCount > 0 {
		updates["quota_vm_count"] = vmCount
	}

	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", id).
		Updates(updates).Error
}

type ResourceUsage struct {
	VMCount     int   `json:"vmCount"`
	CPUUsed     int   `json:"cpuUsed"`
	MemoryUsed  int   `json:"memoryUsed"`
	DiskUsed    int64 `json:"diskUsed"`
}

func (r *UserRepository) GetResourceUsage(ctx context.Context, userID string) (*ResourceUsage, error) {
	var usage ResourceUsage

	err := r.db.WithContext(ctx).
		Model(&models.VirtualMachine{}).
		Where("owner_id = ?", userID).
		Select("COUNT(*) as vm_count, COALESCE(SUM(cpu_allocated), 0) as cpu_used, COALESCE(SUM(memory_allocated), 0) as memory_used").
		Scan(&usage).Error
	if err != nil {
		return nil, err
	}

	var diskUsed int64
	err = r.db.WithContext(ctx).
		Model(&models.VirtualMachine{}).
		Where("owner_id = ?", userID).
		Select("COALESCE(SUM(disk_allocated), 0)").
		Scan(&diskUsed).Error
	if err != nil {
		return nil, err
	}
	usage.DiskUsed = diskUsed

	return &usage, nil
}
