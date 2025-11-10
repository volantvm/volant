// Copyright (c) 2025 HYPR. PTE. LTD.

package builder

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// NativeBuilder implements the Builder interface with embedded BuildKit
type NativeBuilder struct {
	buildkit *EmbeddedBuildKit
	stateDir string
}

// NewNativeBuilder creates a new native builder with embedded BuildKit
func NewNativeBuilder(stateDir string) *NativeBuilder {
	if stateDir == "" {
		stateDir = ensureStateDir()
	}

	return &NativeBuilder{
		stateDir: stateDir,
	}
}

// Build executes the complete build pipeline with spectacular UX
//
// Pipeline:
// 1. Dockerfile → [Embedded BuildKit] → OCI tar
// 2. OCI tar → [skopeo] → OCI layout
// 3. OCI layout → [umoci] → rootfs directory
// 4. rootfs → [mksquashfs] → squashfs image
// 5. Generate manifest
// 6. Cleanup temp files
//
// NO Docker daemon required!
func (b *NativeBuilder) Build(ctx context.Context, opts BuildOptions) (*BuildResult, error) {
	buildStart := time.Now()

	// Setup reporter
	if opts.Reporter == nil {
		opts.Reporter = NewConsoleReporter(os.Stdout)
	}
	reporter := opts.Reporter

	// Print beautiful header
	printBuildHeader(reporter, opts)

	// Create temp directories for intermediate files
	workDir, err := os.MkdirTemp("", "volant-build-*")
	if err != nil {
		reporter.BuildFailed(fmt.Errorf("create work directory: %w", err))
		return nil, err
	}
	defer os.RemoveAll(workDir) // Cleanup on exit

	layoutDir := filepath.Join(workDir, "layout")
	rootfsDir := filepath.Join(workDir, "rootfs")

	// Initialize BuildKit
	if b.buildkit == nil {
		b.buildkit = NewEmbeddedBuildKit(b.stateDir, reporter)
	}

	// Step 1: Build with embedded BuildKit → OCI tar
	ociTarPath, err := b.buildkit.BuildToOCI(ctx, opts)
	if err != nil {
		reporter.BuildFailed(fmt.Errorf("buildkit build: %w", err))
		return nil, err
	}
	defer os.Remove(ociTarPath) // Cleanup OCI tar

	// Step 2: Convert OCI tar → OCI layout (skopeo)
	if err := ConvertOCITarToLayout(ctx, ociTarPath, layoutDir, reporter); err != nil {
		reporter.BuildFailed(fmt.Errorf("convert OCI format: %w", err))
		return nil, err
	}

	// Step 3: Unpack OCI layout → rootfs (umoci)
	if err := UnpackOCILayout(ctx, layoutDir, rootfsDir, reporter); err != nil {
		reporter.BuildFailed(fmt.Errorf("unpack OCI layers: %w", err))
		return nil, err
	}

	// Step 4: Compress rootfs → squashfs (mksquashfs)
	size, checksum, ratio, err := CreateSquashfs(ctx, rootfsDir, opts.OutputPath, reporter)
	if err != nil {
		reporter.BuildFailed(fmt.Errorf("create squashfs: %w", err))
		return nil, err
	}

	// Build successful!
	duration := time.Since(buildStart)

	result := &BuildResult{
		SquashfsPath:     opts.OutputPath,
		Checksum:         checksum,
		Size:             size,
		Duration:         duration,
		Format:           "squashfs",
		CompressionRatio: ratio,
	}

	// Generate manifest
	manifest, err := GenerateManifest(opts, result)
	if err != nil {
		// Non-fatal: we still have the squashfs
		reporter.StartStep(fmt.Sprintf("Warning: failed to generate manifest: %v", err))
	} else {
		// Save manifest next to squashfs
		manifestPath := opts.OutputPath + ".manifest.json"
		if err := SaveManifest(manifest, manifestPath); err != nil {
			reporter.StartStep(fmt.Sprintf("Warning: failed to save manifest: %v", err))
		} else {
			result.ManifestPath = manifestPath
		}
	}

	// Beautiful completion message
	reporter.BuildComplete(opts.OutputPath, size, duration)

	return result, nil
}

// printBuildHeader prints a beautiful header for the build
func printBuildHeader(reporter ProgressReporter, opts BuildOptions) {
	// This is called at the start to set the stage
	reporter.StartStage(fmt.Sprintf("🚀 Building %s:%s", opts.ImageName, opts.ImageVersion))
	reporter.CompleteStage(fmt.Sprintf("Build configuration loaded"), 1*time.Millisecond)
}

// BuildFromDockerfile is a convenience function to build from a Dockerfile
func BuildFromDockerfile(ctx context.Context, dockerfilePath, context, outputPath, imageName, imageVersion string) (*BuildResult, error) {
	builder := NewNativeBuilder("")

	opts := BuildOptions{
		Dockerfile:   dockerfilePath,
		Context:      context,
		OutputPath:   outputPath,
		ImageName:    imageName,
		ImageVersion: imageVersion,
		BuildArgs:    make(map[string]string),
		Reporter:     NewConsoleReporter(os.Stdout),
	}

	return builder.Build(ctx, opts)
}

// BuildFromImage is a convenience function to convert an existing OCI image
func BuildFromImage(ctx context.Context, imageRef, outputPath, imageName, imageVersion string) (*BuildResult, error) {
	buildStart := time.Now()
	reporter := NewConsoleReporter(os.Stdout)

	// Print header
	printBuildHeader(reporter, BuildOptions{ImageName: imageName, ImageVersion: imageVersion})

	// Create temp directories
	workDir, err := os.MkdirTemp("", "volant-build-*")
	if err != nil {
		reporter.BuildFailed(fmt.Errorf("create work directory: %w", err))
		return nil, err
	}
	defer os.RemoveAll(workDir)

	layoutDir := filepath.Join(workDir, "layout")
	rootfsDir := filepath.Join(workDir, "rootfs")

	// Step 1: Pull image with skopeo
	if err := PullOCIImage(ctx, imageRef, layoutDir, reporter); err != nil {
		reporter.BuildFailed(fmt.Errorf("pull image: %w", err))
		return nil, err
	}

	// Step 2: Unpack with umoci
	if err := UnpackOCILayout(ctx, layoutDir, rootfsDir, reporter); err != nil {
		reporter.BuildFailed(fmt.Errorf("unpack layers: %w", err))
		return nil, err
	}

	// Step 3: Create squashfs
	size, checksum, ratio, err := CreateSquashfs(ctx, rootfsDir, outputPath, reporter)
	if err != nil {
		reporter.BuildFailed(fmt.Errorf("create squashfs: %w", err))
		return nil, err
	}

	duration := time.Since(buildStart)

	result := &BuildResult{
		SquashfsPath:     outputPath,
		Checksum:         checksum,
		Size:             size,
		Duration:         duration,
		Format:           "squashfs",
		CompressionRatio: ratio,
	}

	// Generate and save manifest
	opts := BuildOptions{
		ImageName:    imageName,
		ImageVersion: imageVersion,
	}

	manifest, err := GenerateManifest(opts, result)
	if err == nil {
		manifestPath := outputPath + ".manifest.json"
		if err := SaveManifest(manifest, manifestPath); err == nil {
			result.ManifestPath = manifestPath
		}
	}

	reporter.BuildComplete(outputPath, size, duration)

	return result, nil
}
