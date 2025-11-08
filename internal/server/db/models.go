// Copyright (c) 2025 HYPR. PTE. LTD.

package db

import (
	"time"
)

// Stack represents a multi-VM application stack
type Stack struct {
	ID          int64             `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Status      string            `json:"status"`
	ComposeFile string            `json:"compose_file,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	DeletedAt   *time.Time        `json:"deleted_at,omitempty"`
}

// StackVM represents the relationship between a stack and its VMs
type StackVM struct {
	ID        int64     `json:"id"`
	StackID   int64     `json:"stack_id"`
	VMID      string    `json:"vm_id"`
	Role      string    `json:"role,omitempty"`
	Ordinal   int       `json:"ordinal"`
	CreatedAt time.Time `json:"created_at"`
}

// Backward compatibility aliases (temporary - will be removed in v2.0)
type Deployment = Stack

// StackFilters for querying stacks
type StackFilters struct {
	Name   string
	Status string
	Limit  int
	Offset int
}

// StackStatus constants
const (
	StackStatusPending  = "pending"
	StackStatusRunning  = "running"
	StackStatusStopped  = "stopped"
	StackStatusFailed   = "failed"
	StackStatusUpdating = "updating"
)