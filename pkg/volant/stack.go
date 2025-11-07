package volant

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// StackFile represents a volant.yaml stack definition
type StackFile struct {
	Version  string                   `yaml:"version"`
	Name     string                   `yaml:"name"`
	Services map[string]StackService  `yaml:"services"`
	Networks map[string]StackNetwork  `yaml:"networks,omitempty"`
	Volumes  map[string]StackVolume   `yaml:"volumes,omitempty"`
}

// StackService represents a service in a Volant stack
// Each service runs in its own microVM
type StackService struct {
	Image      string            `yaml:"image"`
	Memory     int               `yaml:"memory,omitempty"`      // Memory in MB (default: 512)
	VCPUs      int               `yaml:"vcpus,omitempty"`       // CPU count (default: 1)
	Ports      []string          `yaml:"ports,omitempty"`       // Port mappings "host:guest"
	Env        map[string]string `yaml:"environment,omitempty"` // Environment variables
	Volumes    []string          `yaml:"volumes,omitempty"`     // Volume mounts "vol_name:/path"
	DependsOn  []string          `yaml:"depends_on,omitempty"`  // Service dependencies
	Entrypoint []string          `yaml:"entrypoint,omitempty"`  // Override entrypoint
	Args       []string          `yaml:"args,omitempty"`        // Override args/command
}

// StackNetwork represents a network definition
// Currently Volant uses a single default subnet (192.168.127.0/24)
type StackNetwork struct {
	Subnet  string `yaml:"subnet,omitempty"`
	Gateway string `yaml:"gateway,omitempty"`
}

// StackVolume represents a persistent volume definition
type StackVolume struct {
	Size   string `yaml:"size,omitempty"`   // Size (e.g., "1G", "100M")
	Driver string `yaml:"driver,omitempty"` // Driver type (default: local)
}

// Parse reads and parses a volant.yaml file
func Parse(path string) (*StackFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	var stack StackFile
	if err := yaml.Unmarshal(data, &stack); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if err := stack.Validate(); err != nil {
		return nil, err
	}

	return &stack, nil
}

// WriteYAML writes a StackFile to a YAML file
func WriteYAML(stack *StackFile, path string) error {
	// Set defaults
	if stack.Version == "" {
		stack.Version = "1.0"
	}

	// Validate before writing
	if err := stack.Validate(); err != nil {
		return err
	}

	// Marshal to YAML with indentation
	data, err := yaml.Marshal(stack)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	return nil
}

// Validate checks if the stack file is valid
func (s *StackFile) Validate() error {
	if s.Version == "" {
		return fmt.Errorf("version field is required")
	}

	if s.Version != "1.0" {
		return fmt.Errorf("unsupported version: %s (only version 1.0 is supported)", s.Version)
	}

	if s.Name == "" {
		return fmt.Errorf("name field is required")
	}

	if len(s.Services) == 0 {
		return fmt.Errorf("at least one service is required")
	}

	// Validate each service
	for name, service := range s.Services {
		if err := service.Validate(name); err != nil {
			return err
		}
	}

	// Validate dependencies reference existing services
	for name, service := range s.Services {
		for _, dep := range service.DependsOn {
			if _, exists := s.Services[dep]; !exists {
				return fmt.Errorf("service '%s': depends_on references non-existent service '%s'", name, dep)
			}
		}
	}

	return nil
}

// Validate checks if a service is valid
func (s *StackService) Validate(name string) error {
	if s.Image == "" {
		return fmt.Errorf("service '%s': image field is required", name)
	}

	// Validate memory (if specified)
	if s.Memory < 0 {
		return fmt.Errorf("service '%s': memory must be positive", name)
	}

	// Validate vcpus (if specified)
	if s.VCPUs < 0 {
		return fmt.Errorf("service '%s': vcpus must be positive", name)
	}

	return nil
}

// SetDefaults applies default values to the stack
func (s *StackFile) SetDefaults() {
	if s.Version == "" {
		s.Version = "1.0"
	}

	for name, service := range s.Services {
		if service.Memory == 0 {
			service.Memory = 512 // Default 512MB
		}
		if service.VCPUs == 0 {
			service.VCPUs = 1 // Default 1 vCPU
		}
		s.Services[name] = service
	}

	// Set default network if not specified
	if s.Networks == nil {
		s.Networks = make(map[string]StackNetwork)
	}
	if _, exists := s.Networks["default"]; !exists {
		s.Networks["default"] = StackNetwork{
			Subnet:  "192.168.127.0/24",
			Gateway: "192.168.127.1",
		}
	}
}
