// Copyright (c) 2025 HYPR. PTE. LTD.
//
// Business Source License 1.1
// See LICENSE file in the project root for details.

//go:build linux
// +build linux

package app

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/volantvm/volant/internal/imagespec"
	"golang.org/x/sys/unix"
)

const (
	rootMountPoint   = "/mnt/volant-root"
	oldRootPivotName = ".pivot-old"
)

var bootstrapOnce sync.Once

// isAlreadyInRootfs checks if we're already in a mounted rootfs (C init might have done it).
// This makes kestrel defensive and compatible with any kernel init.
func isAlreadyInRootfs() bool {
	// Check if a block device is mounted on / (means we're in rootfs, not initramfs)
	mounts, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return false
	}

	// Look for lines like: "/dev/vda / ext4 ..." or "/dev/vda / squashfs ..."
	// If root is mounted from a block device, we're in rootfs
	for _, line := range strings.Split(string(mounts), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == "/" {
			device := fields[0]
			// Block devices start with /dev/ (not tmpfs, proc, sysfs, etc.)
			if strings.HasPrefix(device, "/dev/") &&
				!strings.HasPrefix(device, "/dev/pts") &&
				!strings.HasPrefix(device, "/dev/shm") {
				return true
			}
			// Also check for overlay (squashfs+overlayfs case)
			if device == "overlay" {
				return true
			}
		}
	}
	return false
}

func (a *App) bootstrapPID1() error {
	// If we are not PID 1, we are not the init process. Do nothing.
	if os.Getpid() != 1 {
		a.log.Printf("pid=%d; skipping pid1 bootstrap", os.Getpid())
		return nil
	}

	if len(os.Args) > 1 && os.Args[1] == "stage2" {
		return a.enterStage2(false)
	}

	// If we're here, it means it's Stage 1. Run the full pivot logic.
	// We keep the sync.Once just in case, but the logic above prevents re-entry.
	var bootstrapErr error
	bootstrapOnce.Do(func() {
		bootstrapErr = a.bootstrapPID1Inner()
	})
	return bootstrapErr
}

func (a *App) bootstrapPID1Inner() error {
	a.log.Printf("pid1 bootstrap starting")

	if err := mountInitial(); err != nil {
		return fmt.Errorf("mount initial filesystems: %w", err)
	}

	// DEFENSE: Check if C init (or another init) already mounted rootfs for us.
	// This makes kestrel compatible with any kernel init.
	if isAlreadyInRootfs() {
		a.log.Printf("pid1 bootstrap: already in mounted rootfs (kernel init handled it)")
		return a.enterStage2(false) // false = we're in rootfs
	}

	// We're still in initramfs. Determine boot mode: auto (default), initramfs, or rootfs
	mode := resolveBootMode()
	switch mode {
	case "initramfs":
		a.log.Printf("pid1 bootstrap: volant.boot=initramfs, staying on initramfs")
		return a.enterStage2(true)
	case "rootfs":
		a.log.Printf("pid1 bootstrap: volant.boot=rootfs, mounting rootfs ourselves")
		device := resolveRootfsDevice()
		if device == "" {
			return fmt.Errorf("volant.boot=rootfs but no rootfs device detected")
		}
		if !strings.HasPrefix(device, "/dev/") {
			device = "/dev/" + device
		}
		fsType := resolveRootfsFSType()

		// Mount the rootfs (ext4, xfs, btrfs, or squashfs)
		if err := waitForDevice(device, 10*time.Second); err != nil {
			return err
		}

		// For squashfs, we need to set up overlayfs in Go
		if fsType == "squashfs" {
			if err := mountSquashfsWithOverlay(device); err != nil {
				return fmt.Errorf("mount squashfs with overlay: %w", err)
			}
		} else {
			// ext4/xfs/btrfs direct mount
			if err := mountRootfs(device, fsType); err != nil {
				return err
			}
		}
		if err := copySelfToRoot(); err != nil {
			return fmt.Errorf("pid1 bootstrap error: copy self failed: %w", err)
		}
		a.log.Printf("Handing off to switch_root to pivot and re-execute for Stage 2 (rootfs mode)...")
		err := syscall.Exec("/bin/busybox", []string{"/bin/busybox", "switch_root", rootMountPoint, "/usr/local/bin/kestrel", "stage2"}, os.Environ())
		if err != nil {
			return fmt.Errorf("switch_root exec failed: %w", err)
		}
		return nil // Unreachable
	default: // auto
		device := resolveRootfsDevice()
		if device == "" {
			a.log.Printf("pid1 bootstrap: volant.boot=auto, no rootfs device detected; staying on initramfs")
			return a.enterStage2(true)
		}
		if !strings.HasPrefix(device, "/dev/") {
			device = "/dev/" + device
		}
		fsType := resolveRootfsFSType()

		// Mount the rootfs (ext4, xfs, btrfs, or squashfs if C init didn't do it)
		a.log.Printf("pid1 bootstrap: volant.boot=auto, mounting rootfs device=%s fstype=%s", device, fsType)
		if err := waitForDevice(device, 10*time.Second); err != nil {
			return err
		}

		// For squashfs, we need to set up overlayfs in Go
		if fsType == "squashfs" {
			if err := mountSquashfsWithOverlay(device); err != nil {
				return fmt.Errorf("mount squashfs with overlay: %w", err)
			}
		} else {
			// ext4/xfs/btrfs direct mount
			if err := mountRootfs(device, fsType); err != nil {
				return err
			}
		}
		if err := copySelfToRoot(); err != nil {
			return fmt.Errorf("pid1 bootstrap error: copy self failed: %w", err)
		}
		a.log.Printf("Handing off to switch_root to pivot and re-execute for Stage 2 (auto mode)...")
		err := syscall.Exec("/bin/busybox", []string{"/bin/busybox", "switch_root", rootMountPoint, "/usr/local/bin/kestrel", "stage2"}, os.Environ())
		if err != nil {
			return fmt.Errorf("switch_root exec failed: %w", err)
		}
		return nil // Unreachable
	}
}

func (a *App) enterStage2(fromInitramfs bool) error {
	if fromInitramfs {
		a.log.Printf("PID 1, initramfs mode: bootstrap complete without rootfs pivot")
	} else {
		a.log.Printf("PID 1, Stage 2: bootstrap complete. Starting workload.")
	}

	if err := mountEssential(); err != nil {
		a.log.Printf("FATAL: Stage 2 mount essentials failed: %v", err)
		select {}
	}

	if err := ensureConsoleTTY(a.log); err != nil {
		a.log.Printf("warning: console setup failed: %v", err)
	}

	if err := ensureDBusDaemon(a.log); err != nil {
		a.log.Printf("warning: %v", err)
	}

	if fromInitramfs {
		a.log.Printf("PID 1, initramfs mode: reinforcing in-place mounts...")
	} else {
		a.log.Printf("PID 1, Stage 2: reinforcing mounts for child processes...")
	}

	if err := unix.Mount("/proc", "/proc", "", unix.MS_BIND|unix.MS_REC, ""); err != nil && !errors.Is(err, unix.EBUSY) {
		a.log.Printf("FATAL: Stage 2 reinforcing /proc mount failed: %v", err)
		select {}
	}
	if err := unix.Mount("/sys", "/sys", "", unix.MS_BIND|unix.MS_REC, ""); err != nil && !errors.Is(err, unix.EBUSY) {
		a.log.Printf("FATAL: Stage 2 reinforcing /sys mount failed: %v", err)
		select {}
	}

	go reapZombies()
	go handleSignals(a)

	return nil
}

func mountInitial() error {
	toMount := []struct {
		source string
		target string
		fs     string
	}{
		{"proc", "/proc", "proc"},
		{"sysfs", "/sys", "sysfs"},
		{"devtmpfs", "/dev", "devtmpfs"},
	}
	for _, m := range toMount {
		if err := os.MkdirAll(m.target, 0o755); err != nil {
			return err
		}
		if err := unix.Mount(m.source, m.target, m.fs, 0, ""); err != nil && !errors.Is(err, unix.EBUSY) {
			return fmt.Errorf("mount %s on %s: %w", m.source, m.target, err)
		}
	}
	return nil
}

func resolveRootfsDevice() string {
	if value := cmdlineValue(imagespec.RootFSDeviceKey); value != "" {
		return value
	}
	for _, candidate := range []string{"vda", "vdb", "sda", "sdb"} {
		if _, err := os.Stat("/dev/" + candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func resolveRootfsFSType() string {
	if value := cmdlineValue(imagespec.RootFSFSTypeKey); value != "" {
		return value
	}
	// Default to squashfs (new standard), fallback to ext4 for legacy
	return "squashfs"
}

func resolveBootMode() string {
	mode := strings.ToLower(strings.TrimSpace(cmdlineValue(imagespec.BootModeKey)))
	switch mode {
	case "initramfs", "rootfs":
		return mode
	default:
		return "auto"
	}
}

func cmdlineValue(key string) string {
	f, err := os.Open("/proc/cmdline")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		for _, field := range strings.Fields(scanner.Text()) {
			if strings.HasPrefix(field, key+"=") {
				return strings.TrimPrefix(field, key+"=")
			}
		}
	}
	return ""
}

func waitForDevice(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("device %s not found", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func mountRootfs(device, fsType string) error {
	if err := os.MkdirAll(rootMountPoint, 0o755); err != nil {
		return err
	}
	if err := unix.Mount(device, rootMountPoint, fsType, unix.MS_RELATIME, ""); err != nil {
		return fmt.Errorf("mount rootfs %s on %s: %w", device, rootMountPoint, err)
	}
	return nil
}

// loadKernelModule attempts to load a kernel module by name.
// Returns nil if the module is already loaded or successfully loaded.
func loadKernelModule(moduleName string) error {
	// Check if module is already loaded
	modules, err := os.ReadFile("/proc/modules")
	if err == nil && strings.Contains(string(modules), moduleName) {
		return nil // Already loaded
	}

	// Try to load using modprobe (if available)
	if err := exec.Command("modprobe", moduleName).Run(); err == nil {
		return nil
	}

	// Fallback: Try to find and load .ko file directly
	paths := []string{
		"/lib/modules/" + moduleName + ".ko",
		"/lib/modules/" + moduleName + ".ko.gz",
		"/lib/modules/" + moduleName + ".ko.xz",
		"/lib/modules/" + moduleName + ".ko.zst",
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// Use finit_module syscall (313 on x86_64, 379 on arm64)
		if err := unix.FinitModule(0, "", 0); err == nil {
			// Try init_module as fallback
			if err := unix.InitModule(data, ""); err != nil && !errors.Is(err, unix.EEXIST) {
				continue
			}
			return nil
		}
	}

	// Module not found or failed to load - not necessarily fatal if built into kernel
	return nil
}

// mountSquashfsWithOverlay mounts a squashfs device with overlayfs on top for writes.
// This provides Docker-like layered filesystem support.
func mountSquashfsWithOverlay(device string) error {
	// Load required kernel modules (they may be built into kernel, so errors are non-fatal)
	_ = loadKernelModule("squashfs")
	_ = loadKernelModule("overlay")

	// Get overlay size from kernel cmdline (default 1G)
	overlaySize := cmdlineValue("overlay_size")
	if overlaySize == "" {
		overlaySize = "1G"
	}

	// Create mount points
	lowerDir := "/mnt/squashfs-lower"
	upperDir := "/mnt/squashfs-upper"

	if err := os.MkdirAll(lowerDir, 0o755); err != nil {
		return fmt.Errorf("create lower dir: %w", err)
	}
	if err := os.MkdirAll(upperDir, 0o755); err != nil {
		return fmt.Errorf("create upper dir: %w", err)
	}
	if err := os.MkdirAll(rootMountPoint, 0o755); err != nil {
		return fmt.Errorf("create root mount point: %w", err)
	}

	// Mount squashfs as lower layer (read-only)
	if err := unix.Mount(device, lowerDir, "squashfs", unix.MS_RDONLY, ""); err != nil {
		return fmt.Errorf("mount squashfs: %w", err)
	}

	// Mount tmpfs for upper layer (writable)
	tmpfsOpts := "size=" + overlaySize
	if err := unix.Mount("tmpfs", upperDir, "tmpfs", 0, tmpfsOpts); err != nil {
		unix.Unmount(lowerDir, 0)
		return fmt.Errorf("mount tmpfs upper: %w", err)
	}

	// Create work directory inside tmpfs (overlayfs requirement)
	workDir := filepath.Join(upperDir, "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		unix.Unmount(upperDir, 0)
		unix.Unmount(lowerDir, 0)
		return fmt.Errorf("create work dir: %w", err)
	}

	// Mount overlayfs combining lower and upper
	overlayOpts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerDir, upperDir, workDir)
	if err := unix.Mount("overlay", rootMountPoint, "overlay", 0, overlayOpts); err != nil {
		unix.Unmount(upperDir, 0)
		unix.Unmount(lowerDir, 0)
		return fmt.Errorf("mount overlayfs: %w", err)
	}

	return nil
}

func copySelfToRoot() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	src, err := os.Open(self)
	if err != nil {
		return err
	}
	defer src.Close()

	destPath := filepath.Join(rootMountPoint, "usr", "local", "bin", "kestrel")
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	tmpPath := destPath + ".tmp"
	dest, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dest, src); err != nil {
		dest.Close()
		return err
	}
	if err := dest.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		return err
	}
	syscall.Sync()
	return nil
}

func mountEssential() error {
	mounts := []struct {
		source string
		target string
		fs     string
		flags  uintptr
		data   string
		perm   os.FileMode
	}{
		{"proc", "/proc", "proc", 0, "", 0o755},
		{"sysfs", "/sys", "sysfs", 0, "", 0o755},
		{"devtmpfs", "/dev", "devtmpfs", 0, "mode=0755", 0o755},
		{"devpts", "/dev/pts", "devpts", 0, "mode=0620,gid=5", 0o755},
		{"shm", "/dev/shm", "tmpfs", 0, "mode=1777", 0o1777},
		{"run", "/run", "tmpfs", 0, "", 0o755},
		{"tmp", "/tmp", "tmpfs", 0, "", 0o1777},
	}
	for _, m := range mounts {
		if err := os.MkdirAll(m.target, m.perm); err != nil {
			return err
		}
		if m.perm == 0o1777 {
			if err := os.Chmod(m.target, m.perm); err != nil {
				return err
			}
		}
		if err := unix.Mount(m.source, m.target, m.fs, m.flags, m.data); err != nil && !errors.Is(err, unix.EBUSY) {
			return fmt.Errorf("mount %s on %s: %w", m.source, m.target, err)
		}
	}
	return nil
}

func reapZombies() {
	for {
		_, _ = syscall.Wait4(-1, nil, 0, nil)
	}
}

func ensureConsoleTTY(logger *log.Logger) error {
	const consoleTTY = "/dev/ttyS0"

	if _, err := os.Lstat("/dev/console"); err != nil {
		if os.IsNotExist(err) {
			if err := os.Symlink("ttyS0", "/dev/console"); err != nil && !os.IsExist(err) {
				return fmt.Errorf("create /dev/console symlink: %w", err)
			}
		} else {
			return fmt.Errorf("stat /dev/console: %w", err)
		}
	}

	tty, err := os.OpenFile(consoleTTY, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open %s: %w", consoleTTY, err)
	}
	defer tty.Close()

	if err := unix.IoctlSetPointerInt(int(tty.Fd()), unix.TIOCSCTTY, 0); err != nil {
		if errno, ok := err.(syscall.Errno); ok {
			if errno != syscall.EPERM && errno != syscall.EINVAL {
				return fmt.Errorf("set controlling tty: %w", err)
			}
		} else {
			return fmt.Errorf("set controlling tty: %w", err)
		}
	}

	for _, fd := range []int{0, 1, 2} {
		if err := unix.Dup2(int(tty.Fd()), fd); err != nil {
			return fmt.Errorf("dup2 tty to fd %d: %w", fd, err)
		}
	}

	logger.Printf("console %s configured as controlling terminal", consoleTTY)
	return nil
}

func ensureDBusDaemon(logger *log.Logger) error {
	const daemonPath = "/usr/bin/dbus-daemon"
	const socketPath = "/run/dbus/system_bus_socket"

	info, err := os.Stat(daemonPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat %s: %w", daemonPath, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s exists but is not a regular file", daemonPath)
	}

	if _, err := os.Stat(socketPath); err == nil {
		return nil
	}

	if err := os.MkdirAll("/run/dbus", 0o755); err != nil {
		return fmt.Errorf("prepare /run/dbus: %w", err)
	}

	cmd := exec.Command(daemonPath, "--system", "--fork", "--nopidfile")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("start dbus-daemon: %w", err)
	}

	if err := waitForDevice(socketPath, 3*time.Second); err != nil {
		return fmt.Errorf("wait for dbus socket: %w", err)
	}

	logger.Printf("dbus-daemon started for system bus")
	return nil
}

func handleSignals(a *App) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	a.log.Printf("received signal %s, powering off", sig)
	_ = syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
}
