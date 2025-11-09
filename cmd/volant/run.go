// Copyright (c) 2025 HYPR. PTE. LTD.

package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/volantvm/volant/internal/cli/client"
	"github.com/volantvm/volant/internal/server/orchestrator/vmconfig"
)

func newRunCommand() *cobra.Command {
	var (
		name      string
		detach    bool
		cpus      int
		memory    int
		env       []string
		ports     []string
		runtime   string
		kernel    string
	)

	cmd := &cobra.Command{
		Use:   "run [OPTIONS] IMAGE [COMMAND] [ARG...]",
		Short: "Run a VM from an image",
		Long: `Run a new microVM instance from an image.

This command creates and starts a new VM from the specified image,
similar to 'docker run'. By default, the VM runs in the foreground.

Examples:
  # Run an image with default settings
  volant run myapp:latest

  # Run with custom name and resources
  volant run --name web --cpus 2 --memory 512 nginx:latest

  # Run in background (detached)
  volant run -d --name api myapi:v1

  # Run with environment variables
  volant run -e PORT=8080 -e DEBUG=true myapp:latest

  # Run with port mapping
  volant run -p 8080:80 -p 443:443 nginx:latest`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			image := args[0]

			// If no name provided, generate one from image
			if name == "" {
				name = generateVMName(image)
			}

			// Build config overrides
			overrides := vmconfig.Overrides{}

			if cpus > 0 {
				overrides.CPUCores = &cpus
			}
			if memory > 0 {
				overrides.MemoryMB = &memory
			}

			// Parse environment variables
			if len(env) > 0 {
				envMap := make(map[string]string)
				for _, e := range env {
					parts := strings.SplitN(e, "=", 2)
					if len(parts) == 2 {
						envMap[parts[0]] = parts[1]
					} else {
						envMap[parts[0]] = ""
					}
				}
				overrides.Env = envMap
			}

			// Parse port mappings
			if len(ports) > 0 {
				exposePorts := []vmconfig.Expose{}
				for _, p := range ports {
					expose, err := parsePortMapping(p)
					if err != nil {
						return fmt.Errorf("invalid port mapping %q: %w", p, err)
					}
					exposePorts = append(exposePorts, expose)
				}
				overrides.ExposePorts = exposePorts
			}

			// Create VM request
			apiURL := cmd.Flag("api").Value.String()
			c, err := client.New(apiURL)
			if err != nil {
				return err
			}

			createReq := client.CreateVMRequest{
				Name:      name,
				Image:     image,
				Runtime:   runtime,
				Overrides: overrides,
			}

			if kernel != "" {
				createReq.KernelCmdline = kernel
			}

			vm, err := c.CreateVM(ctx, createReq)
			if err != nil {
				return fmt.Errorf("failed to create VM: %w", err)
			}

			if detach {
				fmt.Printf("%s\n", vm.ID)
				fmt.Printf("VM %s started in background\n", vm.Name)
			} else {
				fmt.Printf("VM %s created from %s\n", vm.Name, image)
				fmt.Printf("Status: %s\n", vm.Status)
				if vm.IPAddress != "" {
					fmt.Printf("IP: %s\n", vm.IPAddress)
				}
				fmt.Printf("\nTo see logs: volant logs %s\n", vm.Name)
				fmt.Printf("To stop: volant vms delete %s\n", vm.Name)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Assign a name to the VM")
	cmd.Flags().BoolVarP(&detach, "detach", "d", false, "Run VM in background and print VM ID")
	cmd.Flags().IntVar(&cpus, "cpus", 0, "Number of CPUs (default: from image manifest)")
	cmd.Flags().IntVar(&memory, "memory", 0, "Memory in MB (default: from image manifest)")
	cmd.Flags().StringArrayVarP(&env, "env", "e", nil, "Set environment variables (KEY=VALUE)")
	cmd.Flags().StringArrayVarP(&ports, "publish", "p", nil, "Publish port (HOST:CONTAINER)")
	cmd.Flags().StringVar(&runtime, "runtime", "", "Runtime to use")
	cmd.Flags().StringVar(&kernel, "kernel-cmdline", "", "Additional kernel command line arguments")

	return cmd
}

// generateVMName creates a VM name from an image reference
func generateVMName(image string) string {
	// Remove tag/digest
	name := image
	if idx := strings.LastIndex(name, ":"); idx > 0 {
		name = name[:idx]
	}
	if idx := strings.LastIndex(name, "@"); idx > 0 {
		name = name[:idx]
	}
	// Get last component after /
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	// Sanitize
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ToLower(name)

	// Add random suffix to avoid conflicts
	return fmt.Sprintf("%s-%d", name, os.Getpid()%10000)
}

// parsePortMapping parses a port mapping string like "8080:80" or "8080:80/tcp"
func parsePortMapping(s string) (vmconfig.Expose, error) {
	var expose vmconfig.Expose

	// Check for protocol suffix
	protocol := "tcp"
	if idx := strings.LastIndex(s, "/"); idx > 0 {
		protocol = s[idx+1:]
		s = s[:idx]
	}
	expose.Protocol = protocol

	// Parse host:container
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return expose, fmt.Errorf("expected format HOST:CONTAINER[/PROTOCOL]")
	}

	hostPort, err := strconv.Atoi(parts[0])
	if err != nil {
		return expose, fmt.Errorf("invalid host port: %s", parts[0])
	}

	containerPort, err := strconv.Atoi(parts[1])
	if err != nil {
		return expose, fmt.Errorf("invalid container port: %s", parts[1])
	}

	expose.Port = containerPort
	expose.HostPort = hostPort
	expose.Name = fmt.Sprintf("port-%d", containerPort)

	return expose, nil
}
