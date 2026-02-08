package repository

import (
	"gorm.io/gorm"
)

type Repositories struct {
	User           *UserRepository
	VM             *VMRepository
	Template       *TemplateRepository
	VMStats        *VMStatsRepository
	AuditLog       *AuditLogRepository
	TemplateUpload *TemplateUploadRepository
	AlertRule      *AlertRuleRepository
}

func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		User:           NewUserRepository(db),
		VM:             NewVMRepository(db),
		Template:       NewTemplateRepository(db),
		VMStats:        NewVMStatsRepository(db),
		AuditLog:       NewAuditLogRepository(db),
		TemplateUpload: NewTemplateUploadRepository(db),
		AlertRule:      NewAlertRuleRepository(db),
	}
}

type TxFunc func(tx *gorm.DB) error

func WithTransaction(db *gorm.DB, fn TxFunc) error {
	tx := db.Begin()
	if err := tx.Error; err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
