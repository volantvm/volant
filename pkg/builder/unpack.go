// Copyright (c) 2025 HYPR. PTE. LTD.

package builder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// UnpackOCILayout unpacks an OCI layout to a rootfs directory using umoci
// This properly handles all OCI layers, permissions, and metadata
func UnpackOCILayout(ctx context.Context, layoutDir, rootfsDir string, reporter ProgressReporter) error {
	startTime := time.Now()
	reporter.StartStage("Unpacking OCI layers to rootfs")

	// Check if umoci is available
	if _, err := exec.LookPath("umoci"); err != nil {
		reporter.FailStage("Unpack OCI layers", fmt.Errorf("umoci not found: %w", err))
		return fmt.Errorf("umoci not found (install from: github.com/opencontainers/umoci): %w", err)
	}

	// umoci unpacks to a directory with a specific structure
	// It creates <dest-dir>/<bundle-name>/rootfs
	// We want the rootfs to end up at rootfsDir, so we need to handle this carefully

	bundleDir := filepath.Dir(rootfsDir)
	bundleName := filepath.Base(rootfsDir)

	// Remove existing bundle if it exists (umoci requires it not to exist)
	if err := os.RemoveAll(filepath.Join(bundleDir, bundleName)); err != nil && !os.IsNotExist(err) {
		reporter.FailStage("Unpack OCI layers", err)
		return fmt.Errorf("remove existing bundle: %w", err)
	}

	reporter.StartStep("Extracting layers with umoci")

	// Unpack: oci layout → bundle directory
	cmd := exec.CommandContext(ctx, "umoci", "unpack",
		"--image", fmt.Sprintf("%s:latest", layoutDir),
		filepath.Join(bundleDir, bundleName))

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		reporter.FailStage("Unpack OCI layers", err)
		return fmt.Errorf("umoci unpack failed: %w\nOutput: %s", err, string(output))
	}

	reporter.CompleteStep("OCI layers extracted")

	// umoci creates <bundle>/rootfs, but we want the rootfs directly at rootfsDir
	// So we need to move <bundle>/rootfs to rootfsDir and remove the bundle wrapper
	actualRootfs := filepath.Join(bundleDir, bundleName, "rootfs")
	if _, err := os.Stat(actualRootfs); err == nil {
		// Rename rootfs to final location
		tempRootfs := filepath.Join(bundleDir, "."+bundleName+"-rootfs")
		if err := os.Rename(actualRootfs, tempRootfs); err != nil {
			reporter.FailStage("Unpack OCI layers", err)
			return fmt.Errorf("move rootfs: %w", err)
		}

		// Remove bundle directory
		if err := os.RemoveAll(filepath.Join(bundleDir, bundleName)); err != nil {
			reporter.FailStage("Unpack OCI layers", err)
			return fmt.Errorf("remove bundle: %w", err)
		}

		// Rename temp to final
		if err := os.Rename(tempRootfs, rootfsDir); err != nil {
			reporter.FailStage("Unpack OCI layers", err)
			return fmt.Errorf("finalize rootfs: %w", err)
		}
	}

	reporter.CompleteStage("Unpack OCI layers", time.Since(startTime))

	return nil
}
