package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"vmmanager/internal/libvirt"
	"vmmanager/internal/models"
	"vmmanager/internal/repository"

	"github.com/gorilla/websocket"
)

type InstallStatus string

const (
	InstallStatusPending    InstallStatus = "pending"
	InstallStatusInstalling InstallStatus = "installing"
	InstallStatusCompleted  InstallStatus = "completed"
	InstallStatusFailed     InstallStatus = "failed"
	InstallStatusPaused     InstallStatus = "paused"
)

type InstallProgress struct {
	VMID         string        `json:"vmId"`
	VMName       string        `json:"vmName"`
	Status       InstallStatus `json:"status"`
	Progress     int           `json:"progress"`
	Message      string        `json:"message"`
	CurrentStep  string        `json:"currentStep"`
	TotalSteps   int           `json:"totalSteps"`
	CompletedAt  *time.Time    `json:"completedAt,omitempty"`
	StartedAt    *time.Time    `json:"startedAt,omitempty"`
	ErrorMessage string        `json:"errorMessage,omitempty"`
}

type InstallMonitor struct {
	vmRepo     *repository.VMRepository
	libvirt    *libvirt.Client
	clients    map[string]map[*websocket.Conn]bool
	clientsMu  sync.RWMutex
	progress   map[string]*InstallProgress
	progressMu sync.RWMutex
	stopChan   chan struct{}
	running    bool
	runningMu  sync.Mutex
}

func NewInstallMonitor(vmRepo *repository.VMRepository, libvirtClient *libvirt.Client) *InstallMonitor {
	return &InstallMonitor{
		vmRepo:   vmRepo,
		libvirt:  libvirtClient,
		clients:  make(map[string]map[*websocket.Conn]bool),
		progress: make(map[string]*InstallProgress),
		stopChan: make(chan struct{}),
	}
}

func (m *InstallMonitor) Start() {
	m.runningMu.Lock()
	defer m.runningMu.Unlock()
	if m.running {
		return
	}
	m.running = true
	go m.monitorLoop()
	log.Printf("[InstallMonitor] Started monitoring")
}

func (m *InstallMonitor) Stop() {
	m.runningMu.Lock()
	defer m.runningMu.Unlock()
	if !m.running {
		return
	}
	m.running = false
	close(m.stopChan)
	log.Printf("[InstallMonitor] Stopped monitoring")
}

func (m *InstallMonitor) monitorLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.checkAllVMs()
		}
	}
}

func (m *InstallMonitor) checkAllVMs() {
	ctx := context.Background()

	vms, _, err := m.vmRepo.List(ctx, 0, 1000)
	if err != nil {
		log.Printf("[InstallMonitor] Failed to list VMs: %v", err)
		return
	}

	for _, vm := range vms {
		if vm.InstallStatus != "" && vm.InstallStatus != "completed" && vm.InstallStatus != "failed" {
			m.checkVMInstallStatus(ctx, &vm)
		}
	}
}

func (m *InstallMonitor) checkVMInstallStatus(ctx context.Context, vm *models.VirtualMachine) {
	if m.libvirt == nil {
		return
	}

	progress := m.getOrCreateProgress(vm)

	if vm.LibvirtDomainUUID == "" {
		progress.Status = InstallStatusPending
		progress.Message = "Waiting for domain creation"
		m.broadcastProgress(vm.ID.String(), progress)
		return
	}

	domain, err := m.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
	if err != nil {
		progress.Status = InstallStatusPending
		progress.Message = "Domain not found in libvirt"
		m.broadcastProgress(vm.ID.String(), progress)
		return
	}
	defer domain.Free()

	state, _, err := domain.GetState()
	if err != nil {
		progress.Status = InstallStatusFailed
		progress.Message = "Failed to get domain state"
		m.broadcastProgress(vm.ID.String(), progress)
		return
	}

	switch state {
	case 1:
		if progress.StartedAt == nil {
			now := time.Now()
			progress.StartedAt = &now
			progress.Status = InstallStatusInstalling
			progress.Message = "Installation in progress"
		}
		progress.Progress = m.estimateProgress(vm, progress)

		if m.detectInstallationComplete(vm, domain) {
			progress.Status = InstallStatusCompleted
			progress.Progress = 100
			now := time.Now()
			progress.CompletedAt = &now
			progress.Message = "Installation completed successfully"

			m.vmRepo.UpdateInstallStatus(ctx, vm.ID.String(), "completed")
		}

	case 3:
		progress.Status = InstallStatusPaused
		progress.Message = "VM is paused"

	case 5:
		if progress.Status == InstallStatusInstalling {
			progress.Status = InstallStatusCompleted
			progress.Progress = 100
			now := time.Now()
			progress.CompletedAt = &now
			progress.Message = "Installation completed, VM stopped"

			m.vmRepo.UpdateInstallStatus(ctx, vm.ID.String(), "completed")
		} else {
			progress.Status = InstallStatusPending
			progress.Message = "VM is stopped, waiting to start"
		}

	default:
		progress.Message = fmt.Sprintf("VM state: %d", state)
	}

	m.broadcastProgress(vm.ID.String(), progress)
}

func (m *InstallMonitor) getOrCreateProgress(vm *models.VirtualMachine) *InstallProgress {
	m.progressMu.Lock()
	defer m.progressMu.Unlock()

	if p, exists := m.progress[vm.ID.String()]; exists {
		return p
	}

	p := &InstallProgress{
		VMID:        vm.ID.String(),
		VMName:      vm.Name,
		Status:      InstallStatusPending,
		Progress:    0,
		Message:     "Initializing",
		CurrentStep: "init",
		TotalSteps:  4,
	}
	m.progress[vm.ID.String()] = p
	return p
}

func (m *InstallMonitor) estimateProgress(vm *models.VirtualMachine, progress *InstallProgress) int {
	if progress.StartedAt == nil {
		return 0
	}

	elapsed := time.Since(*progress.StartedAt)

	estimatedDuration := 10 * time.Minute

	estimatedProgress := int(float64(elapsed) / float64(estimatedDuration) * 100)
	if estimatedProgress > 95 {
		estimatedProgress = 95
	}
	if estimatedProgress < 0 {
		estimatedProgress = 0
	}

	return estimatedProgress
}

func (m *InstallMonitor) detectInstallationComplete(vm *models.VirtualMachine, domain *libvirt.Domain) bool {
	xmlDesc, err := domain.GetXMLDesc()
	if err != nil {
		return false
	}

	if vm.InstallationMode == "iso" {
		elapsed := 0 * time.Second
		if progress, exists := m.progress[vm.ID.String()]; exists && progress.StartedAt != nil {
			elapsed = time.Since(*progress.StartedAt)
		}

		minInstallTime := 2 * time.Minute
		if elapsed < minInstallTime {
			return false
		}
	}

	_ = xmlDesc

	return false
}

func (m *InstallMonitor) broadcastProgress(vmID string, progress *InstallProgress) {
	m.clientsMu.RLock()
	clients, exists := m.clients[vmID]
	if !exists {
		m.clientsMu.RUnlock()
		return
	}

	message, err := json.Marshal(map[string]interface{}{
		"type":    "install_progress",
		"payload": progress,
	})
	if err != nil {
		m.clientsMu.RUnlock()
		return
	}

	var disconnected []*websocket.Conn
	for conn := range clients {
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			disconnected = append(disconnected, conn)
		}
	}
	m.clientsMu.RUnlock()

	for _, conn := range disconnected {
		m.Unsubscribe(vmID, conn)
	}
}

func (m *InstallMonitor) Subscribe(vmID string, conn *websocket.Conn) {
	m.clientsMu.Lock()
	defer m.clientsMu.Unlock()

	if m.clients[vmID] == nil {
		m.clients[vmID] = make(map[*websocket.Conn]bool)
	}
	m.clients[vmID][conn] = true

	if progress, exists := m.progress[vmID]; exists {
		message, _ := json.Marshal(map[string]interface{}{
			"type":    "install_progress",
			"payload": progress,
		})
		conn.WriteMessage(websocket.TextMessage, message)
	}

	log.Printf("[InstallMonitor] Client subscribed to VM %s", vmID)
}

func (m *InstallMonitor) Unsubscribe(vmID string, conn *websocket.Conn) {
	m.clientsMu.Lock()
	defer m.clientsMu.Unlock()

	if clients, exists := m.clients[vmID]; exists {
		delete(clients, conn)
		if len(clients) == 0 {
			delete(m.clients, vmID)
		}
	}

	log.Printf("[InstallMonitor] Client unsubscribed from VM %s", vmID)
}

func (m *InstallMonitor) GetProgress(vmID string) *InstallProgress {
	m.progressMu.RLock()
	defer m.progressMu.RUnlock()
	return m.progress[vmID]
}

func (m *InstallMonitor) MarkInstallStarted(vmID string, vmName string) {
	m.progressMu.Lock()
	defer m.progressMu.Unlock()

	now := time.Now()
	m.progress[vmID] = &InstallProgress{
		VMID:        vmID,
		VMName:      vmName,
		Status:      InstallStatusInstalling,
		Progress:    0,
		Message:     "Installation started",
		CurrentStep: "boot",
		TotalSteps:  4,
		StartedAt:   &now,
	}
}

func (m *InstallMonitor) MarkInstallCompleted(vmID string) {
	m.progressMu.Lock()
	defer m.progressMu.Unlock()

	if progress, exists := m.progress[vmID]; exists {
		progress.Status = InstallStatusCompleted
		progress.Progress = 100
		now := time.Now()
		progress.CompletedAt = &now
		progress.Message = "Installation completed"
	}
}

func (m *InstallMonitor) MarkInstallFailed(vmID string, errMsg string) {
	m.progressMu.Lock()
	defer m.progressMu.Unlock()

	if progress, exists := m.progress[vmID]; exists {
		progress.Status = InstallStatusFailed
		progress.ErrorMessage = errMsg
		progress.Message = "Installation failed: " + errMsg
	}
}

func (m *InstallMonitor) UpdateProgress(vmID string, progress int, message string) {
	m.progressMu.Lock()
	defer m.progressMu.Unlock()

	if p, exists := m.progress[vmID]; exists {
		p.Progress = progress
		p.Message = message
	}
}
