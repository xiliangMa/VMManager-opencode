package repository

import (
	"context"
	"time"

	"vmmanager/internal/models"

	"gorm.io/gorm"
)

type LoginHistoryRepository struct {
	db *gorm.DB
}

func NewLoginHistoryRepository(db *gorm.DB) *LoginHistoryRepository {
	return &LoginHistoryRepository{db: db}
}

func (r *LoginHistoryRepository) Create(ctx context.Context, log *models.LoginHistory) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *LoginHistoryRepository) FindByID(ctx context.Context, id string) (*models.LoginHistory, error) {
	var log models.LoginHistory
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *LoginHistoryRepository) List(ctx context.Context, offset, limit int) ([]models.LoginHistory, int64, error) {
	var logs []models.LoginHistory
	var total int64

	r.db.WithContext(ctx).Model(&models.LoginHistory{}).Count(&total)

	err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&logs).Error

	return logs, total, err
}

func (r *LoginHistoryRepository) ListByUser(ctx context.Context, userID string, offset, limit int) ([]models.LoginHistory, int64, error) {
	var logs []models.LoginHistory
	var total int64

	query := r.db.WithContext(ctx).Model(&models.LoginHistory{}).Where("user_id = ?", userID)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&logs).Error

	return logs, total, err
}

func (r *LoginHistoryRepository) ListByStatus(ctx context.Context, status string, offset, limit int) ([]models.LoginHistory, int64, error) {
	var logs []models.LoginHistory
	var total int64

	query := r.db.WithContext(ctx).Model(&models.LoginHistory{}).Where("status = ?", status)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&logs).Error

	return logs, total, err
}

func (r *LoginHistoryRepository) ListByTimeRange(ctx context.Context, start, end time.Time, offset, limit int) ([]models.LoginHistory, int64, error) {
	var logs []models.LoginHistory
	var total int64

	query := r.db.WithContext(ctx).Model(&models.LoginHistory{}).
		Where("created_at BETWEEN ? AND ?", start, end)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&logs).Error

	return logs, total, err
}

func (r *LoginHistoryRepository) ListByIPAddress(ctx context.Context, ipAddress string, offset, limit int) ([]models.LoginHistory, int64, error) {
	var logs []models.LoginHistory
	var total int64

	query := r.db.WithContext(ctx).Model(&models.LoginHistory{}).Where("ip_address = ?", ipAddress)
	query.Count(&total)

	err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&logs).Error

	return logs, total, err
}

func (r *LoginHistoryRepository) UpdateLogout(ctx context.Context, id string, logoutAt time.Time, sessionDuration int) error {
	return r.db.WithContext(ctx).
		Model(&models.LoginHistory{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"logout_at":       logoutAt,
			"session_duration": sessionDuration,
		}).Error
}

func (r *LoginHistoryRepository) GetLatestByUser(ctx context.Context, userID string) (*models.LoginHistory, error) {
	var log models.LoginHistory
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *LoginHistoryRepository) CountByUser(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.LoginHistory{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

func (r *LoginHistoryRepository) CountFailedLoginsByUser(ctx context.Context, userID string, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.LoginHistory{}).
		Where("user_id = ? AND status = ? AND created_at > ?", userID, "failed", since).
		Count(&count).Error
	return count, err
}

func (r *LoginHistoryRepository) DeleteOldLogs(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&models.LoginHistory{}).Error
}

type LoginHistoryWithUser struct {
	models.LoginHistory
	Username string `json:"username"`
	Email    string `json:"email"`
}

func (r *LoginHistoryRepository) ListWithUser(ctx context.Context, offset, limit int) ([]LoginHistoryWithUser, int64, error) {
	var logs []LoginHistoryWithUser
	var total int64

	r.db.WithContext(ctx).Model(&models.LoginHistory{}).Count(&total)

	err := r.db.WithContext(ctx).
		Table("login_histories").
		Select("login_histories.*, users.username, users.email").
		Joins("LEFT JOIN users ON users.id = login_histories.user_id").
		Order("login_histories.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&logs).Error

	return logs, total, err
}

func (r *LoginHistoryRepository) ListByUserWithUser(ctx context.Context, userID string, offset, limit int) ([]LoginHistoryWithUser, int64, error) {
	var logs []LoginHistoryWithUser
	var total int64

	r.db.WithContext(ctx).Model(&models.LoginHistory{}).Where("user_id = ?", userID).Count(&total)

	err := r.db.WithContext(ctx).
		Table("login_histories").
		Select("login_histories.*, users.username, users.email").
		Joins("LEFT JOIN users ON users.id = login_histories.user_id").
		Where("login_histories.user_id = ?", userID).
		Order("login_histories.created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(&logs).Error

	return logs, total, err
}
