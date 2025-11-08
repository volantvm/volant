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
	"github.com/volantvm/volant/internal/config"
)

func newBuildCommand() *cobra.Command {
	var (
		tag        string
		file       string
		output     string
		dockerfile string
		buildArgs  []string
	)

	cmd := &cobra.Command{
		Use:   "build [PATH]",
		Short: "Build a Volant image",
		Long: `Build a Volant image from a Dockerfile or Volantfile.

Examples:
  # Build from Dockerfile in current directory
  volant build -t myapp:latest .

  # Build from specific Dockerfile
  volant build -t myapp --file Dockerfile.prod .

  # Build from Volantfile
  volant build -f Volantfile -o myapp.img

  # Build with arguments
  volant build -t myapp --build-arg VERSION=1.0 .`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuild(buildOptions{
				Tag:        tag,
				File:       file,
				Output:     output,
				Dockerfile: dockerfile,
				Context:    determineContext(args),
				BuildArgs:  parseBuildArgs(buildArgs),
			})
		},
	}

	cmd.Flags().StringVarP(&tag, "tag", "t", "", "Name and optionally a tag (format: name:tag)")
	cmd.Flags().StringVarP(&file, "file", "f", "", "Path to Volantfile or Dockerfile")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path")
	cmd.Flags().StringVar(&dockerfile, "dockerfile", "", "Alias for --file (Docker compatibility)")
	cmd.Flags().StringArrayVar(&buildArgs, "build-arg", nil, "Build arguments (KEY=VALUE)")

	return cmd
}

type buildOptions struct {
	Tag        string
	File       string
	Output     string
	Dockerfile string
	Context    string
	BuildArgs  map[string]string
}

func runBuild(opts buildOptions) error {
	ctx := context.Background()

	// Determine build file
	buildFile := opts.File
	if buildFile == "" && opts.Dockerfile != "" {
		buildFile = opts.Dockerfile
	}
	if buildFile == "" {
		buildFile = detectBuildFile(opts.Context)
	}

	// Auto-detect build strategy
	strategy, err := detectStrategy(buildFile)
	if err != nil {
		return err
	}

	switch strategy {
	case "dockerfile":
		return buildFromDockerfile(ctx, buildFile, opts)
	case "volantfile":
		return buildFromVolantfile(ctx, buildFile, opts)
	default:
		return fmt.Errorf("unknown build strategy: %s", strategy)
	}
}

func detectBuildFile(context string) string {
	// Try Dockerfile first (Docker compatibility)
	dockerfile := filepath.Join(context, "Dockerfile")
	if _, err := os.Stat(dockerfile); err == nil {
		return dockerfile
	}

	// Try Volantfile
	volantfile := filepath.Join(context, "Volantfile")
	if _, err := os.Stat(volantfile); err == nil {
		return volantfile
	}

	// Try fledge.toml (legacy)
	fledgeToml := filepath.Join(context, "fledge.toml")
	if _, err := os.Stat(fledgeToml); err == nil {
		return fledgeToml
	}

	return dockerfile // Default, will error if not found
}

func detectStrategy(buildFile string) (string, error) {
	name := filepath.Base(buildFile)

	switch {
	case name == "Dockerfile" || filepath.Ext(name) == ".dockerfile":
		return "dockerfile", nil
	case name == "Volantfile":
		return "volantfile", nil
	case name == "fledge.toml":
		return "volantfile", nil // Legacy support
	default:
		return "", fmt.Errorf("cannot detect build strategy from file: %s", name)
	}
}

func buildFromDockerfile(ctx context.Context, dockerfile string, opts buildOptions) error {
	fmt.Printf("Building from Dockerfile: %s\n", dockerfile)

	// For now, delegate to fledge if available
	// TODO: Integrate fledge libraries directly
	if _, err := exec.LookPath("fledge"); err == nil {
		return delegateToFledge(dockerfile, opts)
	}

	// Fallback message
	fmt.Println("Note: Fledge integration pending. Please use 'fledge build' directly for now.")
	return nil
}

func buildFromVolantfile(ctx context.Context, volantfilePath string, opts buildOptions) error {
	fmt.Printf("Building from Volantfile: %s\n", volantfilePath)

	// Load Volantfile
	vf, err := config.LoadVolantfile(volantfilePath)
	if err != nil {
		return fmt.Errorf("failed to load Volantfile: %w", err)
	}

	fmt.Printf("✓ Loaded Volantfile for image: %s:%s\n", vf.Image.Name, vf.Image.Version)
	fmt.Printf("  Strategy: %s\n", vf.Build.Strategy)
	fmt.Printf("  CPU: %d cores, Memory: %dMB\n", vf.Runtime.CPUCores, vf.Runtime.MemoryMB)

	// For now, delegate to fledge if available by creating temporary legacy files
	if _, err := exec.LookPath("fledge"); err == nil {
		// Create temporary legacy config files
		tmpDir := "/tmp/volant-build"
		os.MkdirAll(tmpDir, 0755)

		fledgePath := filepath.Join(tmpDir, "fledge.toml")
		manifestPath := filepath.Join(tmpDir, "manifest.toml")

		if err := vf.SaveAsLegacy(fledgePath, manifestPath); err != nil {
			return fmt.Errorf("failed to create legacy config: %w", err)
		}

		fmt.Println("✓ Converting to legacy format for fledge compatibility")
		return delegateToFledge(fledgePath, opts)
	}

	// Fallback message
	fmt.Println("Note: Full Volantfile support pending. Please use 'fledge build' directly for now.")
	return nil
}

func delegateToFledge(buildFile string, opts buildOptions) error {
	// Build fledge command
	args := []string{"build", "-c", buildFile}

	if opts.Output != "" {
		args = append(args, "-o", opts.Output)
	}

	// Add build arguments
	for key, value := range opts.BuildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}

	// Execute fledge
	cmd := exec.Command("fledge", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("fledge build failed: %w", err)
	}

	fmt.Println("✓ Image built successfully")

	// Auto-install if tag provided
	if opts.Tag != "" {
		return autoInstallImage(opts.Output, opts.Tag)
	}

	return nil
}

func autoInstallImage(imagePath, tag string) error {
	fmt.Printf("Auto-installing image with tag: %s\n", tag)
	// TODO: Implement image installation
	return nil
}

func determineContext(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return "."
}

func parseBuildArgs(args []string) map[string]string {
	result := make(map[string]string)
	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

func sanitizeTag(tag string) string {
	// Remove special characters from tag for filename
	replacer := strings.NewReplacer(":", "-", "/", "-")
	return replacer.Replace(tag)
}