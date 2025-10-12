//go:build linux

// Copyright (c) 2025 HYPR. PTE. LTD.
//
// Business Source License 1.1
// See LICENSE file in the project root for details.

package network

import (
	"context"

	internalnetwork "github.com/volantvm/volant/internal/server/orchestrator/network"
)

// BridgeManager wraps the orchestrator's bridge-backed implementation for external use.
type BridgeManager struct {
	inner *internalnetwork.BridgeManager
}

// NewBridgeManager constructs a bridge-backed network manager using the orchestrator implementation.
func NewBridgeManager(bridge string) (*BridgeManager, error) {
	return &BridgeManager{inner: internalnetwork.NewBridgeManager(bridge)}, nil
}

// PrepareTap satisfies the Manager interface by delegating to the inner implementation.
func (b *BridgeManager) PrepareTap(ctx context.Context, vmName, mac string) (string, error) {
	return b.inner.PrepareTap(ctx, vmName, mac)
}

// CleanupTap satisfies the Manager interface by delegating to the inner implementation.
func (b *BridgeManager) CleanupTap(ctx context.Context, tapName string) error {
	return b.inner.CleanupTap(ctx, tapName)
}
