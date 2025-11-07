// Copyright (c) 2025 HYPR. PTE. LTD.
//
// Business Source License 1.1
// See LICENSE file in the project root for details.

package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/volantvm/volant/internal/server/db"
)

// volumeRepository implements db.VolumeRepository
type volumeRepository struct {
	exec executor
}

var _ db.VolumeRepository = (*volumeRepository)(nil)

func (r *volumeRepository) Create(ctx context.Context, vol *db.Volume) error {
	query := `INSERT INTO volumes (name, type, size_gb, persistent, host_path, backup_path, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	backupPath := nullableString(vol.BackupPath)

	result, err := r.exec.ExecContext(ctx, query,
		vol.Name, vol.Type, vol.SizeGB, vol.Persistent,
		vol.HostPath, backupPath)
	if err != nil {
		return fmt.Errorf("insert volume: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	vol.ID = id
	return nil
}

func (r *volumeRepository) GetByName(ctx context.Context, name string) (*db.Volume, error) {
	query := `SELECT id, name, type, size_gb, persistent, host_path, backup_path, created_at, updated_at
	          FROM volumes WHERE name = ?`

	var vol db.Volume
	var backupPath sql.NullString
	var createdAt, updatedAt string

	err := r.exec.QueryRowContext(ctx, query, name).Scan(
		&vol.ID, &vol.Name, &vol.Type, &vol.SizeGB, &vol.Persistent,
		&vol.HostPath, &backupPath, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("volume not found: %s", name)
		}
		return nil, fmt.Errorf("query volume: %w", err)
	}

	if backupPath.Valid {
		vol.BackupPath = backupPath.String
	}
	vol.CreatedAt, _ = parseTimestamp(createdAt)
	vol.UpdatedAt, _ = parseTimestamp(updatedAt)

	return &vol, nil
}

func (r *volumeRepository) GetByID(ctx context.Context, id int64) (*db.Volume, error) {
	query := `SELECT id, name, type, size_gb, persistent, host_path, backup_path, created_at, updated_at
	          FROM volumes WHERE id = ?`

	var vol db.Volume
	var backupPath sql.NullString
	var createdAt, updatedAt string

	err := r.exec.QueryRowContext(ctx, query, id).Scan(
		&vol.ID, &vol.Name, &vol.Type, &vol.SizeGB, &vol.Persistent,
		&vol.HostPath, &backupPath, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("volume not found: %d", id)
		}
		return nil, fmt.Errorf("query volume: %w", err)
	}

	if backupPath.Valid {
		vol.BackupPath = backupPath.String
	}
	vol.CreatedAt, _ = parseTimestamp(createdAt)
	vol.UpdatedAt, _ = parseTimestamp(updatedAt)

	return &vol, nil
}

func (r *volumeRepository) List(ctx context.Context) ([]db.Volume, error) {
	query := `SELECT id, name, type, size_gb, persistent, host_path, backup_path, created_at, updated_at
	          FROM volumes ORDER BY created_at DESC`

	rows, err := r.exec.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query volumes: %w", err)
	}
	defer rows.Close()

	var volumes []db.Volume
	for rows.Next() {
		var vol db.Volume
		var backupPath sql.NullString
		var createdAt, updatedAt string

		if err := rows.Scan(&vol.ID, &vol.Name, &vol.Type, &vol.SizeGB, &vol.Persistent,
			&vol.HostPath, &backupPath, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan volume: %w", err)
		}

		if backupPath.Valid {
			vol.BackupPath = backupPath.String
		}
		vol.CreatedAt, _ = parseTimestamp(createdAt)
		vol.UpdatedAt, _ = parseTimestamp(updatedAt)

		volumes = append(volumes, vol)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate volumes: %w", err)
	}

	return volumes, nil
}

func (r *volumeRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM volumes WHERE id = ?`
	_, err := r.exec.ExecContext(ctx, query, id)
	return err
}

func (r *volumeRepository) Update(ctx context.Context, vol *db.Volume) error {
	query := `UPDATE volumes SET backup_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	backupPath := nullableString(vol.BackupPath)
	_, err := r.exec.ExecContext(ctx, query, backupPath, vol.ID)
	return err
}

// volumeMountRepository implements db.VolumeMountRepository
type volumeMountRepository struct {
	exec executor
}

var _ db.VolumeMountRepository = (*volumeMountRepository)(nil)

func (r *volumeMountRepository) Create(ctx context.Context, mount *db.VolumeMount) error {
	query := `INSERT INTO volume_mounts (volume_id, vm_name, mount_point, read_only, created_at)
	          VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`

	result, err := r.exec.ExecContext(ctx, query,
		mount.VolumeID, mount.VMName, mount.MountPoint, mount.ReadOnly)
	if err != nil {
		return fmt.Errorf("insert volume mount: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	mount.ID = id
	return nil
}

func (r *volumeMountRepository) ListByVolume(ctx context.Context, volumeID int64) ([]db.VolumeMount, error) {
	query := `SELECT id, volume_id, vm_name, mount_point, read_only, created_at
	          FROM volume_mounts WHERE volume_id = ?`

	rows, err := r.exec.QueryContext(ctx, query, volumeID)
	if err != nil {
		return nil, fmt.Errorf("query volume mounts: %w", err)
	}
	defer rows.Close()

	var mounts []db.VolumeMount
	for rows.Next() {
		var m db.VolumeMount
		var createdAt string

		if err := rows.Scan(&m.ID, &m.VolumeID, &m.VMName, &m.MountPoint, &m.ReadOnly, &createdAt); err != nil {
			return nil, fmt.Errorf("scan volume mount: %w", err)
		}

		m.CreatedAt, _ = parseTimestamp(createdAt)
		mounts = append(mounts, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate volume mounts: %w", err)
	}

	return mounts, nil
}

func (r *volumeMountRepository) ListByVM(ctx context.Context, vmName string) ([]db.VolumeMount, error) {
	query := `SELECT id, volume_id, vm_name, mount_point, read_only, created_at
	          FROM volume_mounts WHERE vm_name = ?`

	rows, err := r.exec.QueryContext(ctx, query, vmName)
	if err != nil {
		return nil, fmt.Errorf("query volume mounts: %w", err)
	}
	defer rows.Close()

	var mounts []db.VolumeMount
	for rows.Next() {
		var m db.VolumeMount
		var createdAt string

		if err := rows.Scan(&m.ID, &m.VolumeID, &m.VMName, &m.MountPoint, &m.ReadOnly, &createdAt); err != nil {
			return nil, fmt.Errorf("scan volume mount: %w", err)
		}

		m.CreatedAt, _ = parseTimestamp(createdAt)
		mounts = append(mounts, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate volume mounts: %w", err)
	}

	return mounts, nil
}

func (r *volumeMountRepository) DeleteByVMAndVolume(ctx context.Context, vmName string, volumeID int64) error {
	query := `DELETE FROM volume_mounts WHERE vm_name = ? AND volume_id = ?`
	_, err := r.exec.ExecContext(ctx, query, vmName, volumeID)
	return err
}

func (r *volumeMountRepository) DeleteByVM(ctx context.Context, vmName string) error {
	query := `DELETE FROM volume_mounts WHERE vm_name = ?`
	_, err := r.exec.ExecContext(ctx, query, vmName)
	return err
}
