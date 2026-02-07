package repository

import (
	"context"
	"errors"
	"time"

	"vmmanager/internal/models"

	"gorm.io/gorm"
)

var ErrStatsNotFound = errors.New("stats not found")

type VMStatsRepository struct {
	db *gorm.DB
}

func NewVMStatsRepository(db *gorm.DB) *VMStatsRepository {
	return &VMStatsRepository{db: db}
}

func (r *VMStatsRepository) Create(ctx context.Context, stats *models.VMStats) error {
	return r.db.WithContext(ctx).Create(stats).Error
}

func (r *VMStatsRepository) CreateBatch(ctx context.Context, statsList []models.VMStats) error {
	if len(statsList) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(statsList, 100).Error
}

func (r *VMStatsRepository) FindByVMID(ctx context.Context, vmID string, limit int) ([]models.VMStats, error) {
	var stats []models.VMStats
	err := r.db.WithContext(ctx).
		Where("vm_id = ?", vmID).
		Order("collected_at DESC").
		Limit(limit).
		Find(&stats).Error
	return stats, err
}

func (r *VMStatsRepository) FindByVMIDAndTimeRange(ctx context.Context, vmID string, start, end time.Time) ([]models.VMStats, error) {
	var stats []models.VMStats
	err := r.db.WithContext(ctx).
		Where("vm_id = ? AND collected_at BETWEEN ? AND ?", vmID, start, end).
		Order("collected_at ASC").
		Find(&stats).Error
	return stats, err
}

func (r *VMStatsRepository) DeleteOldStats(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).
		Where("collected_at < ?", before).
		Delete(&models.VMStats{}).Error
}

func (r *VMStatsRepository) GetLatestByVMID(ctx context.Context, vmID string) (*models.VMStats, error) {
	var stats models.VMStats
	err := r.db.WithContext(ctx).
		Where("vm_id = ?", vmID).
		Order("collected_at DESC").
		First(&stats).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrStatsNotFound
	}
	return &stats, err
}

func (r *VMStatsRepository) GetAggregatedStats(ctx context.Context, vmID string, start, end time.Time) (*models.VMStats, error) {
	var stats models.VMStats
	err := r.db.WithContext(ctx).
		Select("AVG(cpu_usage) as cpu_usage, AVG(memory_usage) as memory_usage, SUM(disk_read) as disk_read, SUM(disk_write) as disk_write, SUM(network_rx) as network_rx, SUM(network_tx) as network_tx").
		Where("vm_id = ? AND collected_at BETWEEN ? AND ?", vmID, start, end).
		First(&stats).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrStatsNotFound
	}
	return &stats, err
}
