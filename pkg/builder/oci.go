// Copyright (c) 2025 HYPR. PTE. LTD.

package builder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// ConvertOCITarToLayout converts an OCI tar archive to OCI layout directory using skopeo
// This is needed because BuildKit exports to tar, but umoci requires layout format
func ConvertOCITarToLayout(ctx context.Context, tarPath, layoutDir string, reporter ProgressReporter) error {
	startTime := time.Now()
	reporter.StartStage("Converting OCI tar to layout")

	// Check if skopeo is available
	if _, err := exec.LookPath("skopeo"); err != nil {
		reporter.FailStage("Convert OCI format", fmt.Errorf("skopeo not found: %w", err))
		return fmt.Errorf("skopeo not found (install with: apt install skopeo): %w", err)
	}

	// Create layout directory
	if err := os.MkdirAll(layoutDir, 0o755); err != nil {
		reporter.FailStage("Convert OCI format", err)
		return fmt.Errorf("create layout dir: %w", err)
	}

	reporter.StartStep("Running skopeo copy")

	// Convert: oci-archive (tar) → oci layout (directory)
	cmd := exec.CommandContext(ctx, "skopeo", "copy",
		fmt.Sprintf("oci-archive:%s", tarPath),
		fmt.Sprintf("oci:%s:latest", layoutDir))

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		reporter.FailStage("Convert OCI format", err)
		return fmt.Errorf("skopeo copy failed: %w\nOutput: %s", err, string(output))
	}

	reporter.CompleteStep("skopeo copy complete")
	reporter.CompleteStage("Convert OCI format", time.Since(startTime))

	return nil
}

// PullOCIImage pulls an OCI image from a registry using skopeo
// This allows building from existing images without Docker
func PullOCIImage(ctx context.Context, imageRef, layoutDir string, reporter ProgressReporter) error {
	startTime := time.Now()
	reporter.StartStage(fmt.Sprintf("Pulling %s", imageRef))

	// Check if skopeo is available
	if _, err := exec.LookPath("skopeo"); err != nil {
		reporter.FailStage("Pull image", fmt.Errorf("skopeo not found: %w", err))
		return fmt.Errorf("skopeo not found (install with: apt install skopeo): %w", err)
	}

	// Create layout directory
	if err := os.MkdirAll(layoutDir, 0o755); err != nil {
		reporter.FailStage("Pull image", err)
		return fmt.Errorf("create layout dir: %w", err)
	}

	reporter.StartStep(fmt.Sprintf("Pulling from registry"))

	// Pull: docker://registry/image → oci layout (directory)
	cmd := exec.CommandContext(ctx, "skopeo", "copy",
		fmt.Sprintf("docker://%s", imageRef),
		fmt.Sprintf("oci:%s:latest", layoutDir))

	// Capture output for progress
	output, err := cmd.CombinedOutput()
	if err != nil {
		reporter.FailStage("Pull image", err)
		return fmt.Errorf("skopeo pull failed: %w\nOutput: %s", err, string(output))
	}

	reporter.CompleteStep(fmt.Sprintf("Pulled %s", imageRef))
	reporter.CompleteStage(fmt.Sprintf("Pull %s", imageRef), time.Since(startTime))

	return nil
}
