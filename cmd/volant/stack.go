// Copyright (c) 2025 HYPR. PTE. LTD.

package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/volantvm/volant/internal/cli/client"
)

func newStackCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stack",
		Short: "Manage multi-VM stacks",
		Long: `Manage multi-VM application stacks (groups of VMs).

Stacks are aliases for deployments with Docker-familiar terminology.`,
	}

	cmd.AddCommand(newStackPsCommand())
	cmd.AddCommand(newStackDownCommand())
	cmd.AddCommand(newStackScaleCommand())

	return cmd
}

func newStackPsCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "ps",
		Aliases: []string{"list", "ls"},
		Short:   "List stacks",
		Long:    "List all stacks (deployments).",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			apiURL := cmd.Flag("api").Value.String()
			c, err := client.New(apiURL)
			if err != nil {
				return err
			}

			deployments, err := c.ListDeployments(ctx)
			if err != nil {
				return fmt.Errorf("list stacks: %w", err)
			}

			if len(deployments) == 0 {
				fmt.Println("No stacks found")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tREPLICAS\tREADY\tCREATED")
			for _, dep := range deployments {
				fmt.Fprintf(w, "%s\t%d\t%d\t%s\n",
					dep.Name,
					dep.DesiredReplicas,
					dep.ReadyReplicas,
					dep.CreatedAt.Format("2006-01-02 15:04"),
				)
			}
			w.Flush()

			return nil
		},
	}
}

func newStackDownCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "down STACK",
		Short: "Stop and remove a stack",
		Long:  "Stop and remove a stack (deployment) and all its VMs.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			stackName := args[0]

			apiURL := cmd.Flag("api").Value.String()
			c, err := client.New(apiURL)
			if err != nil {
				return err
			}

			if err := c.DeleteDeployment(ctx, stackName); err != nil {
				return fmt.Errorf("remove stack: %w", err)
			}

			fmt.Printf("✓ Stack %s removed\n", stackName)
			return nil
		},
	}
}

func newStackScaleCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "scale STACK REPLICAS",
		Short: "Scale a stack",
		Long: `Scale a stack to the specified number of replicas.

Examples:
  volant stack scale myapp 3
  volant stack scale web 5`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			stackName := args[0]

			replicas, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid replicas: %s (must be a number)", args[1])
			}

			if replicas < 0 {
				return fmt.Errorf("replicas must be >= 0")
			}

			apiURL := cmd.Flag("api").Value.String()
			c, err := client.New(apiURL)
			if err != nil {
				return err
			}

			deployment, err := c.ScaleDeployment(ctx, stackName, replicas)
			if err != nil {
				return fmt.Errorf("scale stack: %w", err)
			}

			fmt.Printf("✓ Stack %s scaled to %d replicas (%d ready)\n",
				deployment.Name,
				deployment.DesiredReplicas,
				deployment.ReadyReplicas)

			return nil
		},
	}
}
