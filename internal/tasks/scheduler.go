//go:build !linux || mock
// +build !linux mock

package tasks

import (
	"context"
	"log"
	"time"

	"vmmanager/internal/models"
	"vmmanager/internal/repository"
)

type Scheduler struct {
	db        interface{}
	vmRepo    *repository.VMRepository
	statsRepo *repository.VMStatsRepository
	libvirt   interface{}
	stopChan  chan struct{}
}

func NewScheduler(db interface{}, libvirtClient interface{}) *Scheduler {
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	vms, _, _ := s.vmRepo.List(ctx, 0, 0)

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

		s.statsRepo.Create(ctx, &stats)
	}
}
