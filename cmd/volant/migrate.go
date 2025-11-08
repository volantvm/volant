// Copyright (c) 2025 HYPR. PTE. LTD.

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
)

func newMigrateCommand() *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate legacy configs to Volantfile",
		Long: `Migrate fledge.toml and manifest.toml to unified Volantfile format.

Examples:
  volant migrate                           # Auto-detect and migrate
  volant migrate -o custom/path/Volantfile # Custom output path`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrate(outputPath)
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "Volantfile",
		"Output path for Volantfile")

	return cmd
}

func runMigrate(outputPath string) error {
	ctx := context.Background()
	_ = ctx // Suppress unused variable warning

	// 1. Auto-detect legacy configs
	fledgePath := findConfigFile("fledge.toml")
	manifestPath := findConfigFile("manifest.toml")

	if fledgePath == "" && manifestPath == "" {
		return fmt.Errorf("no legacy config files found (fledge.toml or manifest.toml)")
	}

	log.Printf("Found configs:")
	if fledgePath != "" {
		log.Printf("  - %s", fledgePath)
	}
	if manifestPath != "" {
		log.Printf("  - %s", manifestPath)
	}

	// 2. Load legacy configs
	var fledgeCfg map[string]interface{}
	var manifestCfg map[string]interface{}

	if fledgePath != "" {
		data, err := os.ReadFile(fledgePath)
		if err != nil {
			return fmt.Errorf("read fledge.toml: %w", err)
		}
		fledgeCfg = make(map[string]interface{})
		if err := toml.Unmarshal(data, &fledgeCfg); err != nil {
			return fmt.Errorf("parse fledge.toml: %w", err)
		}
	}

	if manifestPath != "" {
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			return fmt.Errorf("read manifest.toml: %w", err)
		}
		manifestCfg = make(map[string]interface{})
		if err := toml.Unmarshal(data, &manifestCfg); err != nil {
			return fmt.Errorf("parse manifest.toml: %w", err)
		}
	}

	// 3. Merge into Volantfile structure
	vf := make(map[string]interface{})

	// Initialize sections
	vf["image"] = make(map[string]interface{})
	vf["build"] = make(map[string]interface{})
	vf["runtime"] = make(map[string]interface{})

	// From fledge.toml
	if fledgeCfg != nil {
		if build, ok := fledgeCfg["build"].(map[string]interface{}); ok {
			if strategy, ok := build["strategy"].(string); ok {
				vf["build"].(map[string]interface{})["strategy"] = strategy
			}
		}
		if source, ok := fledgeCfg["source"].(map[string]interface{}); ok {
			buildSection := vf["build"].(map[string]interface{})
			if dockerfile, ok := source["dockerfile"].(string); ok {
				buildSection["dockerfile"] = dockerfile
			}
			if context, ok := source["context"].(string); ok {
				buildSection["context"] = context
			}
			if target, ok := source["target"].(string); ok {
				buildSection["target"] = target
			}
			if args, ok := source["build_args"].(map[string]interface{}); ok {
				buildSection["args"] = args
			}
			if image, ok := source["image"].(string); ok {
				buildSection["base"] = image
			}
		}
	}

	// From manifest.toml
	if manifestCfg != nil {
		// Image metadata
		imageSection := vf["image"].(map[string]interface{})
		if name, ok := manifestCfg["name"].(string); ok {
			imageSection["name"] = name
		}
		if version, ok := manifestCfg["version"].(string); ok {
			imageSection["version"] = version
		}

		// Runtime resources
		runtimeSection := vf["runtime"].(map[string]interface{})
		if resources, ok := manifestCfg["resources"].(map[string]interface{}); ok {
			if cpuCores, ok := resources["cpu_cores"].(int64); ok {
				runtimeSection["cpu_cores"] = cpuCores
			}
			if memoryMB, ok := resources["memory_mb"].(int64); ok {
				runtimeSection["memory_mb"] = memoryMB
			}
		}

		// Workload config
		if workload, ok := manifestCfg["workload"].(map[string]interface{}); ok {
			if entrypoint, ok := workload["entrypoint"].([]interface{}); ok {
				runtimeSection["entrypoint"] = entrypoint
			}
			if workdir, ok := workload["workdir"].(string); ok {
				runtimeSection["workdir"] = workdir
			}
			if env, ok := workload["env"].(map[string]interface{}); ok {
				envMap := make(map[string]interface{})
				for k, v := range env {
					envMap[k] = v
				}
				runtimeSection["env"] = envMap
			}
		}

		// Network exposure
		if network, ok := manifestCfg["network"].(map[string]interface{}); ok {
			if expose, ok := network["expose"].([]interface{}); ok {
				ports := []int{}
				for _, item := range expose {
					if portMap, ok := item.(map[string]interface{}); ok {
						if port, ok := portMap["port"].(int64); ok {
							ports = append(ports, int(port))
						}
					}
				}
				if len(ports) > 0 {
					exposeSection := make(map[string]interface{})
					exposeSection["ports"] = ports
					runtimeSection["expose"] = exposeSection
				}
			}
		}

		// Storage volumes
		if storage, ok := manifestCfg["storage"].(map[string]interface{}); ok {
			if volumes, ok := storage["volumes"].(map[string]interface{}); ok {
				runtimeSection["volumes"] = volumes
			}
		}
	}

	// 4. Set defaults if missing
	imageSection := vf["image"].(map[string]interface{})
	if imageSection["name"] == nil {
		imageSection["name"] = "myapp"
	}
	if imageSection["version"] == nil {
		imageSection["version"] = "latest"
	}

	buildSection := vf["build"].(map[string]interface{})
	if buildSection["strategy"] == nil {
		buildSection["strategy"] = "dockerfile"
	}
	if buildSection["context"] == nil {
		buildSection["context"] = "."
	}

	runtimeSection := vf["runtime"].(map[string]interface{})
	if runtimeSection["cpu_cores"] == nil {
		runtimeSection["cpu_cores"] = 2
	}
	if runtimeSection["memory_mb"] == nil {
		runtimeSection["memory_mb"] = 512
	}

	// 5. Write Volantfile
	data, err := toml.Marshal(vf)
	if err != nil {
		return fmt.Errorf("marshal Volantfile: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("write Volantfile: %w", err)
	}

	log.Printf("✓ Created %s", outputPath)
	log.Printf("\nNext steps:")
	log.Printf("  1. Review the generated Volantfile")
	log.Printf("  2. Test build: volant build .")
	log.Printf("  3. Archive old configs: mkdir .legacy && mv fledge.toml manifest.toml .legacy/")

	return nil
}

func findConfigFile(name string) string {
	// Check current directory
	if _, err := os.Stat(name); err == nil {
		return name
	}

	// Check .volant/ directory
	volantPath := filepath.Join(".volant", name)
	if _, err := os.Stat(volantPath); err == nil {
		return volantPath
	}

	return ""
}