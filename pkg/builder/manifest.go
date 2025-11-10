// Copyright (c) 2025 HYPR. PTE. LTD.

package builder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ImageManifest represents a Volant image manifest
type ImageManifest struct {
	SchemaVersion string           `json:"schema_version"`
	Name          string           `json:"name"`
	Version       string           `json:"version"`
	Runtime       string           `json:"runtime"`
	Enabled       bool             `json:"enabled"`
	RootFS        *RootFS          `json:"rootfs,omitempty"`
	Resources     *ResourceConfig  `json:"resources,omitempty"`
	Network       *NetworkConfig   `json:"network,omitempty"`
}

// RootFS configuration
type RootFS struct {
	URL      string `json:"url"`
	Checksum string `json:"checksum,omitempty"`
	Format   string `json:"format,omitempty"`
}

// ResourceConfig defines default resources
type ResourceConfig struct {
	CPUCores int `json:"cpu_cores"`
	MemoryMB int `json:"memory_mb"`
}

// NetworkConfig defines network settings
type NetworkConfig struct {
	Mode string `json:"mode"`
}

// GenerateManifest creates a Volant image manifest from build results
func GenerateManifest(opts BuildOptions, result *BuildResult) (*ImageManifest, error) {
	// Create base manifest
	manifest := &ImageManifest{
		SchemaVersion: "v1",
		Name:          opts.ImageName,
		Version:       opts.ImageVersion,
		Runtime:       opts.ImageName, // Default runtime name to image name
		Enabled:       true,
	}

	// Add rootfs configuration
	manifest.RootFS = &RootFS{
		URL:      fmt.Sprintf("file://%s", result.SquashfsPath),
		Checksum: result.Checksum,
		Format:   "squashfs",
	}

	// Add default resources (can be overridden at VM creation time)
	manifest.Resources = &ResourceConfig{
		CPUCores: 2,    // Default: 2 cores
		MemoryMB: 2048, // Default: 2GB RAM
	}

	// Add default network configuration
	manifest.Network = &NetworkConfig{
		Mode: "bridged",
	}

	// TODO: Extract workload configuration from Dockerfile/Volantfile
	// For now, users can override at VM creation or via Volantfile

	return manifest, nil
}

// SaveManifest saves a manifest to a JSON file
func SaveManifest(manifest *ImageManifest, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create manifest dir: %w", err)
	}

	// Marshal to pretty JSON
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}
