// Copyright (c) 2025 HYPR. PTE. LTD.

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/volantvm/volant/internal/cli/standard"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "volant",
		Short: "Volant - MicroVM Platform",
		Long: `Volant - Developer-friendly microVM orchestration

Volant provides Docker-compatible commands for managing lightweight,
hardware-isolated virtual machines powered by Cloud Hypervisor.

Common Commands:
  build       Build an image from a Dockerfile or Volantfile
  run         Run a VM from an image
  ps          List running VMs
  images      Manage images
  stack       Manage multi-VM application stacks

Management Commands:
  system      Manage the Volant daemon
  volume      Manage persistent volumes
  network     Manage networks (coming soon)

Run 'volant COMMAND --help' for more information on a command.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add the new commands
	rootCmd.AddCommand(newBuildCommand())
	rootCmd.AddCommand(newStackCommand())

	// For now, also execute the standard CLI to preserve existing functionality
	// This allows us to gradually migrate commands
	if len(os.Args) > 1 {
		// Check if it's one of our new commands
		switch os.Args[1] {
		case "build", "stack":
			// Execute our new root command
			if err := rootCmd.Execute(); err != nil {
				fmt.Fprintf(os.Stderr, "command error: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	// Fall back to standard CLI for all other commands
	if err := standard.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "command error: %v\n", err)
		os.Exit(1)
	}
}