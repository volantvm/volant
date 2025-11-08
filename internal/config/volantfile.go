// Copyright (c) 2025 HYPR. PTE. LTD.

package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Volantfile represents the unified build + runtime configuration
type Volantfile struct {
	Image   ImageConfig   `toml:"image"`
	Build   BuildConfig   `toml:"build"`
	Runtime RuntimeConfig `toml:"runtime"`
}

type ImageConfig struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
}

type BuildConfig struct {
	Strategy string `toml:"strategy"` // dockerfile, oci-rootfs, initramfs

	// For dockerfile strategy
	Dockerfile string            `toml:"dockerfile,omitempty"`
	Context    string            `toml:"context,omitempty"`
	Target     string            `toml:"target,omitempty"`
	Args       map[string]string `toml:"args,omitempty"`

	// For oci-rootfs strategy
	Base     string   `toml:"base,omitempty"`
	Packages []string `toml:"packages,omitempty"`

	// For initramfs strategy
	BusyboxURL string            `toml:"busybox_url,omitempty"`
	Files      map[string]string `toml:"files,omitempty"`
}

type RuntimeConfig struct {
	CPUCores   int               `toml:"cpu_cores"`
	MemoryMB   int               `toml:"memory_mb"`
	Entrypoint []string          `toml:"entrypoint,omitempty"`
	WorkDir    string            `toml:"workdir,omitempty"`
	Env        map[string]string `toml:"env,omitempty"`
	Expose     ExposeConfig      `toml:"expose,omitempty"`
	Volumes    map[string]string `toml:"volumes,omitempty"`
	Health     *HealthConfig     `toml:"health,omitempty"`
}

type ExposeConfig struct {
	Ports []int              `toml:"ports,omitempty"`
	HTTP  int                `toml:"http,omitempty"`
	HTTPS int                `toml:"https,omitempty"`
	Named map[string]int     `toml:"named,omitempty"`
}

type HealthConfig struct {
	Command  []string `toml:"command"`
	Interval int      `toml:"interval"` // seconds
	Timeout  int      `toml:"timeout"`  // seconds
	Retries  int      `toml:"retries"`
}

// LoadVolantfile parses a Volantfile
func LoadVolantfile(path string) (*Volantfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read volantfile: %w", err)
	}

	var vf Volantfile
	if err := toml.Unmarshal(data, &vf); err != nil {
		return nil, fmt.Errorf("parse volantfile: %w", err)
	}

	// Set defaults
	vf.SetDefaults()

	// Validate
	if err := vf.Validate(); err != nil {
		return nil, err
	}

	return &vf, nil
}

func (vf *Volantfile) SetDefaults() {
	// Image defaults
	if vf.Image.Version == "" {
		vf.Image.Version = "latest"
	}

	// Build defaults
	if vf.Build.Context == "" {
		vf.Build.Context = "."
	}

	// Runtime defaults
	if vf.Runtime.CPUCores == 0 {
		vf.Runtime.CPUCores = 2
	}
	if vf.Runtime.MemoryMB == 0 {
		vf.Runtime.MemoryMB = 512
	}
}

func (vf *Volantfile) Validate() error {
	// Image validation
	if vf.Image.Name == "" {
		return fmt.Errorf("image.name is required")
	}

	// Build validation
	if vf.Build.Strategy == "" {
		return fmt.Errorf("build.strategy is required")
	}

	switch vf.Build.Strategy {
	case "dockerfile":
		if vf.Build.Dockerfile == "" {
			return fmt.Errorf("build.dockerfile required for dockerfile strategy")
		}
	case "oci-rootfs":
		if vf.Build.Base == "" {
			return fmt.Errorf("build.base required for oci-rootfs strategy")
		}
	case "initramfs":
		// OK
	default:
		return fmt.Errorf("unknown build.strategy: %s", vf.Build.Strategy)
	}

	// Runtime validation
	if vf.Runtime.CPUCores < 1 {
		return fmt.Errorf("runtime.cpu_cores must be >= 1")
	}
	if vf.Runtime.MemoryMB < 128 {
		return fmt.Errorf("runtime.memory_mb must be >= 128")
	}

	// Port validation
	for _, port := range vf.Runtime.Expose.Ports {
		if port < 1 || port > 65535 {
			return fmt.Errorf("port %d out of range (1-65535)", port)
		}
	}

	return nil
}

// ToFledgeConfig converts to legacy fledge format for compatibility
func (vf *Volantfile) ToFledgeConfig() map[string]interface{} {
	return map[string]interface{}{
		"version":  "1",
		"strategy": vf.Build.Strategy,
		"source": map[string]interface{}{
			"dockerfile": vf.Build.Dockerfile,
			"context":    vf.Build.Context,
			"image":      vf.Build.Base,
			"packages":   vf.Build.Packages,
		},
		"filesystem": map[string]interface{}{
			"format": "squashfs",
		},
		"agent": map[string]interface{}{
			"enabled": true,
			"port":    9999,
		},
	}
}

// ToManifestConfig converts to legacy manifest format for compatibility
func (vf *Volantfile) ToManifestConfig() map[string]interface{} {
	ports := make([]map[string]interface{}, 0)
	for _, port := range vf.Runtime.Expose.Ports {
		ports = append(ports, map[string]interface{}{
			"port":     port,
			"protocol": "tcp",
		})
	}

	return map[string]interface{}{
		"version": "1.0",
		"resources": map[string]interface{}{
			"cpu_cores": vf.Runtime.CPUCores,
			"memory_mb": vf.Runtime.MemoryMB,
		},
		"workload": map[string]interface{}{
			"entrypoint": vf.Runtime.Entrypoint,
			"workdir":    vf.Runtime.WorkDir,
			"env":        vf.Runtime.Env,
		},
		"network": map[string]interface{}{
			"expose": ports,
		},
		"storage": map[string]interface{}{
			"volumes": vf.Runtime.Volumes,
		},
	}
}

// SaveAsLegacy saves the Volantfile as legacy fledge.toml and manifest.toml
func (vf *Volantfile) SaveAsLegacy(fledgePath, manifestPath string) error {
	// Save fledge.toml
	fledgeConfig := vf.ToFledgeConfig()
	fledgeData, err := toml.Marshal(fledgeConfig)
	if err != nil {
		return fmt.Errorf("marshal fledge config: %w", err)
	}
	if err := os.WriteFile(fledgePath, fledgeData, 0644); err != nil {
		return fmt.Errorf("write fledge.toml: %w", err)
	}

	// Save manifest.toml
	manifestConfig := vf.ToManifestConfig()
	manifestData, err := toml.Marshal(manifestConfig)
	if err != nil {
		return fmt.Errorf("marshal manifest config: %w", err)
	}
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("write manifest.toml: %w", err)
	}

	return nil
}