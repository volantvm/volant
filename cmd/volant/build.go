// Copyright (c) 2025 HYPR. PTE. LTD.

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/volantvm/volant/internal/cli/client"
	"github.com/volantvm/volant/internal/config"
	"github.com/volantvm/volant/internal/imagespec"
	"github.com/volantvm/volant/pkg/builder"
	"github.com/volantvm/volant/pkg/fsdetect"
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
	// Parse tag into name:version
	name, version := parseImageTag(opts.Tag)
	if opts.Tag == "" {
		// Generate default tag from context directory
		name = filepath.Base(opts.Context)
		if name == "." || name == "/" {
			name = "myapp"
		}
		version = "latest"
	}

	// Determine output path
	outputPath := opts.Output
	if outputPath == "" {
		outputPath = fmt.Sprintf("%s-%s.squashfs", name, version)
	}

	// Use native builder - NO Docker daemon required!
	nb := builder.NewNativeBuilder("")

	buildOpts := builder.BuildOptions{
		Dockerfile:   dockerfile,
		Context:      opts.Context,
		Target:       "",
		BuildArgs:    opts.BuildArgs,
		OutputPath:   outputPath,
		ImageName:    name,
		ImageVersion: version,
		Reporter:     builder.NewConsoleReporter(os.Stdout),
	}

	// Build with spectacular UX!
	result, err := nb.Build(ctx, buildOpts)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Auto-install to volantd registry
	return autoInstallImage("http://127.0.0.1:7777", result.SquashfsPath, fmt.Sprintf("%s:%s", name, version))
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
	fmt.Printf("  CPU: %d cores, Memory: %dMB\n\n", vf.Runtime.CPUCores, vf.Runtime.MemoryMB)

	// Only support dockerfile strategy for now
	if vf.Build.Strategy != "dockerfile" {
		return fmt.Errorf("unsupported build strategy: %s (only 'dockerfile' is currently supported)", vf.Build.Strategy)
	}

	// Resolve paths relative to Volantfile directory
	volantfileDir := filepath.Dir(volantfilePath)
	dockerfilePath := vf.Build.Dockerfile
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile"
	}
	if !filepath.IsAbs(dockerfilePath) {
		dockerfilePath = filepath.Join(volantfileDir, dockerfilePath)
	}

	contextPath := vf.Build.Context
	if contextPath == "" {
		contextPath = volantfileDir
	}
	if !filepath.IsAbs(contextPath) {
		contextPath = filepath.Join(volantfileDir, contextPath)
	}

	// Determine output path
	outputPath := opts.Output
	if outputPath == "" {
		outputPath = fmt.Sprintf("%s-%s.squashfs", vf.Image.Name, vf.Image.Version)
	}

	// Merge build args (CLI args override Volantfile)
	buildArgs := make(map[string]string)
	for k, v := range vf.Build.Args {
		buildArgs[k] = v
	}
	for k, v := range opts.BuildArgs {
		buildArgs[k] = v // CLI takes precedence
	}

	// Use native builder with Volantfile configuration
	nb := builder.NewNativeBuilder("")

	buildOpts := builder.BuildOptions{
		Dockerfile:   dockerfilePath,
		Context:      contextPath,
		Target:       vf.Build.Target,
		BuildArgs:    buildArgs,
		OutputPath:   outputPath,
		ImageName:    vf.Image.Name,
		ImageVersion: vf.Image.Version,
		Reporter:     builder.NewConsoleReporter(os.Stdout),
	}

	// Build with spectacular UX!
	result, err := nb.Build(ctx, buildOpts)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Auto-install to volantd registry
	return autoInstallImage("http://127.0.0.1:7777", result.SquashfsPath, fmt.Sprintf("%s:%s", vf.Image.Name, vf.Image.Version))
}

func autoInstallImage(apiURL, imagePath, tag string) error {
	fmt.Printf("Auto-installing image with tag: %s\n", tag)
	ctx := context.Background()

	// Parse tag into name:version
	name, version := parseImageTag(tag)

	// Verify image file exists
	if _, err := os.Stat(imagePath); err != nil {
		return fmt.Errorf("image file not found: %w", err)
	}

	// Calculate checksum
	checksum, err := calculateSHA256(imagePath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Detect filesystem format using unified fsdetect package
	format := fsdetect.DetectFormat(imagePath, fsdetect.FormatSquashFS)

	// Get absolute path for image
	absPath, err := filepath.Abs(imagePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create manifest
	manifest := imagespec.Manifest{
		SchemaVersion: "1.0",
		Name:          name,
		Version:       version,
		Runtime:       name, // Default runtime same as name
		RootFS: imagespec.RootFS{
			URL:      absPath,
			Checksum: checksum,
			Format:   format.String(),
		},
	}

	// Normalize and validate
	manifest.Normalize()
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("manifest validation failed: %w", err)
	}

	// Connect to volant daemon
	c, err := client.New(apiURL)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Install the image
	if err := c.InstallImage(ctx, manifest); err != nil {
		return fmt.Errorf("failed to install image: %w", err)
	}

	fmt.Printf("✓ Image installed successfully: %s:%s\n", name, version)
	fmt.Printf("  Format: %s\n", format)
	fmt.Printf("  Path: %s\n", absPath)
	fmt.Printf("  Checksum: %s\n", checksum)

	return nil
}

// parseImageTag parses a tag in the format "name:version" or "name" (defaults to "latest")
func parseImageTag(tag string) (name, version string) {
	parts := strings.SplitN(tag, ":", 2)
	name = parts[0]
	if len(parts) == 2 {
		version = parts[1]
	} else {
		version = "latest"
	}
	return name, version
}

// calculateSHA256 calculates the SHA256 checksum of a file
func calculateSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
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