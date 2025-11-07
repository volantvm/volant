package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/volantvm/volant/pkg/compose"
	"github.com/volantvm/volant/pkg/converter"
	"github.com/volantvm/volant/pkg/volant"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "volant-compose",
	Short: "Convert Docker Compose files to Volant stacks",
	Long: `volant-compose is a CLI tool that converts docker-compose.yml files
to volant.yaml stack definitions for the Volant microVM orchestrator.

Volant provides VM-level security isolation with container-like speed,
and this tool helps you migrate from Docker Compose to Volant seamlessly.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
}

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Convert docker-compose.yml to volant.yaml",
	Long: `Convert a Docker Compose file to a Volant stack definition.

This command parses your docker-compose.yml file and generates a
volant.yaml file that can be deployed to the Volant orchestrator.

Supported features:
  - Services (image, ports, environment, volumes)
  - Named volumes
  - Service dependencies (depends_on)
  - Environment variables (map and array syntax)
  - Port mappings

Unsupported features (will error):
  - Build contexts (use Fledge separately)
  - Bind mounts (use named volumes)
  - deploy.replicas (Phase 2)
  - Health checks (Track D)
  - Secrets (Track A)

Example:
  volant-compose convert -f docker-compose.yml -o volant.yaml
  volant-compose convert --name my-stack
`,
	RunE: runConvert,
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate docker-compose.yml can be converted",
	Long: `Validate that a Docker Compose file can be successfully converted
to a Volant stack definition without actually writing the output file.

This is useful for checking if your docker-compose.yml file uses
any unsupported features before attempting conversion.

Example:
  volant-compose validate -f docker-compose.yml
`,
	RunE: runValidate,
}

var (
	inputFile  string
	outputFile string
	stackName  string
	verbose    bool
)

func init() {
	// Add commands
	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(validateCmd)

	// Convert command flags
	convertCmd.Flags().StringVarP(&inputFile, "file", "f", "docker-compose.yml", "Input docker-compose.yml file")
	convertCmd.Flags().StringVarP(&outputFile, "output", "o", "volant.yaml", "Output volant.yaml file")
	convertCmd.Flags().StringVarP(&stackName, "name", "n", "", "Stack name (default: derived from directory)")
	convertCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	// Validate command flags
	validateCmd.Flags().StringVarP(&inputFile, "file", "f", "docker-compose.yml", "Input docker-compose.yml file")
	validateCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
}

func runConvert(cmd *cobra.Command, args []string) error {
	if verbose {
		fmt.Printf("Reading %s...\n", inputFile)
	}

	// Parse Docker Compose file
	composeFile, err := compose.Parse(inputFile)
	if err != nil {
		return fmt.Errorf("failed to parse docker-compose.yml: %w", err)
	}

	if verbose {
		fmt.Printf("Parsed %d services\n", len(composeFile.Services))
	}

	// Derive stack name if not provided
	name := stackName
	if name == "" {
		name = converter.DeriveStackName(inputFile)
		if verbose {
			fmt.Printf("Derived stack name: %s\n", name)
		}
	}

	// Convert to Volant format
	if verbose {
		fmt.Println("Converting to Volant stack...")
	}

	stack, err := converter.Convert(composeFile, name)
	if err != nil {
		return fmt.Errorf("conversion failed: %w", err)
	}

	// Write Volant YAML
	if verbose {
		fmt.Printf("Writing %s...\n", outputFile)
	}

	if err := volant.WriteYAML(stack, outputFile); err != nil {
		return fmt.Errorf("failed to write volant.yaml: %w", err)
	}

	fmt.Printf("✅ Successfully converted %s → %s\n", inputFile, outputFile)
	fmt.Printf("   Stack name: %s\n", stack.Name)
	fmt.Printf("   Services: %d\n", len(stack.Services))
	if len(stack.Volumes) > 0 {
		fmt.Printf("   Volumes: %d\n", len(stack.Volumes))
	}

	fmt.Println("\nNext steps:")
	fmt.Println("  1. Review the generated volant.yaml file")
	fmt.Println("  2. Build any required images with Fledge:")
	fmt.Println("     fledge build -f Dockerfile --name myapp --version 1.0.0")
	fmt.Println("  3. Deploy the stack (Phase 2):")
	fmt.Println("     volant stack deploy -f volant.yaml")

	return nil
}

func runValidate(cmd *cobra.Command, args []string) error {
	if verbose {
		fmt.Printf("Reading %s...\n", inputFile)
	}

	// Parse Docker Compose file
	composeFile, err := compose.Parse(inputFile)
	if err != nil {
		return fmt.Errorf("failed to parse docker-compose.yml: %w", err)
	}

	if verbose {
		fmt.Printf("Parsed %d services\n", len(composeFile.Services))
	}

	// Try to convert (but don't write)
	if verbose {
		fmt.Println("Validating conversion...")
	}

	name := stackName
	if name == "" {
		name = converter.DeriveStackName(inputFile)
	}

	_, err = converter.Convert(composeFile, name)
	if err != nil {
		fmt.Println("❌ Validation failed:")
		fmt.Printf("   %v\n", err)
		return err
	}

	fmt.Println("✅ Validation successful!")
	fmt.Printf("   File: %s\n", inputFile)
	fmt.Printf("   Services: %d\n", len(composeFile.Services))
	if len(composeFile.Volumes) > 0 {
		fmt.Printf("   Volumes: %d\n", len(composeFile.Volumes))
	}
	fmt.Println("\nThis docker-compose.yml file can be converted to Volant format.")
	fmt.Println("Run 'volant-compose convert' to generate volant.yaml")

	return nil
}
