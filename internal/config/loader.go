// Copyright (c) 2025 HYPR. PTE. LTD.

package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// LoadBuildConfig loads build configuration with fallback chain:
// 1. Volantfile (preferred)
// 2. fledge.toml (legacy)
func LoadBuildConfig(dir string) (*Volantfile, error) {
	// Try Volantfile first
	volantfilePath := filepath.Join(dir, "Volantfile")
	if _, err := os.Stat(volantfilePath); err == nil {
		return LoadVolantfile(volantfilePath)
	}

	// Fallback to legacy fledge.toml
	fledgePath := filepath.Join(dir, "fledge.toml")
	if _, err := os.Stat(fledgePath); err == nil {
		log.Printf("⚠️  Using legacy fledge.toml - consider migrating with: volant migrate")
		return loadLegacyFledgeConfig(fledgePath)
	}

	return nil, fmt.Errorf("no build config found (Volantfile or fledge.toml)")
}

func loadLegacyFledgeConfig(path string) (*Volantfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse as generic map first
	var legacy map[string]interface{}
	if err := toml.Unmarshal(data, &legacy); err != nil {
		return nil, err
	}

	// Convert to Volantfile in-memory
	vf := &Volantfile{
		Image: ImageConfig{
			Name:    "app",
			Version: "latest",
		},
	}

	// Extract build strategy
	if build, ok := legacy["build"].(map[string]interface{}); ok {
		if strategy, ok := build["strategy"].(string); ok {
			vf.Build.Strategy = strategy
		}
	}

	// Extract source configuration
	if source, ok := legacy["source"].(map[string]interface{}); ok {
		if dockerfile, ok := source["dockerfile"].(string); ok {
			vf.Build.Dockerfile = dockerfile
		}
		if context, ok := source["context"].(string); ok {
			vf.Build.Context = context
		}
		if target, ok := source["target"].(string); ok {
			vf.Build.Target = target
		}
		if image, ok := source["image"].(string); ok {
			vf.Build.Base = image
		}

		// Handle build args
		if args, ok := source["build_args"].(map[string]interface{}); ok {
			vf.Build.Args = make(map[string]string)
			for k, v := range args {
				if strVal, ok := v.(string); ok {
					vf.Build.Args[k] = strVal
				}
			}
		}
	}

	vf.SetDefaults()
	return vf, nil
}

// LoadRuntimeConfig loads runtime configuration with fallback:
// 1. Volantfile (preferred)
// 2. manifest.toml (legacy)
func LoadRuntimeConfig(dir string) (*Volantfile, error) {
	// Try Volantfile first
	volantfilePath := filepath.Join(dir, "Volantfile")
	if _, err := os.Stat(volantfilePath); err == nil {
		return LoadVolantfile(volantfilePath)
	}

	// Fallback to legacy manifest.toml
	manifestPath := filepath.Join(dir, "manifest.toml")
	if _, err := os.Stat(manifestPath); err == nil {
		log.Printf("⚠️  Using legacy manifest.toml - consider migrating with: volant migrate")
		return loadLegacyManifestConfig(manifestPath)
	}

	return nil, fmt.Errorf("no runtime config found (Volantfile or manifest.toml)")
}

func loadLegacyManifestConfig(path string) (*Volantfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Parse as generic map first
	var legacy map[string]interface{}
	if err := toml.Unmarshal(data, &legacy); err != nil {
		return nil, err
	}

	// Convert to Volantfile in-memory
	vf := &Volantfile{}

	// Image metadata
	if name, ok := legacy["name"].(string); ok {
		vf.Image.Name = name
	}
	if version, ok := legacy["version"].(string); ok {
		vf.Image.Version = version
	}

	// Resources
	if resources, ok := legacy["resources"].(map[string]interface{}); ok {
		if cpuCores, ok := resources["cpu_cores"].(int64); ok {
			vf.Runtime.CPUCores = int(cpuCores)
		}
		if memoryMB, ok := resources["memory_mb"].(int64); ok {
			vf.Runtime.MemoryMB = int(memoryMB)
		}
	}

	// Workload
	if workload, ok := legacy["workload"].(map[string]interface{}); ok {
		if entrypoint, ok := workload["entrypoint"].([]interface{}); ok {
			vf.Runtime.Entrypoint = make([]string, len(entrypoint))
			for i, v := range entrypoint {
				if str, ok := v.(string); ok {
					vf.Runtime.Entrypoint[i] = str
				}
			}
		}
		if workdir, ok := workload["workdir"].(string); ok {
			vf.Runtime.WorkDir = workdir
		}
		if env, ok := workload["env"].(map[string]interface{}); ok {
			vf.Runtime.Env = make(map[string]string)
			for k, v := range env {
				if str, ok := v.(string); ok {
					vf.Runtime.Env[k] = str
				}
			}
		}
	}

	// Network
	if network, ok := legacy["network"].(map[string]interface{}); ok {
		if expose, ok := network["expose"].([]interface{}); ok {
			for _, item := range expose {
				if portMap, ok := item.(map[string]interface{}); ok {
					if port, ok := portMap["port"].(int64); ok {
						vf.Runtime.Expose.Ports = append(vf.Runtime.Expose.Ports, int(port))
					}
				}
			}
		}
	}

	// Volumes
	if storage, ok := legacy["storage"].(map[string]interface{}); ok {
		if volumes, ok := storage["volumes"].(map[string]interface{}); ok {
			vf.Runtime.Volumes = make(map[string]string)
			for k, v := range volumes {
				if str, ok := v.(string); ok {
					vf.Runtime.Volumes[k] = str
				}
			}
		}
	}

	vf.SetDefaults()
	return vf, nil
}

// FindVolantfile searches for a Volantfile in common locations
func FindVolantfile(dir string) (string, error) {
	candidates := []string{
		filepath.Join(dir, "Volantfile"),
		filepath.Join(dir, "volantfile"),
		filepath.Join(dir, ".volant", "Volantfile"),
		filepath.Join(dir, ".volant", "config.toml"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no Volantfile found in %s", dir)
}