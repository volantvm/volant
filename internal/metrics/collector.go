package metrics

import (
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// Point represents a single time-series data point
type Point struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// SystemMetrics represents current system-wide metrics
type SystemMetrics struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	MemoryUsedMB  uint64  `json:"memory_used_mb"`
	MemoryTotalMB uint64  `json:"memory_total_mb"`
	NumGoroutines int     `json:"num_goroutines"`
	NetworkRxMB   float64 `json:"network_rx_mb"`
	NetworkTxMB   float64 `json:"network_tx_mb"`
}

// VMMetrics represents per-VM resource metrics
type VMMetrics struct {
	VMName        string  `json:"vm_name"`
	VMID          int64   `json:"vm_id"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsedMB  uint64  `json:"memory_used_mb"`
	MemoryLimitMB uint64  `json:"memory_limit_mb"`
	NetworkRxMB   float64 `json:"network_rx_mb"`
	NetworkTxMB   float64 `json:"network_tx_mb"`
}

// TimeSeriesData represents historical metrics
type TimeSeriesData struct {
	CPUHistory     []Point      `json:"cpu_history"`
	MemoryHistory  []Point      `json:"memory_history"`
	NetworkHistory []Point      `json:"network_history"`
	VMMetrics      []VMMetrics  `json:"vm_metrics"`
	LastUpdated    time.Time    `json:"last_updated"`
}

// Collector collects and stores system metrics
type Collector struct {
	mu                sync.RWMutex
	cpuHistory        []Point
	memoryHistory     []Point
	networkHistory    []Point
	vmMetrics         map[int64]*VMMetrics
	maxHistoryPoints  int
	lastNetworkStats  *net.IOCountersStat
}

// NewCollector creates a new metrics collector
func NewCollector(maxPoints int) *Collector {
	return &Collector{
		cpuHistory:       make([]Point, 0, maxPoints),
		memoryHistory:    make([]Point, 0, maxPoints),
		networkHistory:   make([]Point, 0, maxPoints),
		vmMetrics:        make(map[int64]*VMMetrics),
		maxHistoryPoints: maxPoints,
	}
}

// StartCollection begins collecting metrics at the specified interval
func (c *Collector) StartCollection(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			c.collectSystemMetrics()
		}
	}()
}

// collectSystemMetrics collects current system metrics
func (c *Collector) collectSystemMetrics() {
	now := time.Now()

	// CPU usage
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		c.addPoint(&c.cpuHistory, Point{Timestamp: now, Value: cpuPercent[0]})
	}

	// Memory usage
	memStats, err := mem.VirtualMemory()
	if err == nil {
		c.addPoint(&c.memoryHistory, Point{Timestamp: now, Value: memStats.UsedPercent})
	}

	// Network I/O
	netStats, err := net.IOCounters(false)
	if err == nil && len(netStats) > 0 {
		if c.lastNetworkStats != nil {
			// Calculate delta (MB/s)
			rxDelta := float64(netStats[0].BytesRecv-c.lastNetworkStats.BytesRecv) / 1024 / 1024
			txDelta := float64(netStats[0].BytesSent-c.lastNetworkStats.BytesSent) / 1024 / 1024
			totalDelta := rxDelta + txDelta
			c.addPoint(&c.networkHistory, Point{Timestamp: now, Value: totalDelta})
		}
		c.lastNetworkStats = &netStats[0]
	}
}

// addPoint adds a point to the history and maintains max size
func (c *Collector) addPoint(history *[]Point, point Point) {
	c.mu.Lock()
	defer c.mu.Unlock()

	*history = append(*history, point)
	if len(*history) > c.maxHistoryPoints {
		*history = (*history)[1:]
	}
}

// GetSystemMetrics returns current system metrics
func (c *Collector) GetSystemMetrics() SystemMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metrics := SystemMetrics{
		NumGoroutines: runtime.NumGoroutine(),
	}

	// Latest CPU
	if len(c.cpuHistory) > 0 {
		metrics.CPUPercent = c.cpuHistory[len(c.cpuHistory)-1].Value
	}

	// Latest Memory
	memStats, err := mem.VirtualMemory()
	if err == nil {
		metrics.MemoryPercent = memStats.UsedPercent
		metrics.MemoryUsedMB = memStats.Used / 1024 / 1024
		metrics.MemoryTotalMB = memStats.Total / 1024 / 1024
	}

	// Latest Network
	if len(c.networkHistory) > 0 {
		// Sum last 10 seconds for smoother display
		sum := 0.0
		count := 0
		for i := len(c.networkHistory) - 1; i >= 0 && count < 10; i-- {
			sum += c.networkHistory[i].Value
			count++
		}
		if count > 0 {
			avgMBps := sum / float64(count)
			metrics.NetworkRxMB = avgMBps / 2 // Approximate split
			metrics.NetworkTxMB = avgMBps / 2
		}
	}

	return metrics
}

// GetTimeSeriesData returns historical metrics
func (c *Collector) GetTimeSeriesData(duration time.Duration) TimeSeriesData {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cutoff := time.Now().Add(-duration)

	return TimeSeriesData{
		CPUHistory:     c.filterPoints(c.cpuHistory, cutoff),
		MemoryHistory:  c.filterPoints(c.memoryHistory, cutoff),
		NetworkHistory: c.filterPoints(c.networkHistory, cutoff),
		VMMetrics:      c.getVMMetricsList(),
		LastUpdated:    time.Now(),
	}
}

// filterPoints returns points after the cutoff time
func (c *Collector) filterPoints(points []Point, cutoff time.Time) []Point {
	result := make([]Point, 0)
	for _, p := range points {
		if p.Timestamp.After(cutoff) {
			result = append(result, p)
		}
	}
	return result
}

// getVMMetricsList returns a list of VM metrics
func (c *Collector) getVMMetricsList() []VMMetrics {
	result := make([]VMMetrics, 0, len(c.vmMetrics))
	for _, vm := range c.vmMetrics {
		result = append(result, *vm)
	}
	return result
}

// UpdateVMMetrics updates metrics for a specific VM
func (c *Collector) UpdateVMMetrics(vmID int64, vmName string, cpuPercent float64, memUsedMB, memLimitMB uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.vmMetrics[vmID] = &VMMetrics{
		VMID:          vmID,
		VMName:        vmName,
		CPUPercent:    cpuPercent,
		MemoryUsedMB:  memUsedMB,
		MemoryLimitMB: memLimitMB,
	}
}

// RemoveVMMetrics removes metrics for a VM (when deleted)
func (c *Collector) RemoveVMMetrics(vmID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.vmMetrics, vmID)
}
