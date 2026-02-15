package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"vmmanager/internal/libvirt"
	"vmmanager/internal/models"
	"vmmanager/internal/repository"

	"github.com/google/uuid"
)

type BackupService struct {
	backupRepo   *repository.VMBackupRepository
	scheduleRepo *repository.BackupScheduleRepository
	vmRepo       *repository.VMRepository
	libvirt      *libvirt.Client
	backupDir    string
	stopChan     chan struct{}
	wg           sync.WaitGroup
	mu           sync.Mutex
	running      map[string]bool
}

func NewBackupService(
	backupRepo *repository.VMBackupRepository,
	scheduleRepo *repository.BackupScheduleRepository,
	vmRepo *repository.VMRepository,
	libvirtClient *libvirt.Client,
	backupDir string,
) *BackupService {
	return &BackupService{
		backupRepo:   backupRepo,
		scheduleRepo: scheduleRepo,
		vmRepo:       vmRepo,
		libvirt:      libvirtClient,
		backupDir:    backupDir,
		stopChan:     make(chan struct{}),
		running:      make(map[string]bool),
	}
}

func (s *BackupService) Start() {
	log.Println("[BackupService] Starting backup service...")

	if err := os.MkdirAll(s.backupDir, 0755); err != nil {
		log.Printf("[BackupService] Failed to create backup directory: %v", err)
	}

	s.wg.Add(2)
	go s.runScheduledBackups()
	go s.runExpiredCleanup()

	log.Println("[BackupService] Backup service started")
}

func (s *BackupService) Stop() {
	log.Println("[BackupService] Stopping backup service...")
	close(s.stopChan)
	s.wg.Wait()
	log.Println("[BackupService] Backup service stopped")
}

func (s *BackupService) runScheduledBackups() {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkScheduledBackups()
			s.checkScheduleTriggers()
		}
	}
}

func (s *BackupService) runExpiredCleanup() {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.cleanupExpiredBackups()
		}
	}
}

func (s *BackupService) checkScheduledBackups() {
	ctx := context.Background()

	backups, err := s.backupRepo.ListScheduled(ctx, time.Now())
	if err != nil {
		log.Printf("[BackupService] Failed to list scheduled backups: %v", err)
		return
	}

	for _, backup := range backups {
		if !s.isRunning(backup.ID.String()) {
			go s.ExecuteBackup(backup.ID.String())
		}
	}
}

func (s *BackupService) checkScheduleTriggers() {
	ctx := context.Background()

	schedules, err := s.scheduleRepo.ListEnabled(ctx)
	if err != nil {
		log.Printf("[BackupService] Failed to list enabled schedules: %v", err)
		return
	}

	now := time.Now()
	for _, schedule := range schedules {
		if schedule.NextRunAt == nil || schedule.NextRunAt.Before(now) || schedule.NextRunAt.Equal(now) {
			s.triggerScheduledBackup(ctx, &schedule)
		}
	}
}

func (s *BackupService) triggerScheduledBackup(ctx context.Context, schedule *models.BackupSchedule) {
	vm, err := s.vmRepo.FindByID(ctx, schedule.VMID.String())
	if err != nil {
		log.Printf("[BackupService] Failed to find VM %s: %v", schedule.VMID, err)
		return
	}

	backupName := fmt.Sprintf("%s-auto-%s", vm.Name, time.Now().Format("20060102-150405"))
	expiresAt := time.Now().AddDate(0, 0, schedule.Retention)

	backup := &models.VMBackup{
		VMID:        schedule.VMID,
		Name:        backupName,
		Description: fmt.Sprintf("Auto backup from schedule: %s", schedule.Name),
		BackupType:  schedule.BackupType,
		Status:      "pending",
		ExpiresAt:   &expiresAt,
		CreatedBy:   schedule.CreatedBy,
	}

	if err := s.backupRepo.Create(ctx, backup); err != nil {
		log.Printf("[BackupService] Failed to create scheduled backup: %v", err)
		return
	}

	nextRun := s.calculateNextRun(schedule.CronExpr)
	if err := s.scheduleRepo.UpdateLastRun(ctx, schedule.ID.String(), time.Now(), nextRun); err != nil {
		log.Printf("[BackupService] Failed to update schedule last run: %v", err)
	}

	log.Printf("[BackupService] Created scheduled backup %s for VM %s", backup.ID, vm.Name)

	go s.ExecuteBackup(backup.ID.String())
}

func (s *BackupService) calculateNextRun(cronExpr string) time.Time {
	return time.Now().Add(24 * time.Hour)
}

func (s *BackupService) ExecuteBackup(backupID string) error {
	s.mu.Lock()
	if s.isRunning(backupID) {
		s.mu.Unlock()
		return fmt.Errorf("backup %s is already running", backupID)
	}
	s.running[backupID] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.running, backupID)
		s.mu.Unlock()
	}()

	ctx := context.Background()

	backup, err := s.backupRepo.FindByID(ctx, backupID)
	if err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}

	if backup.Status != "pending" {
		return fmt.Errorf("backup is not in pending status")
	}

	vm, err := s.vmRepo.FindByID(ctx, backup.VMID.String())
	if err != nil {
		s.backupRepo.UpdateStatus(ctx, backupID, "failed", 0, fmt.Sprintf("VM not found: %v", err))
		return fmt.Errorf("VM not found: %w", err)
	}

	if err := s.backupRepo.UpdateStatus(ctx, backupID, "running", 0, ""); err != nil {
		return fmt.Errorf("failed to update backup status: %w", err)
	}

	log.Printf("[BackupService] Starting backup %s for VM %s", backupID, vm.Name)

	backupFileName := fmt.Sprintf("%s-%s.qcow2", vm.Name, time.Now().Format("20060102-150405"))
	backupPath := filepath.Join(s.backupDir, backupFileName)

	if err := s.createBackupFile(ctx, backup, vm, backupPath); err != nil {
		s.backupRepo.UpdateStatus(ctx, backupID, "failed", 0, err.Error())
		return fmt.Errorf("failed to create backup file: %w", err)
	}

	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		s.backupRepo.UpdateStatus(ctx, backupID, "failed", 0, fmt.Sprintf("Failed to get file info: %v", err))
		return fmt.Errorf("failed to get file info: %w", err)
	}

	backup.FilePath = backupPath
	backup.FileSize = fileInfo.Size()
	backup.Status = "completed"
	backup.Progress = 100
	now := time.Now()
	backup.CompletedAt = &now

	if err := s.backupRepo.Update(ctx, backup); err != nil {
		return fmt.Errorf("failed to update backup: %w", err)
	}

	log.Printf("[BackupService] Backup %s completed successfully", backupID)
	return nil
}

func (s *BackupService) createBackupFile(ctx context.Context, backup *models.VMBackup, vm *models.VirtualMachine, backupPath string) error {
	s.backupRepo.UpdateStatus(ctx, backup.ID.String(), "running", 10, "")

	if vm.Status != "running" {
		s.backupRepo.UpdateStatus(ctx, backup.ID.String(), "running", 20, "")
		if err := s.copyDiskFile(vm.DiskPath, backupPath); err != nil {
			return fmt.Errorf("failed to copy disk file: %w", err)
		}
		s.backupRepo.UpdateStatus(ctx, backup.ID.String(), "running", 80, "")
		return nil
	}

	s.backupRepo.UpdateStatus(ctx, backup.ID.String(), "running", 20, "Pausing VM for snapshot")

	s.backupRepo.UpdateStatus(ctx, backup.ID.String(), "running", 30, "Creating snapshot")

	if err := s.copyDiskFile(vm.DiskPath, backupPath); err != nil {
		return fmt.Errorf("failed to copy disk file: %w", err)
	}

	s.backupRepo.UpdateStatus(ctx, backup.ID.String(), "running", 80, "Finalizing backup")

	log.Printf("[BackupService] Created backup file at %s", backupPath)
	return nil
}

func (s *BackupService) copyDiskFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	return nil
}

func (s *BackupService) RestoreBackup(backupID string, vmID string) error {
	ctx := context.Background()

	backup, err := s.backupRepo.FindByID(ctx, backupID)
	if err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}

	if backup.Status != "completed" {
		return fmt.Errorf("backup is not completed")
	}

	vm, err := s.vmRepo.FindByID(ctx, vmID)
	if err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	if vm.Status == "running" {
		return fmt.Errorf("cannot restore backup to a running VM")
	}

	if _, err := os.Stat(backup.FilePath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backup.FilePath)
	}

	if err := s.copyDiskFile(backup.FilePath, vm.DiskPath); err != nil {
		return fmt.Errorf("failed to restore disk: %w", err)
	}

	log.Printf("[BackupService] Restored backup %s to VM %s", backupID, vm.Name)
	return nil
}

func (s *BackupService) cleanupExpiredBackups() {
	ctx := context.Background()

	backups, err := s.backupRepo.ListExpired(ctx)
	if err != nil {
		log.Printf("[BackupService] Failed to list expired backups: %v", err)
		return
	}

	for _, backup := range backups {
		log.Printf("[BackupService] Cleaning up expired backup %s", backup.ID)

		if backup.FilePath != "" {
			if err := os.Remove(backup.FilePath); err != nil && !os.IsNotExist(err) {
				log.Printf("[BackupService] Failed to delete backup file %s: %v", backup.FilePath, err)
				continue
			}
		}

		if err := s.backupRepo.Delete(ctx, backup.ID.String()); err != nil {
			log.Printf("[BackupService] Failed to delete backup record %s: %v", backup.ID, err)
		}
	}
}

func (s *BackupService) DeleteBackup(backupID string) error {
	ctx := context.Background()

	backup, err := s.backupRepo.FindByID(ctx, backupID)
	if err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}

	if backup.Status == "running" {
		return fmt.Errorf("cannot delete a running backup")
	}

	if backup.FilePath != "" {
		if err := os.Remove(backup.FilePath); err != nil && !os.IsNotExist(err) {
			log.Printf("[BackupService] Warning: failed to delete backup file %s: %v", backup.FilePath, err)
		}
	}

	if err := s.backupRepo.Delete(ctx, backupID); err != nil {
		return fmt.Errorf("failed to delete backup record: %w", err)
	}

	log.Printf("[BackupService] Deleted backup %s", backupID)
	return nil
}

func (s *BackupService) isRunning(backupID string) bool {
	_, running := s.running[backupID]
	return running
}

func (s *BackupService) GetBackupProgress(backupID string) (int, string, error) {
	ctx := context.Background()

	backup, err := s.backupRepo.FindByID(ctx, backupID)
	if err != nil {
		return 0, "", fmt.Errorf("backup not found: %w", err)
	}

	return backup.Progress, backup.Status, nil
}

func (s *BackupService) CreateManualBackup(vmID string, name string, description string, backupType string, retentionDays int) (*models.VMBackup, error) {
	ctx := context.Background()

	vm, err := s.vmRepo.FindByID(ctx, vmID)
	if err != nil {
		return nil, fmt.Errorf("VM not found: %w", err)
	}

	var expiresAt *time.Time
	if retentionDays > 0 {
		exp := time.Now().AddDate(0, 0, retentionDays)
		expiresAt = &exp
	}

	if backupType == "" {
		backupType = "full"
	}

	backup := &models.VMBackup{
		VMID:        uuid.MustParse(vmID),
		Name:        name,
		Description: description,
		BackupType:  backupType,
		Status:      "pending",
		ExpiresAt:   expiresAt,
	}

	if err := s.backupRepo.Create(ctx, backup); err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	go s.ExecuteBackup(backup.ID.String())

	log.Printf("[BackupService] Created manual backup %s for VM %s", backup.ID, vm.Name)
	return backup, nil
}
