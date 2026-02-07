package middleware

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type Metrics struct {
	RequestsTotal     int64
	RequestsInFlight  int64
	ResponseTimeSum   float64
	ResponseTimeCount int64
	GoRoutines        int
	HeapAlloc         uint64
	StackInUse        uint64
}

var (
	metrics = &Metrics{}
	mu      sync.Mutex
)

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		atomic.AddInt64(&metrics.RequestsTotal, 1)
		atomic.AddInt64(&metrics.RequestsInFlight, 1)

		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()

		atomic.AddInt64(&metrics.RequestsInFlight, -1)
		atomic.AddInt64(&metrics.ResponseTimeCount, 1)

		mu.Lock()
		metrics.ResponseTimeSum += duration
		mu.Unlock()
	}
}

func UpdateMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics.GoRoutines = runtime.NumGoroutine()
	metrics.HeapAlloc = m.HeapAlloc / 1024 / 1024
	metrics.StackInUse = m.StackInuse / 1024 / 1024
}

func GetMetrics() Metrics {
	UpdateMetrics()
	mu.Lock()
	defer mu.Unlock()
	return *metrics
}

func MetricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		m := GetMetrics()

		avgResponseTime := float64(0)
		if m.ResponseTimeCount > 0 {
			avgResponseTime = m.ResponseTimeSum / float64(m.ResponseTimeCount)
		}

		c.String(200, "# HELP vmmanager_requests_total Total number of HTTP requests\n"+
			"# TYPE vmmanager_requests_total counter\n"+
			"vmmanager_requests_total %d\n\n"+
			"# HELP vmmanager_requests_in_flight Current number of HTTP requests\n"+
			"# TYPE vmmanager_requests_in_flight gauge\n"+
			"vmmanager_requests_in_flight %d\n\n"+
			"# HELP vmmanager_response_time_seconds Average response time in seconds\n"+
			"# TYPE vmmanager_response_time_seconds gauge\n"+
			"vmmanager_response_time_seconds %.6f\n\n"+
			"# HELP vmmanager_goroutines Number of goroutines\n"+
			"# TYPE vmmanager_goroutines gauge\n"+
			"vmmanager_goroutines %d\n\n"+
			"# HELP vmmanager_heap_alloc_bytes Heap memory allocated in MB\n"+
			"# TYPE vmmanager_heap_alloc_bytes gauge\n"+
			"vmmanager_heap_alloc_bytes %d\n\n"+
			"# HELP vmmanager_stack_in_use_bytes Stack memory in use in MB\n"+
			"# TYPE vmmanager_stack_in_use_bytes gauge\n"+
			"vmmanager_stack_in_use_bytes %d\n",
			m.RequestsTotal,
			m.RequestsInFlight,
			avgResponseTime,
			m.GoRoutines,
			m.HeapAlloc,
			m.StackInUse,
		)
	}
}
