// Copyright (c) 2025 HYPR. PTE. LTD.
//go:build !linux

package builder

import (
	"context"
	"fmt"
)

// EmbeddedBuildKit is not available on non-Linux platforms
type EmbeddedBuildKit struct{}

func NewEmbeddedBuildKit(stateDir string, reporter ProgressReporter) *EmbeddedBuildKit {
	return &EmbeddedBuildKit{}
}

func (b *EmbeddedBuildKit) BuildToOCI(ctx context.Context, opts BuildOptions) (string, error) {
	return "", fmt.Errorf("embedded BuildKit is only available on Linux")
}
