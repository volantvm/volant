// Copyright (c) 2025 HYPR. PTE. LTD.

package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/volantvm/volant/internal/server/db"
)

// CreateStack creates a new stack in the database
func (d *DB) CreateStack(ctx context.Context, stack *db.Stack) error {
	envJSON, err := json.Marshal(stack.Environment)
	if err != nil {
		return fmt.Errorf("marshal environment: %w", err)
	}

	query := `
		INSERT INTO stacks (name, description, status, compose_file, environment, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := d.db.ExecContext(ctx, query,
		stack.Name,
		stack.Description,
		stack.Status,
		stack.ComposeFile,
		string(envJSON),
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert stack: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get insert id: %w", err)
	}

	stack.ID = id
	stack.CreatedAt = now
	stack.UpdatedAt = now

	return nil
}

// GetStack retrieves a stack by name
func (d *DB) GetStack(ctx context.Context, name string) (*db.Stack, error) {
	query := `
		SELECT id, name, description, status, compose_file, environment, created_at, updated_at, deleted_at
		FROM stacks
		WHERE name = ? AND deleted_at IS NULL
	`

	var stack db.Stack
	var envJSON string
	var deletedAt sql.NullTime

	err := d.db.QueryRowContext(ctx, query, name).Scan(
		&stack.ID,
		&stack.Name,
		&stack.Description,
		&stack.Status,
		&stack.ComposeFile,
		&envJSON,
		&stack.CreatedAt,
		&stack.UpdatedAt,
		&deletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("stack not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("query stack: %w", err)
	}

	if envJSON != "" {
		if err := json.Unmarshal([]byte(envJSON), &stack.Environment); err != nil {
			return nil, fmt.Errorf("unmarshal environment: %w", err)
		}
	}

	if deletedAt.Valid {
		stack.DeletedAt = &deletedAt.Time
	}

	return &stack, nil
}

// ListStacks returns all non-deleted stacks matching the filters
func (d *DB) ListStacks(ctx context.Context, filters db.StackFilters) ([]*db.Stack, error) {
	query := `
		SELECT id, name, description, status, compose_file, environment, created_at, updated_at
		FROM stacks
		WHERE deleted_at IS NULL
	`

	args := []interface{}{}

	if filters.Name != "" {
		query += " AND name LIKE ?"
		args = append(args, "%"+filters.Name+"%")
	}

	if filters.Status != "" {
		query += " AND status = ?"
		args = append(args, filters.Status)
	}

	query += " ORDER BY created_at DESC"

	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
	}

	if filters.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filters.Offset)
	}

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query stacks: %w", err)
	}
	defer rows.Close()

	var stacks []*db.Stack
	for rows.Next() {
		var stack db.Stack
		var envJSON string

		err := rows.Scan(
			&stack.ID,
			&stack.Name,
			&stack.Description,
			&stack.Status,
			&stack.ComposeFile,
			&envJSON,
			&stack.CreatedAt,
			&stack.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan stack: %w", err)
		}

		if envJSON != "" {
			if err := json.Unmarshal([]byte(envJSON), &stack.Environment); err != nil {
				return nil, fmt.Errorf("unmarshal environment: %w", err)
			}
		}

		stacks = append(stacks, &stack)
	}

	return stacks, nil
}

// UpdateStack updates a stack's properties
func (d *DB) UpdateStack(ctx context.Context, name string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	query := "UPDATE stacks SET updated_at = ?"
	args := []interface{}{time.Now()}

	for key, value := range updates {
		switch key {
		case "status":
			query += ", status = ?"
		case "description":
			query += ", description = ?"
		case "compose_file":
			query += ", compose_file = ?"
		case "environment":
			envJSON, err := json.Marshal(value)
			if err != nil {
				return fmt.Errorf("marshal environment: %w", err)
			}
			value = string(envJSON)
			query += ", environment = ?"
		default:
			continue
		}
		args = append(args, value)
	}

	query += " WHERE name = ? AND deleted_at IS NULL"
	args = append(args, name)

	result, err := d.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update stack: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("stack not found: %s", name)
	}

	return nil
}

// DeleteStack marks a stack as deleted (soft delete)
func (d *DB) DeleteStack(ctx context.Context, name string) error {
	query := `
		UPDATE stacks
		SET deleted_at = ?, updated_at = ?
		WHERE name = ? AND deleted_at IS NULL
	`

	now := time.Now()
	result, err := d.db.ExecContext(ctx, query, now, now, name)
	if err != nil {
		return fmt.Errorf("delete stack: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("stack not found: %s", name)
	}

	return nil
}

// AddStackVM associates a VM with a stack
func (d *DB) AddStackVM(ctx context.Context, stackID int64, vmID string, role string, ordinal int) error {
	query := `
		INSERT INTO stack_vms (stack_id, vm_id, role, ordinal, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := d.db.ExecContext(ctx, query, stackID, vmID, role, ordinal, time.Now())
	if err != nil {
		return fmt.Errorf("add stack vm: %w", err)
	}

	return nil
}

// GetStackVMs returns all VMs associated with a stack
func (d *DB) GetStackVMs(ctx context.Context, stackID int64) ([]*db.StackVM, error) {
	query := `
		SELECT id, stack_id, vm_id, role, ordinal, created_at
		FROM stack_vms
		WHERE stack_id = ?
		ORDER BY ordinal, created_at
	`

	rows, err := d.db.QueryContext(ctx, query, stackID)
	if err != nil {
		return nil, fmt.Errorf("query stack vms: %w", err)
	}
	defer rows.Close()

	var vms []*db.StackVM
	for rows.Next() {
		var vm db.StackVM
		err := rows.Scan(&vm.ID, &vm.StackID, &vm.VMID, &vm.Role, &vm.Ordinal, &vm.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan stack vm: %w", err)
		}
		vms = append(vms, &vm)
	}

	return vms, nil
}

// RemoveStackVM removes a VM association from a stack
func (d *DB) RemoveStackVM(ctx context.Context, stackID int64, vmID string) error {
	query := `DELETE FROM stack_vms WHERE stack_id = ? AND vm_id = ?`

	result, err := d.db.ExecContext(ctx, query, stackID, vmID)
	if err != nil {
		return fmt.Errorf("remove stack vm: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("stack vm association not found")
	}

	return nil
}