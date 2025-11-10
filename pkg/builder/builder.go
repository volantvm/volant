// Copyright (c) 2025 HYPR. PTE. LTD.

package builder

import (
	"context"
	"time"
)

// Builder builds container images from Dockerfiles without requiring Docker daemon
type Builder interface {
	Build(ctx context.Context, opts BuildOptions) (*BuildResult, error)
}

// BuildOptions contains all options for building an image
type BuildOptions struct {
	// Required: Path to Dockerfile
	Dockerfile string

	// Required: Build context directory
	Context string

	// Optional: Multi-stage build target
	Target string

	// Optional: Build arguments
	BuildArgs map[string]string

	// Required: Output path for squashfs image
	OutputPath string

	// Required: Image name (e.g., "nginx")
	ImageName string

	// Required: Image version/tag (e.g., "latest", "v1.0.0")
	ImageVersion string

	// Optional: Progress reporter (defaults to ConsoleReporter)
	Reporter ProgressReporter

	// Optional: BuildKit state directory for caching
	StateDir string
}

// BuildResult contains the results of a successful build
type BuildResult struct {
	// Path to the final squashfs image
	SquashfsPath string

	// SHA256 checksum of the squashfs image
	Checksum string

	// Size of the squashfs image in bytes
	Size int64

	// Total build duration
	Duration time.Duration

	// Image format (always "squashfs")
	Format string

	// Compression ratio achieved
	CompressionRatio float64

	// Path to generated manifest (if any)
	ManifestPath string
}

// StageInfo represents a Dockerfile build stage
type StageInfo struct {
	Name     string
	Number   int
	Base     string
	Cached   bool
	Duration time.Duration
}

// BuildStats contains detailed build statistics
type BuildStats struct {
	Stages           []StageInfo
	TotalLayers      int
	CachedLayers     int
	DownloadedBytes  int64
	UnpackedSize     int64
	CompressedSize   int64
	CompressionRatio float64
	TotalDuration    time.Duration
}
