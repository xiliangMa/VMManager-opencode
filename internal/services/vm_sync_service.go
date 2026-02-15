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
	libvirtgo "github.com/libvirt/libvirt-go"
)

type VMStatus string

const (
	VMStatusNoState   VMStatus = "no_state"
	VMStatusRunning   VMStatus = "running"
	VMStatusBlocked   VMStatus = "blocked"
	VMStatusPaused    VMStatus = "paused"
	VMStatusShutdown  VMStatus = "shutdown"
	VMStatusShutoff   VMStatus = "shutoff"
	VMStatusCrashed   VMStatus = "crashed"
	VMStatusSuspended VMStatus = "suspended"
)

type VMStatusInfo struct {
	VMID           string    `json:"vm_id"`
	VMName         string    `json:"vm_name"`
	Status         string    `json:"status"`
	LibvirtState   int       `json:"libvirt_state"`
	LibvirtReason  int       `json:"libvirt_reason"`
	CPUCount       uint      `json:"cpu_count"`
	MemoryMB       uint64    `json:"memory_mb"`
	ActualMemoryKB uint64    `json:"actual_memory_kb"`
	CPUTime        uint64    `json:"cpu_time"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type StatusChangeEvent struct {
	Type      string        `json:"type"`
	VMID      string        `json:"vm_id"`
	OldStatus string        `json:"old_status"`
	NewStatus string        `json:"new_status"`
	Timestamp time.Time     `json:"timestamp"`
	Info      *VMStatusInfo `json:"info,omitempty"`
}

type VMSyncService struct {
	libvirt      *libvirt.Client
	vmRepo       *repository.VMRepository
	wsHub        *WebSocketHub
	stopChan     chan struct{}
	wg           sync.WaitGroup
	mu           sync.RWMutex
	statusCache  map[string]*VMStatusInfo
	eventChan    chan StatusChangeEvent
	syncInterval time.Duration
}

type WebSocketHub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[WS_HUB] Client connected, total clients: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
			}
			h.mu.Unlock()
			log.Printf("[WS_HUB] Client disconnected, total clients: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					log.Printf("[WS_HUB] Error sending message: %v", err)
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *WebSocketHub) Register(client *websocket.Conn) {
	h.register <- client
}

func (h *WebSocketHub) Unregister(client *websocket.Conn) {
	h.unregister <- client
}

func (h *WebSocketHub) Broadcast(message []byte) {
	select {
	case h.broadcast <- message:
	default:
		log.Printf("[WS_HUB] Broadcast channel full, dropping message")
	}
}

func NewVMSyncService(libvirtClient *libvirt.Client, vmRepo *repository.VMRepository, syncInterval time.Duration) *VMSyncService {
	if syncInterval == 0 {
		syncInterval = 10 * time.Second
	}

	return &VMSyncService{
		libvirt:      libvirtClient,
		vmRepo:       vmRepo,
		wsHub:        NewWebSocketHub(),
		stopChan:     make(chan struct{}),
		statusCache:  make(map[string]*VMStatusInfo),
		eventChan:    make(chan StatusChangeEvent, 100),
		syncInterval: syncInterval,
	}
}

func (s *VMSyncService) Start() {
	s.wg.Add(3)

	go s.wsHub.Run()

	go s.syncLoop()

	go s.eventBroadcastLoop()

	log.Printf("[VM_SYNC] Service started with sync interval: %v", s.syncInterval)
}

func (s *VMSyncService) Stop() {
	close(s.stopChan)
	s.wg.Wait()
	log.Printf("[VM_SYNC] Service stopped")
}

func (s *VMSyncService) syncLoop() {
	defer s.wg.Done()

	s.fullSync()

	ticker := time.NewTicker(s.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.fullSync()
		case <-s.stopChan:
			return
		}
	}
}

func (s *VMSyncService) fullSync() {
	if s.libvirt == nil || s.vmRepo == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	vms, _, err := s.vmRepo.List(ctx, 0, 0)
	if err != nil {
		log.Printf("[VM_SYNC] Failed to list VMs: %v", err)
		return
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10)

	for i := range vms {
		vm := &vms[i]
		if vm.LibvirtDomainUUID == "" || vm.LibvirtDomainUUID == "new-uuid" || vm.LibvirtDomainUUID == "defined-uuid" {
			continue
		}

		wg.Add(1)
		go func(vm *models.VirtualMachine) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			s.syncSingleVM(ctx, vm)
		}(vm)
	}

	wg.Wait()
}

func (s *VMSyncService) syncSingleVM(ctx context.Context, vm *models.VirtualMachine) {
	if s.libvirt == nil {
		return
	}

	domain, err := s.libvirt.LookupByUUID(vm.LibvirtDomainUUID)
	if err != nil {
		s.mu.RLock()
		cachedInfo, exists := s.statusCache[vm.ID.String()]
		s.mu.RUnlock()

		if exists && cachedInfo.Status != "stopped" {
			s.handleStatusChange(ctx, vm, cachedInfo.Status, "stopped", nil)
		}
		return
	}
	defer domain.Free()

	state, reason, err := domain.GetState()
	if err != nil {
		log.Printf("[VM_SYNC] Failed to get domain state for %s: %v", vm.Name, err)
		return
	}

	info, err := s.getDomainInfo(domain, vm, state, int(reason))
	if err != nil {
		log.Printf("[VM_SYNC] Failed to get domain info for %s: %v", vm.Name, err)
		return
	}

	s.mu.RLock()
	cachedInfo, exists := s.statusCache[vm.ID.String()]
	s.mu.RUnlock()

	newStatus := libvirtStateToStatus(state)
	oldStatus := vm.Status

	if exists && cachedInfo.Status != newStatus {
		oldStatus = cachedInfo.Status
	}

	if !isTransitionalStatus(oldStatus) && oldStatus != newStatus {
		s.handleStatusChange(ctx, vm, oldStatus, newStatus, info)
	}

	s.mu.Lock()
	s.statusCache[vm.ID.String()] = info
	s.mu.Unlock()
}

func (s *VMSyncService) getDomainInfo(domain *libvirt.Domain, vm *models.VirtualMachine, state int, reason int) (*VMStatusInfo, error) {
	info := &VMStatusInfo{
		VMID:          vm.ID.String(),
		VMName:        vm.Name,
		Status:        libvirtStateToStatus(state),
		LibvirtState:  state,
		LibvirtReason: reason,
		CPUCount:      uint(vm.CPUAllocated),
		MemoryMB:      uint64(vm.MemoryAllocated),
		UpdatedAt:     time.Now(),
	}

	return info, nil
}

func (s *VMSyncService) handleStatusChange(ctx context.Context, vm *models.VirtualMachine, oldStatus, newStatus string, info *VMStatusInfo) {
	log.Printf("[VM_SYNC] VM %s status changed: %s -> %s", vm.Name, oldStatus, newStatus)

	if err := s.vmRepo.UpdateStatus(ctx, vm.ID.String(), newStatus); err != nil {
		log.Printf("[VM_SYNC] Failed to update VM status in database: %v", err)
		return
	}

	event := StatusChangeEvent{
		Type:      "status_change",
		VMID:      vm.ID.String(),
		OldStatus: oldStatus,
		NewStatus: newStatus,
		Timestamp: time.Now(),
		Info:      info,
	}

	select {
	case s.eventChan <- event:
	default:
		log.Printf("[VM_SYNC] Event channel full, dropping event for VM %s", vm.Name)
	}
}

func (s *VMSyncService) eventBroadcastLoop() {
	defer s.wg.Done()

	for {
		select {
		case event := <-s.eventChan:
			message := map[string]interface{}{
				"type": "vm_status_update",
				"data": event,
			}
			messageData, err := json.Marshal(message)
			if err != nil {
				log.Printf("[VM_SYNC] Failed to marshal message: %v", err)
				continue
			}

			s.wsHub.Broadcast(messageData)

		case <-s.stopChan:
			return
		}
	}
}

func (s *VMSyncService) GetVMStatus(vmID string) (*VMStatusInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	info, exists := s.statusCache[vmID]
	return info, exists
}

func (s *VMSyncService) GetAllStatuses() map[string]*VMStatusInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*VMStatusInfo)
	for k, v := range s.statusCache {
		result[k] = v
	}
	return result
}

func (s *VMSyncService) ForceSyncVM(ctx context.Context, vmID string) (*VMStatusInfo, error) {
	vm, err := s.vmRepo.FindByID(ctx, vmID)
	if err != nil {
		return nil, fmt.Errorf("VM not found: %w", err)
	}

	if vm.LibvirtDomainUUID == "" || vm.LibvirtDomainUUID == "new-uuid" || vm.LibvirtDomainUUID == "defined-uuid" {
		return nil, fmt.Errorf("VM domain not defined")
	}

	s.syncSingleVM(ctx, vm)

	s.mu.RLock()
	info, exists := s.statusCache[vmID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("failed to get VM status")
	}

	return info, nil
}

func (s *VMSyncService) GetWebSocketHub() *WebSocketHub {
	return s.wsHub
}

func libvirtStateToStatus(state int) string {
	switch libvirtgo.DomainState(state) {
	case libvirtgo.DOMAIN_NOSTATE:
		return "no_state"
	case libvirtgo.DOMAIN_RUNNING:
		return "running"
	case libvirtgo.DOMAIN_BLOCKED:
		return "blocked"
	case libvirtgo.DOMAIN_PAUSED:
		return "suspended"
	case libvirtgo.DOMAIN_SHUTDOWN:
		return "shutdown"
	case libvirtgo.DOMAIN_SHUTOFF:
		return "stopped"
	case libvirtgo.DOMAIN_CRASHED:
		return "crashed"
	case libvirtgo.DOMAIN_PMSUSPENDED:
		return "suspended"
	default:
		return "unknown"
	}
}

func isTransitionalStatus(status string) bool {
	return status == "starting" || status == "stopping" || status == "creating" || status == "migrating"
}
