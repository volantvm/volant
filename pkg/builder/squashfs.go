// Copyright (c) 2025 HYPR. PTE. LTD.

package builder

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// CreateSquashfs compresses a rootfs directory into a squashfs image
// Returns the path, size, checksum, and compression ratio
func CreateSquashfs(ctx context.Context, rootfsDir, outputPath string, reporter ProgressReporter) (size int64, checksum string, ratio float64, err error) {
	startTime := time.Now()
	reporter.StartStage("Creating squashfs image")

	// Check if mksquashfs is available
	if _, err := exec.LookPath("mksquashfs"); err != nil {
		reporter.FailStage("Create squashfs", fmt.Errorf("mksquashfs not found: %w", err))
		return 0, "", 0, fmt.Errorf("mksquashfs not found (install with: apt install squashfs-tools): %w", err)
	}

	// Calculate uncompressed size
	uncompressedSize, err := dirSize(rootfsDir)
	if err != nil {
		reporter.FailStage("Create squashfs", err)
		return 0, "", 0, fmt.Errorf("calculate rootfs size: %w", err)
	}

	reporter.StartStep(fmt.Sprintf("Compressing %s", formatBytes(uncompressedSize)))

	// Create squashfs with gzip compression
	cmd := exec.CommandContext(ctx, "mksquashfs",
		rootfsDir,
		outputPath,
		"-comp", "gzip",
		"-no-progress", // We'll show our own progress
		"-noappend")    // Don't append to existing image

	// Run compression
	output, err := cmd.CombinedOutput()
	if err != nil {
		reporter.FailStage("Create squashfs", err)
		return 0, "", 0, fmt.Errorf("mksquashfs failed: %w\nOutput: %s", err, string(output))
	}

	// Get compressed size
	stat, err := os.Stat(outputPath)
	if err != nil {
		reporter.FailStage("Create squashfs", err)
		return 0, "", 0, fmt.Errorf("stat squashfs: %w", err)
	}

	compressedSize := stat.Size()
	compressionRatio := float64(uncompressedSize) / float64(compressedSize)

	reporter.CompleteStep(fmt.Sprintf("Compressed to %s (%.1fx)", formatBytes(compressedSize), compressionRatio))

	// Calculate checksum
	reporter.StartStep("Calculating SHA256 checksum")

	checksum, err = calculateChecksum(outputPath)
	if err != nil {
		reporter.FailStage("Create squashfs", err)
		return 0, "", 0, fmt.Errorf("calculate checksum: %w", err)
	}

	reporter.CompleteStep(fmt.Sprintf("Checksum: %s", checksum[:16]+"..."))
	reporter.CompleteStage("Create squashfs", time.Since(startTime))

	return compressedSize, checksum, compressionRatio, nil
}

// calculateChecksum computes SHA256 checksum of a file
func calculateChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}

// dirSize calculates the total size of all files in a directory recursively
func dirSize(path string) (int64, error) {
	var size int64

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}
