//go:build !linux

// Copyright (c) 2025 HYPR. PTE. LTD.
//
// Business Source License 1.1
// See LICENSE file in the project root for details.

package network

import (
	"context"
	"fmt"
)

// BridgeManager provides a compile-time placeholder for non-Linux builds.
type BridgeManager struct{}

// NewBridgeManager returns an unsupported stub on non-Linux platforms.
func NewBridgeManager(string) (*BridgeManager, error) {
	return nil, fmt.Errorf("bridge-backed networking is only supported on linux")
}

// PrepareTap returns an unsupported error on non-Linux platforms.
func (b *BridgeManager) PrepareTap(context.Context, string, string) (string, error) {
	return "", fmt.Errorf("bridge-backed networking is only supported on linux")
}

// CleanupTap returns an unsupported error on non-Linux platforms.
func (b *BridgeManager) CleanupTap(context.Context, string) error {
	return fmt.Errorf("bridge-backed networking is only supported on linux")
}
