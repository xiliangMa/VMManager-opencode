package handlers

import (
	"net/http"
	"sync"
	"time"

	"vmmanager/internal/repository"

	"github.com/gin-gonic/gin"
)

type VMStatsHandler struct {
	vmStatsRepo *repository.VMStatsRepository
	mu          sync.RWMutex
	cache       map[string]*VMStatsCache
}

type VMStatsCache struct {
	cpuHistory     []DataPoint
	memoryHistory  []DataPoint
	diskHistory    []DataPoint
	networkHistory []DataPoint
	lastUpdate     time.Time
}

type DataPoint struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

type VMResourceStats struct {
	CPUUsage      float64     `json:"cpuUsage"`
	MemoryUsage   float64     `json:"memoryUsage"`
	DiskUsage     float64     `json:"diskUsage"`
	NetworkIn     float64     `json:"networkIn"`
	NetworkOut    float64     `json:"networkOut"`
	CPUHistory    []DataPoint `json:"cpuHistory"`
	MemoryHistory []DataPoint `json:"memoryHistory"`
	DiskHistory   []DataPoint `json:"diskHistory"`
}

type SystemResourceStats struct {
	TotalCPU       float64 `json:"totalCpu"`
	UsedCPU        float64 `json:"usedCpu"`
	CPUPercent     float64 `json:"cpuPercent"`
	TotalMemory    float64 `json:"totalMemory"`
	UsedMemory     float64 `json:"usedMemory"`
	MemoryPercent  float64 `json:"memoryPercent"`
	TotalDisk      float64 `json:"totalDisk"`
	UsedDisk       float64 `json:"usedDisk"`
	DiskPercent    float64 `json:"diskPercent"`
	VMCount        int     `json:"vmCount"`
	RunningVMCount int     `json:"runningVmCount"`
	ActiveUsers    int     `json:"activeUsers"`
}

func NewVMStatsHandler(vmStatsRepo *repository.VMStatsRepository) *VMStatsHandler {
	return &VMStatsHandler{
		vmStatsRepo: vmStatsRepo,
		cache:       make(map[string]*VMStatsCache),
	}
}

func (h *VMStatsHandler) GetVMStats(c *gin.Context) {
	vmID := c.Param("id")

	h.mu.RLock()
	cache, exists := h.cache[vmID]
	h.mu.RUnlock()

	now := time.Now()

	if !exists || time.Since(cache.lastUpdate) > 30*time.Second {
		stats := h.collectVMStats(vmID)

		h.mu.Lock()
		if !exists {
			h.cache[vmID] = &VMStatsCache{
				cpuHistory:    make([]DataPoint, 0, 60),
				memoryHistory: make([]DataPoint, 0, 60),
				diskHistory:   make([]DataPoint, 0, 60),
			}
		}
		cache = h.cache[vmID]
		cache.lastUpdate = now

		if len(cache.cpuHistory) > 60 {
			cache.cpuHistory = cache.cpuHistory[1:]
			cache.memoryHistory = cache.memoryHistory[1:]
			cache.diskHistory = cache.diskHistory[1:]
		}

		cache.cpuHistory = append(cache.cpuHistory, DataPoint{
			Timestamp: now.Format("15:04:05"),
			Value:     stats.CPUUsage,
		})
		cache.memoryHistory = append(cache.memoryHistory, DataPoint{
			Timestamp: now.Format("15:04:05"),
			Value:     stats.MemoryUsage,
		})
		cache.diskHistory = append(cache.diskHistory, DataPoint{
			Timestamp: now.Format("15:04:05"),
			Value:     stats.DiskUsage,
		})
		h.mu.Unlock()
	}

	h.mu.RLock()
	cache = h.cache[vmID]
	result := VMResourceStats{
		CPUUsage:      cache.cpuHistory[len(cache.cpuHistory)-1].Value,
		MemoryUsage:   cache.memoryHistory[len(cache.memoryHistory)-1].Value,
		DiskUsage:     cache.diskHistory[len(cache.diskHistory)-1].Value,
		CPUHistory:    cache.cpuHistory,
		MemoryHistory: cache.memoryHistory,
		DiskHistory:   cache.diskHistory,
	}
	h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": result,
	})
}

func (h *VMStatsHandler) collectVMStats(vmID string) VMResourceStats {
	return VMResourceStats{
		CPUUsage:    25.5,
		MemoryUsage: 45.2,
		DiskUsage:   12.8,
		NetworkIn:   1024.5,
		NetworkOut:  512.3,
	}
}

func (h *VMStatsHandler) GetSystemStats(c *gin.Context) {
	stats := SystemResourceStats{
		TotalCPU:       800,
		UsedCPU:        320,
		CPUPercent:     40.0,
		TotalMemory:    16384,
		UsedMemory:     6144,
		MemoryPercent:  37.5,
		TotalDisk:      512,
		UsedDisk:       128,
		DiskPercent:    25.0,
		VMCount:        10,
		RunningVMCount: 6,
		ActiveUsers:    25,
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": stats,
	})
}

func (h *VMStatsHandler) GetVMHistory(c *gin.Context) {
	vmID := c.Param("id")
	_ = c.DefaultQuery("duration", "1h")

	h.mu.RLock()
	cache, exists := h.cache[vmID]
	h.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"cpuHistory":    []DataPoint{},
				"memoryHistory": []DataPoint{},
				"diskHistory":   []DataPoint{},
			},
		})
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"cpuHistory":    cache.cpuHistory,
			"memoryHistory": cache.memoryHistory,
			"diskHistory":   cache.diskHistory,
		},
	})
}
