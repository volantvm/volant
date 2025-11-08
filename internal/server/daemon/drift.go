package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

type DriftManager struct {
	cmd     *exec.Cmd
	logFile *os.File
}

func NewDriftManager() *DriftManager {
	return &DriftManager{}
}

func (dm *DriftManager) Start(ctx context.Context) error {
	// 1. Find drift binary
	driftPath, err := dm.findDriftBinary()
	if err != nil {
		slog.Warn("Drift binary not found, falling back to vsock proxy", "error", err)
		return nil // Graceful fallback
	}

	// 2. Setup logging
	logDir := "/var/log/volant"
	os.MkdirAll(logDir, 0755)

	logPath := filepath.Join(logDir, "drift.log")
	dm.logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open drift log: %w", err)
	}

	// 3. Start drift process
	dm.cmd = exec.CommandContext(ctx, driftPath,
		"--log-level", "info",
		"--tap-device", "volant0",
	)

	dm.cmd.Stdout = dm.logFile
	dm.cmd.Stderr = dm.logFile

	// Run in new process group for clean shutdown
	dm.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := dm.cmd.Start(); err != nil {
		return fmt.Errorf("start drift: %w", err)
	}

	slog.Info("Started drift daemon", "pid", dm.cmd.Process.Pid)

	// 4. Health check
	if err := dm.waitForReady(ctx); err != nil {
		dm.Stop()
		return fmt.Errorf("drift health check failed: %w", err)
	}

	return nil
}

func (dm *DriftManager) findDriftBinary() (string, error) {
	// Check common locations
	candidates := []string{
		"/usr/local/bin/driftd",
		"/usr/bin/driftd",
		"./bin/driftd",
		"../drift/bin/driftd",
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Try PATH
	if path, err := exec.LookPath("driftd"); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("drift binary not found")
}

func (dm *DriftManager) waitForReady(ctx context.Context) error {
	// Wait for drift to be ready (check tap device exists)
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for drift")
		case <-ticker.C:
			if dm.isReady() {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (dm *DriftManager) isReady() bool {
	// Check if tap device exists
	_, err := os.Stat("/sys/class/net/volant0")
	return err == nil
}

func (dm *DriftManager) Stop() error {
	if dm.cmd == nil || dm.cmd.Process == nil {
		return nil
	}

	slog.Info("Stopping drift daemon", "pid", dm.cmd.Process.Pid)

	// Send SIGTERM for graceful shutdown
	if err := dm.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return err
	}

	// Wait up to 5 seconds
	done := make(chan error, 1)
	go func() {
		done <- dm.cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		// Force kill
		slog.Warn("Drift didn't stop gracefully, killing")
		dm.cmd.Process.Kill()
	case err := <-done:
		if err != nil && err.Error() != "signal: terminated" {
			return err
		}
	}

	if dm.logFile != nil {
		dm.logFile.Close()
	}

	slog.Info("Drift daemon stopped")
	return nil
}

func (dm *DriftManager) Status() string {
	if dm.cmd == nil || dm.cmd.Process == nil {
		return "stopped"
	}

	// Check if process is still running
	if err := dm.cmd.Process.Signal(syscall.Signal(0)); err != nil {
		return "stopped"
	}

	return "running"
}