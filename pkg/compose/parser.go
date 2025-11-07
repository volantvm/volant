package compose

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeFile represents a docker-compose.yml file structure
type ComposeFile struct {
	Version  string                `yaml:"version"`
	Services map[string]Service    `yaml:"services"`
	Networks map[string]Network    `yaml:"networks,omitempty"`
	Volumes  map[string]Volume     `yaml:"volumes,omitempty"`
}

// Service represents a service definition in docker-compose
type Service struct {
	Image       string                 `yaml:"image,omitempty"`
	Build       *BuildConfig           `yaml:"build,omitempty"`
	ContainerName string               `yaml:"container_name,omitempty"`
	Ports       []string               `yaml:"ports,omitempty"`
	Environment interface{}            `yaml:"environment,omitempty"` // Can be array or map
	Volumes     []string               `yaml:"volumes,omitempty"`
	DependsOn   interface{}            `yaml:"depends_on,omitempty"` // Can be array or map
	Networks    interface{}            `yaml:"networks,omitempty"`   // Can be array or map
	Command     interface{}            `yaml:"command,omitempty"`    // Can be string or array
	Entrypoint  interface{}            `yaml:"entrypoint,omitempty"` // Can be string or array
	Deploy      *DeployConfig          `yaml:"deploy,omitempty"`
	Labels      interface{}            `yaml:"labels,omitempty"`
	Restart     string                 `yaml:"restart,omitempty"`
	HealthCheck *HealthCheck           `yaml:"healthcheck,omitempty"`
	Extra       map[string]interface{} `yaml:",inline"` // Capture any other fields
}

// BuildConfig represents build configuration
type BuildConfig struct {
	Context    string            `yaml:"context,omitempty"`
	Dockerfile string            `yaml:"dockerfile,omitempty"`
	Args       map[string]string `yaml:"args,omitempty"`
}

// DeployConfig represents deploy configuration (Docker Swarm)
type DeployConfig struct {
	Replicas int                    `yaml:"replicas,omitempty"`
	Extra    map[string]interface{} `yaml:",inline"`
}

// HealthCheck represents health check configuration
type HealthCheck struct {
	Test     []string `yaml:"test,omitempty"`
	Interval string   `yaml:"interval,omitempty"`
	Timeout  string   `yaml:"timeout,omitempty"`
	Retries  int      `yaml:"retries,omitempty"`
}

// Network represents a network definition
type Network struct {
	Driver string                 `yaml:"driver,omitempty"`
	Extra  map[string]interface{} `yaml:",inline"`
}

// Volume represents a volume definition
type Volume struct {
	Driver string                 `yaml:"driver,omitempty"`
	Extra  map[string]interface{} `yaml:",inline"`
}

// Parse reads and parses a docker-compose.yml file
func Parse(path string) (*ComposeFile, error) {
	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	// Parse YAML
	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate
	if err := compose.Validate(); err != nil {
		return nil, err
	}

	return &compose, nil
}

// Validate checks if the compose file is valid
func (c *ComposeFile) Validate() error {
	// Check version
	if c.Version == "" {
		return fmt.Errorf("version field is required")
	}

	// Validate version format (should be 3.x)
	if !strings.HasPrefix(c.Version, "3.") && c.Version != "3" {
		return fmt.Errorf("unsupported version: %s (only version 3.x is supported)", c.Version)
	}

	// Check services
	if len(c.Services) == 0 {
		return fmt.Errorf("at least one service is required")
	}

	// Validate each service
	for name, service := range c.Services {
		if err := service.Validate(name); err != nil {
			return err
		}
	}

	return nil
}

// Validate checks if a service is valid
func (s *Service) Validate(name string) error {
	// Either image or build must be specified
	if s.Image == "" && s.Build == nil {
		return fmt.Errorf("service '%s': either 'image' or 'build' must be specified", name)
	}

	// If both are specified, that's ok (build takes precedence)

	return nil
}

// GetEnvironmentMap converts environment to a map (from array or map format)
func (s *Service) GetEnvironmentMap() map[string]string {
	env := make(map[string]string)

	if s.Environment == nil {
		return env
	}

	switch e := s.Environment.(type) {
	case []interface{}:
		// Array format: ["FOO=bar", "BAZ=qux"]
		for _, item := range e {
			if str, ok := item.(string); ok {
				parts := strings.SplitN(str, "=", 2)
				if len(parts) == 2 {
					env[parts[0]] = parts[1]
				} else {
					env[parts[0]] = ""
				}
			}
		}
	case map[string]interface{}:
		// Map format: {FOO: bar, BAZ: qux}
		for k, v := range e {
			if v == nil {
				env[k] = ""
			} else {
				env[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	return env
}

// GetDependsOnList converts depends_on to a string slice
func (s *Service) GetDependsOnList() []string {
	var deps []string

	if s.DependsOn == nil {
		return deps
	}

	switch d := s.DependsOn.(type) {
	case []interface{}:
		// Array format: ["db", "cache"]
		for _, item := range d {
			if str, ok := item.(string); ok {
				deps = append(deps, str)
			}
		}
	case map[string]interface{}:
		// Map format (v3.x): {db: {condition: service_started}}
		for k := range d {
			deps = append(deps, k)
		}
	}

	return deps
}

// GetCommandSlice converts command to a string slice
func (s *Service) GetCommandSlice() []string {
	if s.Command == nil {
		return nil
	}

	switch c := s.Command.(type) {
	case string:
		// String format: "npm start"
		return []string{"/bin/sh", "-c", c}
	case []interface{}:
		// Array format: ["npm", "start"]
		var cmd []string
		for _, item := range c {
			if str, ok := item.(string); ok {
				cmd = append(cmd, str)
			}
		}
		return cmd
	}

	return nil
}

// GetEntrypointSlice converts entrypoint to a string slice
func (s *Service) GetEntrypointSlice() []string {
	if s.Entrypoint == nil {
		return nil
	}

	switch e := s.Entrypoint.(type) {
	case string:
		// String format: "/app/entrypoint.sh"
		return []string{e}
	case []interface{}:
		// Array format: ["/app/entrypoint.sh", "arg1"]
		var ep []string
		for _, item := range e {
			if str, ok := item.(string); ok {
				ep = append(ep, str)
			}
		}
		return ep
	}

	return nil
}
