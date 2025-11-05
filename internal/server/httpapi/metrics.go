package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/volantvm/volant/internal/metrics"
)

// getSystemMetrics returns current system resource metrics
func (a *apiServer) getSystemMetrics(c *gin.Context) {
	if a.metricsCollector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "metrics collection not enabled"})
		return
	}

	systemMetrics := a.metricsCollector.GetSystemMetrics()
	c.JSON(http.StatusOK, systemMetrics)
}

// getTimeSeriesMetrics returns historical time-series metrics
func (a *apiServer) getTimeSeriesMetrics(c *gin.Context) {
	if a.metricsCollector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "metrics collection not enabled"})
		return
	}

	// Parse duration parameter (default 1 hour)
	durationStr := c.DefaultQuery("duration", "1h")
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid duration parameter"})
		return
	}

	// Limit to max 24 hours
	if duration > 24*time.Hour {
		duration = 24 * time.Hour
	}

	timeSeriesData := a.metricsCollector.GetTimeSeriesData(duration)
	c.JSON(http.StatusOK, timeSeriesData)
}

// getVMMetrics returns per-VM resource metrics
func (a *apiServer) getVMMetrics(c *gin.Context) {
	if a.metricsCollector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "metrics collection not enabled"})
		return
	}

	vmName := c.Param("name")

	// Get VM from orchestrator
	vm, err := a.engine.GetVM(c.Request.Context(), vmName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "VM not found"})
		return
	}

	// Get metrics for this specific VM
	timeSeriesData := a.metricsCollector.GetTimeSeriesData(1 * time.Hour)

	// Find this VM's metrics
	var vmMetrics *metrics.VMMetrics
	for i := range timeSeriesData.VMMetrics {
		if timeSeriesData.VMMetrics[i].VMID == vm.ID {
			vmMetrics = &timeSeriesData.VMMetrics[i]
			break
		}
	}

	if vmMetrics == nil {
		// Return empty metrics if not found
		vmMetrics = &metrics.VMMetrics{
			VMID:          vm.ID,
			VMName:        vm.Name,
			CPUPercent:    0,
			MemoryUsedMB:  0,
			MemoryLimitMB: uint64(vm.MemoryMB),
		}
	}

	c.JSON(http.StatusOK, vmMetrics)
}

// getDriftMetrics returns driftd load balancer metrics
func (a *apiServer) getDriftMetrics(c *gin.Context) {
	if a.drift == nil || !a.drift.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "drift client not initialized"})
		return
	}

	// TODO: Implement drift metrics endpoint when driftd exposes /metrics
	c.JSON(http.StatusOK, gin.H{"message": "drift metrics endpoint not yet implemented"})
}

// getDriftHealth returns driftd health status
func (a *apiServer) getDriftHealth(c *gin.Context) {
	if a.drift == nil || !a.drift.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "drift client not initialized"})
		return
	}

	// TODO: Implement drift health check when driftd exposes /healthz
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "message": "drift health endpoint not yet implemented"})
}
