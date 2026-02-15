package tasks

import (
	"context"
	"log"
	"os"
	"time"
	"vmmanager/internal/models"
	"vmmanager/internal/repository"

	"vmmanager/internal/libvirt"

	"gorm.io/gorm"
)

const (
	UploadCleanupInterval = 1 * time.Hour
	UploadExpireDuration  = 24 * time.Hour
)

type Scheduler struct {
	db                 *gorm.DB
	vmRepo             *repository.VMRepository
	statsRepo          *repository.VMStatsRepository
	isoUploadRepo      *repository.ISOUploadRepository
	templateUploadRepo *repository.TemplateUploadRepository
	libvirt            *libvirt.Client
	stopChan           chan struct{}
}

func NewScheduler(db *gorm.DB, libvirtClient *libvirt.Client) *Scheduler {
	return &Scheduler{
		db:                 db,
		vmRepo:             repository.NewVMRepository(db),
		statsRepo:          repository.NewVMStatsRepository(db),
		isoUploadRepo:      repository.NewISOUploadRepository(db),
		templateUploadRepo: repository.NewTemplateUploadRepository(db),
		libvirt:            libvirtClient,
		stopChan:           make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	ticker := time.NewTicker(30 * time.Second)
	cleanupTicker := time.NewTicker(UploadCleanupInterval)

	go func() {
		for {
			select {
			case <-ticker.C:
				s.collectStats()
				s.syncVMStatus()
			case <-cleanupTicker.C:
				s.cleanupExpiredUploads()
			case <-s.stopChan:
				ticker.Stop()
				cleanupTicker.Stop()
				return
			}
		}
	}()

	log.Println("Task scheduler started")
}

func (s *Scheduler) syncVMStatus() {
	if s.libvirt == nil || s.vmRepo == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	vms, _, err := s.vmRepo.List(ctx, 0, 0)
	if err != nil {
		return
	}

	for _, vm := range vms {
		if vm.LibvirtDomainUUID == "" || vm.LibvirtDomainUUID == "new-uuid" || vm.LibvirtDomainUUID == "defined-uuid" {
			continue
		}

		domain, err := s.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
		if err != nil {
			continue
		}

		state, _, _ := domain.GetState()

		var expectedStatus string
		switch state {
		case 1:
			expectedStatus = "running"
		case 0:
			expectedStatus = "stopped"
		case 3:
			expectedStatus = "suspended"
		default:
			continue
		}

		// 避免覆盖中间状态（starting/stopping/creating）
		if vm.Status != expectedStatus && !isTransitionalStatus(vm.Status) {
			log.Printf("[SCHEDULER] Syncing VM %s status: %s -> %s", vm.Name, vm.Status, expectedStatus)
			s.vmRepo.UpdateStatus(ctx, vm.ID.String(), expectedStatus)
		}
	}
}

func (s *Scheduler) Stop() {
	close(s.stopChan)
	log.Println("Task scheduler stopped")
}

// isTransitionalStatus 检查状态是否为中间状态
// 中间状态表示 VM 正在执行操作中，不应被调度器覆盖
func isTransitionalStatus(status string) bool {
	return status == "starting" || status == "stopping" || status == "creating"
}

func (s *Scheduler) collectStats() {
	if s.vmRepo == nil || s.statsRepo == nil {
		log.Println("Warning: repositories not initialized, skipping stats collection")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	vms, _, err := s.vmRepo.List(ctx, 0, 0)
	if err != nil {
		log.Printf("Error collecting VM stats: %v", err)
		return
	}

	for _, vm := range vms {
		if s.libvirt == nil || vm.Status != "running" {
			continue
		}

		stats := models.VMStats{
			VMID:        vm.ID,
			CPUUsage:    0,
			MemoryUsage: 0,
			MemoryTotal: int64(vm.MemoryAllocated),
			CollectedAt: time.Now(),
		}

		if err := s.statsRepo.Create(ctx, &stats); err != nil {
			log.Printf("Error creating VM stats: %v", err)
		}
	}
}

func (s *Scheduler) cleanupExpiredUploads() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	expireTime := time.Now().Add(-UploadExpireDuration)

	s.cleanupExpiredISOUploads(ctx, expireTime)
	s.cleanupExpiredTemplateUploads(ctx, expireTime)
}

func (s *Scheduler) cleanupExpiredISOUploads(ctx context.Context, expireTime time.Time) {
	var expiredUploads []models.ISOUpload
	if err := s.db.WithContext(ctx).
		Where("status = ? AND created_at < ?", "uploading", expireTime).
		Find(&expiredUploads).Error; err != nil {
		log.Printf("[CLEANUP] Error finding expired ISO uploads: %v", err)
		return
	}

	for _, upload := range expiredUploads {
		if upload.TempPath != "" {
			if err := os.Remove(upload.TempPath); err != nil && !os.IsNotExist(err) {
				log.Printf("[CLEANUP] Error removing temp file %s: %v", upload.TempPath, err)
			}
		}

		if err := s.isoUploadRepo.Delete(ctx, upload.ID.String()); err != nil {
			log.Printf("[CLEANUP] Error deleting ISO upload record %s: %v", upload.ID, err)
		} else {
			log.Printf("[CLEANUP] Cleaned up expired ISO upload: %s (%s)", upload.FileName, upload.ID)
		}
	}
}

func (s *Scheduler) cleanupExpiredTemplateUploads(ctx context.Context, expireTime time.Time) {
	var expiredUploads []models.TemplateUpload
	if err := s.db.WithContext(ctx).
		Where("status = ? AND created_at < ?", "uploading", expireTime).
		Find(&expiredUploads).Error; err != nil {
		log.Printf("[CLEANUP] Error finding expired template uploads: %v", err)
		return
	}

	for _, upload := range expiredUploads {
		if upload.TempPath != "" {
			if err := os.Remove(upload.TempPath); err != nil && !os.IsNotExist(err) {
				log.Printf("[CLEANUP] Error removing temp file %s: %v", upload.TempPath, err)
			}
		}

		if err := s.templateUploadRepo.Delete(ctx, upload.ID.String()); err != nil {
			log.Printf("[CLEANUP] Error deleting template upload record %s: %v", upload.ID, err)
		} else {
			log.Printf("[CLEANUP] Cleaned up expired template upload: %s (%s)", upload.FileName, upload.ID)
		}
	}
}
