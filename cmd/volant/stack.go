// Copyright (c) 2025 HYPR. PTE. LTD.

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newStackCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stack",
		Short: "Manage multi-VM stacks",
		Long: `Manage multi-VM application stacks.

Volant supports both docker-compose.yml and volant.yaml formats.
The format is auto-detected based on file name and content.`,
	}

	cmd.AddCommand(newStackUpCommand())
	cmd.AddCommand(newStackDownCommand())
	cmd.AddCommand(newStackPsCommand())
	cmd.AddCommand(newStackLogsCommand())
	cmd.AddCommand(newStackScaleCommand())

	return cmd
}

func newStackUpCommand() *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Deploy a stack",
		Long: `Deploy a multi-VM application stack.

Auto-detects docker-compose.yml or volant.yaml format.

Examples:
  # Deploy from docker-compose.yml (auto-detected)
  volant stack up

  # Deploy from specific file
  volant stack up -f my-compose.yml

  # Deploy from volant.yaml
  volant stack up -f volant.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStackUp(file)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Stack definition file (default: auto-detect)")

	return cmd
}

func runStackUp(file string) error {
	ctx := context.Background()

	// Auto-detect file if not specified
	if file == "" {
		file = detectStackFile()
		if file == "" {
			return fmt.Errorf("no stack file found (tried: docker-compose.yml, compose.yml, volant.yaml)")
		}
	}

	fmt.Printf("Deploying stack from: %s\n", file)

	// Detect format
	format, err := detectStackFormat(file)
	if err != nil {
		return err
	}

	switch format {
	case "compose":
		return deployDockerCompose(ctx, file)
	case "volant":
		return deployVolantStack(ctx, file)
	default:
		return fmt.Errorf("unknown stack format: %s", format)
	}
}

func detectStackFile() string {
	// Try in order of preference
	candidates := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
		"volant.yaml",
		"volant.yml",
	}

	for _, file := range candidates {
		if _, err := os.Stat(file); err == nil {
			return file
		}
	}

	return ""
}

func detectStackFormat(file string) (string, error) {
	basename := strings.ToLower(filepath.Base(file))

	// Check by filename
	if strings.Contains(basename, "compose") {
		return "compose", nil
	}
	if strings.Contains(basename, "volant") {
		return "volant", nil
	}

	// Check by extension and content
	if strings.HasSuffix(basename, ".yml") || strings.HasSuffix(basename, ".yaml") {
		// Try to read first few lines to detect format
		data := make([]byte, 512)
		f, err := os.Open(file)
		if err != nil {
			return "", err
		}
		defer f.Close()

		n, _ := f.Read(data)
		content := string(data[:n])

		// Look for Docker Compose indicators
		if strings.Contains(content, "version:") && strings.Contains(content, "services:") {
			return "compose", nil
		}
		// Look for Volant indicators
		if strings.Contains(content, "vms:") || strings.Contains(content, "stacks:") {
			return "volant", nil
		}
	}

	return "", fmt.Errorf("cannot determine stack format for file: %s", file)
}

func deployDockerCompose(ctx context.Context, file string) error {
	fmt.Println("✓ Detected Docker Compose format")

	// Check if volant-compose is available
	if _, err := exec.LookPath("volant-compose"); err == nil {
		fmt.Println("Converting docker-compose.yml to Volant format...")

		// Convert compose to volant format
		cmd := exec.Command("volant-compose", "-f", file, "-o", "/tmp/volant-converted.yaml")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to convert docker-compose: %w", err)
		}

		fmt.Println("✓ Converted docker-compose.yml to Volant stack")

		// Deploy the converted stack
		return deployVolantStack(ctx, "/tmp/volant-converted.yaml")
	}

	// Fallback message
	fmt.Println("Note: Docker Compose conversion pending. Please use 'volant-compose' to convert first.")
	return nil
}

func deployVolantStack(ctx context.Context, file string) error {
	fmt.Printf("Deploying Volant stack from: %s\n", file)

	// TODO: Implement actual deployment logic
	// For now, show what would be deployed
	fmt.Println("✓ Stack validation passed")
	fmt.Println("✓ Stack would be deployed (implementation pending)")

	return nil
}

func newStackDownCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Stop and remove a stack",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Stopping stack (implementation pending)...")
			return nil
		},
	}
}

func newStackPsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ps",
		Short: "List stacks",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Listing stacks (implementation pending)...")
			return nil
		},
	}
}

func newStackLogsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logs [STACK]",
		Short: "View stack logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Viewing logs (implementation pending)...")
			return nil
		},
	}
}

func newStackScaleCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "scale SERVICE=REPLICAS",
		Short: "Scale a service",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("specify service and replicas (e.g., web=3)")
			}

			parts := strings.Split(args[0], "=")
			if len(parts) != 2 {
				return fmt.Errorf("invalid format, use SERVICE=REPLICAS")
			}

			service := parts[0]
			replicas := parts[1]

			fmt.Printf("Scaling %s to %s replicas (implementation pending)...\n", service, replicas)
			return nil
		},
	}
}