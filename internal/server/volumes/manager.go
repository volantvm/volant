// Copyright (c) 2025 HYPR. PTE. LTD.
//
// Business Source License 1.1
// See LICENSE file in the project root for details.

package volumes

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/volantvm/volant/internal/server/db"
)

const (
	VolumeTypeExt4     = "ext4"
	VolumeTypeSquashfs = "squashfs"
	VolumeTypeBind     = "bind"
)

type Manager struct {
	basePath string // /var/lib/volant/volumes
	store    db.Store
}

func NewManager(basePath string, store db.Store) *Manager {
	return &Manager{
		basePath: basePath,
		store:    store,
	}
}

func (m *Manager) CreateVolume(ctx context.Context, vol *db.Volume) error {
	// 1. Validate
	if vol.Name == "" {
		return fmt.Errorf("volume name required")
	}
	if vol.SizeGB <= 0 && vol.Type != VolumeTypeBind {
		return fmt.Errorf("volume size must be > 0")
	}

	// 2. Set host path
	if vol.Type == VolumeTypeBind {
		// For bind mounts, HostPath should already be set
		if vol.HostPath == "" {
			return fmt.Errorf("bind mount requires host_path")
		}
	} else {
		vol.HostPath = filepath.Join(m.basePath, vol.Name+".img")
	}

	// 3. Create volume file
	switch vol.Type {
	case VolumeTypeExt4:
		if err := m.createExt4Volume(vol); err != nil {
			return fmt.Errorf("failed to create ext4 volume: %w", err)
		}
	case VolumeTypeSquashfs:
		return fmt.Errorf("squashfs volumes are read-only, use bind mounts instead")
	case VolumeTypeBind:
		// Bind mounts don't need creation, just path validation
		if _, err := os.Stat(vol.HostPath); err != nil {
			return fmt.Errorf("bind mount path does not exist: %s", vol.HostPath)
		}
	default:
		return fmt.Errorf("unsupported volume type: %s", vol.Type)
	}

	// 4. Store in database
	return m.store.Queries().Volumes().Create(ctx, vol)
}

func (m *Manager) createExt4Volume(vol *db.Volume) error {
	// Ensure base path exists
	if err := os.MkdirAll(m.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create volume directory: %w", err)
	}

	// 1. Allocate sparse file
	sizeMB := vol.SizeGB * 1024
	cmd := exec.Command("dd",
		"if=/dev/zero",
		"of="+vol.HostPath,
		"bs=1M",
		"count=0",
		"seek="+strconv.Itoa(sizeMB),
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("dd failed: %w", err)
	}

	// 2. Create ext4 filesystem
	cmd = exec.Command("mkfs.ext4", "-F", vol.HostPath)
	if err := cmd.Run(); err != nil {
		os.Remove(vol.HostPath) // Cleanup
		return fmt.Errorf("mkfs.ext4 failed: %w", err)
	}

	return nil
}

func (m *Manager) DeleteVolume(ctx context.Context, name string) error {
	vol, err := m.store.Queries().Volumes().GetByName(ctx, name)
	if err != nil {
		return fmt.Errorf("volume not found: %w", err)
	}

	// 1. Check if attached to any VMs
	mounts, err := m.store.Queries().VolumeMounts().ListByVolume(ctx, vol.ID)
	if err != nil {
		return err
	}
	if len(mounts) > 0 {
		return fmt.Errorf("volume is attached to %d VM(s), detach first", len(mounts))
	}

	// 2. Delete file
	if vol.Type != VolumeTypeBind {
		if err := os.Remove(vol.HostPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete volume file: %w", err)
		}
	}

	// 3. Delete from database
	return m.store.Queries().Volumes().Delete(ctx, vol.ID)
}

func (m *Manager) AttachVolume(ctx context.Context, vmName, volumeName, mountPoint string, readOnly bool) error {
	// 1. Get volume
	vol, err := m.store.Queries().Volumes().GetByName(ctx, volumeName)
	if err != nil {
		return fmt.Errorf("volume not found: %w", err)
	}

	// 2. Get VM
	vm, err := m.store.Queries().VirtualMachines().GetByName(ctx, vmName)
	if err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	// 3. Store mount
	mount := &db.VolumeMount{
		VolumeID:   vol.ID,
		VMName:     vm.Name,
		MountPoint: mountPoint,
		ReadOnly:   readOnly,
	}
	return m.store.Queries().VolumeMounts().Create(ctx, mount)
}

func (m *Manager) DetachVolume(ctx context.Context, vmName, volumeName string) error {
	vol, err := m.store.Queries().Volumes().GetByName(ctx, volumeName)
	if err != nil {
		return fmt.Errorf("volume not found: %w", err)
	}

	return m.store.Queries().VolumeMounts().DeleteByVMAndVolume(ctx, vmName, vol.ID)
}

func (m *Manager) BackupVolume(ctx context.Context, name string) error {
	vol, err := m.store.Queries().Volumes().GetByName(ctx, name)
	if err != nil {
		return fmt.Errorf("volume not found: %w", err)
	}

	if vol.Type == VolumeTypeBind {
		return fmt.Errorf("bind mounts cannot be backed up")
	}

	backupPath := vol.HostPath + ".backup-" + fmt.Sprintf("%d", time.Now().Unix())

	// Snapshot + compress
	cmd := exec.Command("tar", "czf", backupPath, "-C", filepath.Dir(vol.HostPath), filepath.Base(vol.HostPath))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	vol.BackupPath = backupPath
	return m.store.Queries().Volumes().Update(ctx, vol)
}

func (m *Manager) ListVolumes(ctx context.Context) ([]db.Volume, error) {
	return m.store.Queries().Volumes().List(ctx)
}

func (m *Manager) GetVolume(ctx context.Context, name string) (*db.Volume, error) {
	return m.store.Queries().Volumes().GetByName(ctx, name)
}

func (m *Manager) GetVolumeMounts(ctx context.Context, vmName string) ([]db.VolumeMount, error) {
	return m.store.Queries().VolumeMounts().ListByVM(ctx, vmName)
}
