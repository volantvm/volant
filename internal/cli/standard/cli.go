// Copyright (c) 2025 HYPR. PTE. LTD.
//
// Business Source License 1.1
// See LICENSE file in the project root for details.

package standard

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Execute runs the Cobra-based CLI entry point.
func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volant",
		Short: "volant command-line interface",
		Long: `volant provides access to the VOLANT control plane.

Core commands:
  vms       Manage microVMs
  images    Install/remove image manifests
  setup     Helper for host networking/service configuration
  console   Inspect or attach to VM consoles
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().StringP("api", "a", envOrDefault("VOLANT_API_BASE", "http://127.0.0.1:7777"), "volantd base URL")

	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newVMsCmd())
	cmd.AddCommand(newImagesCmd())
	cmd.AddCommand(newSetupCmd())
	cmd.AddCommand(newDeploymentsCmd())
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the volant client version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "volant CLI (prototype)\n")
		},
	}
}
