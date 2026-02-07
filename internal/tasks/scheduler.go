//go:build !linux || mock
// +build !linux mock

package tasks

import (
	"log"
	"time"

	"vmmanager/internal/models"

	"gorm.io/gorm"
)

type Scheduler struct {
	db       *gorm.DB
	libvirt  interface{}
	stopChan chan struct{}
}

func NewScheduler(db *gorm.DB, libvirtClient interface{}) *Scheduler {
	return &Scheduler{
		db:       db,
		libvirt:  libvirtClient,
		stopChan: make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		for {
			select {
			case <-ticker.C:
				s.collectStats()
			case <-s.stopChan:
				ticker.Stop()
				return
			}
		}
	}()

	log.Println("Task scheduler started")
}

func (s *Scheduler) Stop() {
	close(s.stopChan)
	log.Println("Task scheduler stopped")
}

func (s *Scheduler) collectStats() {
	var vms []models.VirtualMachine
	s.db.Where("status = ? AND deleted_at IS NULL", "running").Find(&vms)

	for _, vm := range vms {
		if s.libvirt == nil {
			continue
		}

		stats := models.VMStats{
			VMID:        vm.ID,
			CPUUsage:    0,
			MemoryUsage: 0,
			MemoryTotal: int64(vm.MemoryAllocated),
			CollectedAt: time.Now(),
		}

		s.db.Create(&stats)
	}
}
