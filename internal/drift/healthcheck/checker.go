package healthcheck

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/volantvm/volant/internal/drift/dataplane"
)

// CheckType specifies the health check method.
type CheckType string

const (
	CheckTypeTCP  CheckType = "tcp"
	CheckTypeHTTP CheckType = "http"
)

// BackendTarget identifies a backend to health check.
type BackendTarget struct {
	Proto      uint8
	HostPort   uint16
	BackendIdx uint32
	IP         net.IP
	Port       uint16
	CheckType  CheckType
	HTTPPath   string // For HTTP checks, defaults to "/"
}

// Checker periodically health checks backends and updates their status.
type Checker struct {
	dp       dataplane.Interface
	interval time.Duration
	timeout  time.Duration
	logger   *slog.Logger
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.Mutex
	targets  map[string]BackendTarget // key: proto:hostport:idx
}

// New creates a health checker.
func New(dp dataplane.Interface, interval, timeout time.Duration, logger *slog.Logger) *Checker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Checker{
		dp:       dp,
		interval: interval,
		timeout:  timeout,
		logger:   logger.With("component", "healthcheck"),
		targets:  make(map[string]BackendTarget),
	}
}

// Register adds a backend target for health checking.
func (c *Checker) Register(target BackendTarget) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := targetKey(target.Proto, target.HostPort, target.BackendIdx)
	c.targets[key] = target
	c.logger.Info("backend registered for health checks",
		"proto", target.Proto,
		"host_port", target.HostPort,
		"backend_idx", target.BackendIdx,
		"check_type", target.CheckType)
}

// Unregister removes a backend from health checking.
func (c *Checker) Unregister(proto uint8, hostPort uint16, backendIdx uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := targetKey(proto, hostPort, backendIdx)
	delete(c.targets, key)
	c.logger.Info("backend unregistered from health checks",
		"proto", proto,
		"host_port", hostPort,
		"backend_idx", backendIdx)
}

// Start begins periodic health checking in the background.
func (c *Checker) Start(ctx context.Context) {
	bgCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		c.logger.Info("health checker started", "interval", c.interval)

		// Initial check immediately
		c.checkAll(bgCtx)

		for {
			select {
			case <-ticker.C:
				c.checkAll(bgCtx)
			case <-bgCtx.Done():
				c.logger.Info("health checker stopped")
				return
			}
		}
	}()
}

// Stop halts health checking.
func (c *Checker) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
}

func (c *Checker) checkAll(ctx context.Context) {
	c.mu.Lock()
	targets := make([]BackendTarget, 0, len(c.targets))
	for _, target := range c.targets {
		targets = append(targets, target)
	}
	c.mu.Unlock()

	for _, target := range targets {
		healthy := c.checkBackend(ctx, target)
		if err := c.dp.SetBackendHealth(ctx, target.Proto, target.HostPort, target.BackendIdx, healthy); err != nil {
			c.logger.Error("failed to update backend health",
				"proto", target.Proto,
				"host_port", target.HostPort,
				"backend_idx", target.BackendIdx,
				"error", err)
		} else {
			c.logger.Debug("health check completed",
				"proto", target.Proto,
				"host_port", target.HostPort,
				"backend_idx", target.BackendIdx,
				"healthy", healthy)
		}
	}
}

func (c *Checker) checkBackend(ctx context.Context, target BackendTarget) bool {
	checkCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	switch target.CheckType {
	case CheckTypeTCP:
		return c.checkTCP(checkCtx, target)
	case CheckTypeHTTP:
		return c.checkHTTP(checkCtx, target)
	default:
		c.logger.Warn("unknown check type, defaulting to TCP", "check_type", target.CheckType)
		return c.checkTCP(checkCtx, target)
	}
}

func (c *Checker) checkTCP(ctx context.Context, target BackendTarget) bool {
	addr := fmt.Sprintf("%s:%d", target.IP.String(), target.Port)
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func (c *Checker) checkHTTP(ctx context.Context, target BackendTarget) bool {
	path := target.HTTPPath
	if path == "" {
		path = "/"
	}
	url := fmt.Sprintf("http://%s:%d%s", target.IP.String(), target.Port, path)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	client := &http.Client{
		Timeout: c.timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Consider 2xx and 3xx as healthy
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

func targetKey(proto uint8, hostPort uint16, backendIdx uint32) string {
	return fmt.Sprintf("%d:%d:%d", proto, hostPort, backendIdx)
}
