// Copyright (c) 2025 HYPR. PTE. LTD.
//
// Business Source License 1.1
// See LICENSE file in the project root for details.

package network

import (
	"context"

	internalnetwork "github.com/volantvm/volant/internal/server/orchestrator/network"
)

// Manager re-exports the Orchestrator network manager interface.
type Manager = internalnetwork.Manager

// NoopManager exposes the noop implementation used for development environments.
type NoopManager = internalnetwork.NoopManager

// NewNoop constructs a no-op network manager.
func NewNoop() *NoopManager {
	return internalnetwork.NewNoop()
}

// PrepareTap delegates to the underlying manager implementation.
func PrepareTap(ctx context.Context, manager Manager, vmName, mac string) (string, error) {
	return manager.PrepareTap(ctx, vmName, mac)
}

// CleanupTap delegates to the underlying manager implementation.
func CleanupTap(ctx context.Context, manager Manager, tapName string) error {
	return manager.CleanupTap(ctx, tapName)
}
