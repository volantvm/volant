package converter

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/volantvm/volant/pkg/compose"
	"github.com/volantvm/volant/pkg/volant"
)

// ConversionError represents an error during conversion with helpful context
type ConversionError struct {
	Service string
	Field   string
	Message string
	Help    string
}

func (e *ConversionError) Error() string {
	msg := fmt.Sprintf("service '%s': %s", e.Service, e.Message)
	if e.Help != "" {
		msg += "\n   → " + e.Help
	}
	return msg
}

// Convert converts a Docker Compose file to a Volant stack file
func Convert(composeFile *compose.ComposeFile, stackName string) (*volant.StackFile, error) {
	// Derive stack name from compose file if not provided
	if stackName == "" {
		stackName = "default-stack"
	}

	stack := &volant.StackFile{
		Version:  "1.0",
		Name:     stackName,
		Services: make(map[string]volant.StackService),
		Volumes:  make(map[string]volant.StackVolume),
	}

	// Convert services
	for name, svc := range composeFile.Services {
		stackService, err := convertService(name, svc)
		if err != nil {
			return nil, err
		}
		stack.Services[name] = stackService
	}

	// Convert volumes
	for name := range composeFile.Volumes {
		stack.Volumes[name] = volant.StackVolume{
			Size: "10G", // Default size, can be customized
		}
	}

	// Set defaults and validate
	stack.SetDefaults()
	if err := stack.Validate(); err != nil {
		return nil, err
	}

	return stack, nil
}

// convertService converts a Docker Compose service to a Volant service
func convertService(name string, svc compose.Service) (volant.StackService, error) {
	stackSvc := volant.StackService{}

	// Check for unsupported 'build' directive
	if svc.Build != nil {
		return stackSvc, &ConversionError{
			Service: name,
			Field:   "build",
			Message: "'build' directive not supported",
			Help:    "Use Fledge to build images: fledge build -f Dockerfile",
		}
	}

	// Image is required (validated earlier, but double-check)
	if svc.Image == "" {
		return stackSvc, &ConversionError{
			Service: name,
			Field:   "image",
			Message: "image is required",
			Help:    "Specify an image or use Fledge to build one",
		}
	}
	stackSvc.Image = svc.Image

	// Convert ports (direct copy)
	stackSvc.Ports = svc.Ports

	// Convert environment (array or map → map)
	stackSvc.Env = svc.GetEnvironmentMap()

	// Convert depends_on (array or map → array)
	stackSvc.DependsOn = svc.GetDependsOnList()

	// Convert volumes
	volumes, err := convertVolumes(name, svc.Volumes)
	if err != nil {
		return stackSvc, err
	}
	stackSvc.Volumes = volumes

	// Convert command
	if cmd := svc.GetCommandSlice(); cmd != nil {
		// In Volant, command goes to args
		stackSvc.Args = cmd
	}

	// Convert entrypoint
	if ep := svc.GetEntrypointSlice(); ep != nil {
		stackSvc.Entrypoint = ep
	}

	// Check for unsupported deploy.replicas
	if svc.Deploy != nil && svc.Deploy.Replicas > 1 {
		return stackSvc, &ConversionError{
			Service: name,
			Field:   "deploy.replicas",
			Message: fmt.Sprintf("deploy.replicas=%d not supported yet", svc.Deploy.Replicas),
			Help:    "Phase 2 feature (horizontal scaling)",
		}
	}

	return stackSvc, nil
}

// convertVolumes converts Docker Compose volume mounts to Volant volume mounts
// Rejects bind mounts, only allows named volumes
func convertVolumes(serviceName string, volumes []string) ([]string, error) {
	var converted []string

	for _, vol := range volumes {
		// Parse volume string: can be "volume_name:/path" or "./local:/path" (bind mount)
		parts := strings.SplitN(vol, ":", 2)
		if len(parts) != 2 {
			return nil, &ConversionError{
				Service: serviceName,
				Field:   "volumes",
				Message: fmt.Sprintf("invalid volume format: %s", vol),
				Help:    "Use format: volume_name:/container/path",
			}
		}

		source := parts[0]
		target := parts[1]

		// Check for bind mount (starts with . or /)
		if isBindMount(source) {
			return nil, &ConversionError{
				Service: serviceName,
				Field:   "volumes",
				Message: fmt.Sprintf("bind mount '%s' not supported", vol),
				Help:    "Use named volumes instead: volumes: [data:/data]",
			}
		}

		// Named volume - keep as is
		converted = append(converted, fmt.Sprintf("%s:%s", source, target))
	}

	return converted, nil
}

// isBindMount checks if a volume source is a bind mount (local path)
func isBindMount(source string) bool {
	// Bind mounts start with:
	// - . (relative path like ./data or ../config)
	// - / (absolute path like /var/data)
	// - ~ (home directory like ~/data)
	return strings.HasPrefix(source, ".") ||
		strings.HasPrefix(source, "/") ||
		strings.HasPrefix(source, "~")
}

// DeriveStackName derives a stack name from a compose file path
func DeriveStackName(composePath string) string {
	// Get directory name as stack name
	dir := filepath.Dir(composePath)
	name := filepath.Base(dir)

	// If in current directory, use "app"
	if name == "." {
		name = "app"
	}

	// Sanitize name (lowercase, replace spaces/special chars with -)
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")

	return name
}
