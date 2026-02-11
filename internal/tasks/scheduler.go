//go:build !linux || mock
// +build !linux mock

package tasks

import (
	"context"
	"log"
	"time"
	"vmmanager/internal/models"
	"vmmanager/internal/repository"

	"gorm.io/gorm"
	"vmmanager/internal/libvirt"
)

type Scheduler struct {
	db        *gorm.DB
	vmRepo    *repository.VMRepository
	statsRepo *repository.VMStatsRepository
	libvirt   *libvirt.Client
	stopChan  chan struct{}
}

func NewScheduler(db *gorm.DB, libvirtClient *libvirt.Client) *Scheduler {
	return &Scheduler{
		db:        db,
		vmRepo:    repository.NewVMRepository(db),
		statsRepo: repository.NewVMStatsRepository(db),
		libvirt:   libvirtClient,
		stopChan:  make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	ticker := time.NewTicker(30 * time.Second)

	go func() {
		for {
			select {
			case <-ticker.C:
				s.collectStats()
				s.syncVMStatus()
			case <-s.stopChan:
				ticker.Stop()
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

		if vm.Status != expectedStatus {
			log.Printf("[SCHEDULER] Syncing VM %s status: %s -> %s", vm.Name, vm.Status, expectedStatus)
			s.vmRepo.UpdateStatus(ctx, vm.ID.String(), expectedStatus)
		}
	}
}

func (s *Scheduler) Stop() {
	close(s.stopChan)
	log.Println("Task scheduler stopped")
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
