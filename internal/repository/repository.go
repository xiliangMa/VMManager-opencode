package repository

import (
	"gorm.io/gorm"
)

type Repositories struct {
	DB             *gorm.DB
	User           *UserRepository
	VM             *VMRepository
	Template       *TemplateRepository
	VMStats        *VMStatsRepository
	AuditLog       *AuditLogRepository
	TemplateUpload *TemplateUploadRepository
	AlertRule      *AlertRuleRepository
	AlertHistory   *AlertHistoryRepository
	ISO            *ISORepository
	ISOUpload      *ISOUploadRepository
	VirtualNetwork *VirtualNetworkRepository
	StoragePool    *StoragePoolRepository
	StorageVolume  *StorageVolumeRepository
	VMBackup       *VMBackupRepository
	BackupSchedule *BackupScheduleRepository
	VMSnapshot     *VMSnapshotRepository
}

func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		DB:             db,
		User:           NewUserRepository(db),
		VM:             NewVMRepository(db),
		Template:       NewTemplateRepository(db),
		VMStats:        NewVMStatsRepository(db),
		AuditLog:       NewAuditLogRepository(db),
		TemplateUpload: NewTemplateUploadRepository(db),
		AlertRule:      NewAlertRuleRepository(db),
		AlertHistory:   NewAlertHistoryRepository(db),
		ISO:            NewISORepository(db),
		ISOUpload:      NewISOUploadRepository(db),
		VirtualNetwork: NewVirtualNetworkRepository(db),
		StoragePool:    NewStoragePoolRepository(db),
		StorageVolume:  NewStorageVolumeRepository(db),
		VMBackup:       NewVMBackupRepository(db),
		BackupSchedule: NewBackupScheduleRepository(db),
		VMSnapshot:     NewVMSnapshotRepository(db),
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
