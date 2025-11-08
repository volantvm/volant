package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

func newDaemonCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the Volant daemon",
	}

	cmd.AddCommand(
		newDaemonStartCommand(),
		newDaemonStopCommand(),
		newDaemonStatusCommand(),
		newDaemonLogsCommand(),
	)

	return cmd
}

func newDaemonStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the Volant daemon",
		Long: `Start the Volant daemon (volantd).

This starts both the orchestrator and the L4 network switch.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if already running
			if isRunning := checkDaemon(); isRunning {
				return fmt.Errorf("daemon already running")
			}

			// Start as background service
			return startDaemonService()
		},
	}
}

func newDaemonStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the Volant daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return stopDaemonService()
		},
	}
}

func newDaemonStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check daemon status",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Try to connect to API
			client := &http.Client{Timeout: 2 * time.Second}
			resp, err := client.Get("http://localhost:8080/api/v1/health")
			if err != nil {
				fmt.Println("Daemon: stopped")
				return nil
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				fmt.Println("Daemon: running")
				fmt.Printf("  API: http://localhost:8080\n")

				// Try to get more status info
				statusResp, err := client.Get("http://localhost:8080/api/v1/status")
				if err == nil {
					defer statusResp.Body.Close()
					// Parse and display status info if available
					fmt.Printf("  Health: OK\n")
				}
			} else {
				fmt.Println("Daemon: unhealthy")
			}

			return nil
		},
	}
}

func newDaemonLogsCommand() *cobra.Command {
	var follow bool
	var lines int

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Show daemon logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			logFile := "/var/log/volant/volantd.log"

			// Check if log file exists
			if _, err := os.Stat(logFile); os.IsNotExist(err) {
				// Try systemd/journalctl
				if runtime.GOOS == "linux" {
					return showSystemdLogs(follow, lines)
				}
				// Try launchd on macOS
				if runtime.GOOS == "darwin" {
					return showLaunchdLogs(follow, lines)
				}
				return fmt.Errorf("log file not found: %s", logFile)
			}

			// Show log file
			return showLogFile(logFile, follow, lines)
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	cmd.Flags().IntVarP(&lines, "lines", "n", 50, "Number of lines to show")

	return cmd
}

func checkDaemon() bool {
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get("http://localhost:8080/api/v1/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func startDaemonService() error {
	switch runtime.GOOS {
	case "linux":
		// Try systemd first
		if _, err := exec.LookPath("systemctl"); err == nil {
			cmd := exec.Command("sudo", "systemctl", "start", "volantd")
			if err := cmd.Run(); err != nil {
				// Fallback to direct start
				return startDaemonDirect()
			}
			return nil
		}
		// Fallback to direct start
		return startDaemonDirect()

	case "darwin":
		// Try launchd
		plistPath := "/Library/LaunchDaemons/dev.volant.volantd.plist"
		if _, err := os.Stat(plistPath); err == nil {
			cmd := exec.Command("sudo", "launchctl", "load", plistPath)
			if err := cmd.Run(); err != nil {
				return startDaemonDirect()
			}
			return nil
		}
		// Fallback to direct start
		return startDaemonDirect()

	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func stopDaemonService() error {
	switch runtime.GOOS {
	case "linux":
		// Try systemd first
		if _, err := exec.LookPath("systemctl"); err == nil {
			cmd := exec.Command("sudo", "systemctl", "stop", "volantd")
			return cmd.Run()
		}
		// Fallback to pkill
		return stopDaemonDirect()

	case "darwin":
		// Try launchd
		plistPath := "/Library/LaunchDaemons/dev.volant.volantd.plist"
		if _, err := os.Stat(plistPath); err == nil {
			cmd := exec.Command("sudo", "launchctl", "unload", plistPath)
			return cmd.Run()
		}
		// Fallback to pkill
		return stopDaemonDirect()

	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func startDaemonDirect() error {
	// Find volantd binary
	volantdPath, err := exec.LookPath("volantd")
	if err != nil {
		// Try relative paths
		candidates := []string{
			"/usr/local/bin/volantd",
			"/usr/bin/volantd",
			"./bin/volantd",
		}
		found := false
		for _, path := range candidates {
			if _, err := os.Stat(path); err == nil {
				volantdPath = path
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("volantd binary not found")
		}
	}

	// Start in background
	cmd := exec.Command("sudo", volantdPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Create log directory
	os.MkdirAll("/var/log/volant", 0755)

	// Redirect output to log file
	logFile, err := os.OpenFile("/var/log/volant/volantd.log",
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer logFile.Close()

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start volantd: %w", err)
	}

	// Wait for daemon to be ready
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for daemon to start")
		case <-ticker.C:
			if checkDaemon() {
				fmt.Printf("Volant daemon started (PID %d)\n", cmd.Process.Pid)
				return nil
			}
		}
	}
}

func stopDaemonDirect() error {
	// Find volantd process
	cmd := exec.Command("pgrep", "volantd")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("volantd not running")
	}

	// Send SIGTERM
	cmd = exec.Command("sudo", "kill", "-TERM", string(output[:len(output)-1]))
	return cmd.Run()
}

func showSystemdLogs(follow bool, lines int) error {
	args := []string{"-u", "volantd", "-n", fmt.Sprintf("%d", lines)}
	if follow {
		args = append(args, "-f")
	}
	cmd := exec.Command("journalctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func showLaunchdLogs(follow bool, lines int) error {
	// macOS logs are typically in /var/log/system.log or Console.app
	logFile := "/var/log/volant/volantd.log"
	if _, err := os.Stat(logFile); err == nil {
		return showLogFile(logFile, follow, lines)
	}

	// Try using log command
	args := []string{"show", "--predicate", "process == 'volantd'", "--last", fmt.Sprintf("%dm", lines)}
	if follow {
		args = append(args, "--stream")
	}
	cmd := exec.Command("log", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func showLogFile(path string, follow bool, lines int) error {
	if follow {
		cmd := exec.Command("tail", "-f", "-n", fmt.Sprintf("%d", lines), path)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Just show last N lines
	cmd := exec.Command("tail", "-n", fmt.Sprintf("%d", lines), path)
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	fmt.Print(string(output))
	return nil
}