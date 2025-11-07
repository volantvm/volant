package converter

import (
	"strings"
	"testing"

	"github.com/volantvm/volant/pkg/compose"
)

func TestConvert_Simple(t *testing.T) {
	composeFile := &compose.ComposeFile{
		Version: "3.8",
		Services: map[string]compose.Service{
			"web": {
				Image: "nginx:latest",
				Ports: []string{"8080:80"},
			},
		},
	}

	stack, err := Convert(composeFile, "test-stack")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	if stack.Name != "test-stack" {
		t.Errorf("expected name test-stack, got %s", stack.Name)
	}

	if len(stack.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(stack.Services))
	}

	web := stack.Services["web"]
	if web.Image != "nginx:latest" {
		t.Errorf("expected nginx:latest, got %s", web.Image)
	}

	if len(web.Ports) != 1 || web.Ports[0] != "8080:80" {
		t.Errorf("unexpected ports: %v", web.Ports)
	}

	// Check defaults were applied
	if web.Memory != 512 {
		t.Errorf("expected default memory 512, got %d", web.Memory)
	}
	if web.VCPUs != 1 {
		t.Errorf("expected default vcpus 1, got %d", web.VCPUs)
	}
}

func TestConvert_WithEnvironment(t *testing.T) {
	composeFile := &compose.ComposeFile{
		Version: "3.8",
		Services: map[string]compose.Service{
			"app": {
				Image: "myapp:1.0",
				Environment: []interface{}{
					"FOO=bar",
					"BAZ=qux",
				},
			},
		},
	}

	stack, err := Convert(composeFile, "test")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	app := stack.Services["app"]
	if app.Env["FOO"] != "bar" {
		t.Errorf("expected FOO=bar, got %v", app.Env)
	}
	if app.Env["BAZ"] != "qux" {
		t.Errorf("expected BAZ=qux, got %v", app.Env)
	}
}

func TestConvert_WithDependencies(t *testing.T) {
	composeFile := &compose.ComposeFile{
		Version: "3.8",
		Services: map[string]compose.Service{
			"web": {
				Image:     "nginx",
				DependsOn: []interface{}{"db", "cache"},
			},
			"db": {
				Image: "postgres",
			},
			"cache": {
				Image: "redis",
			},
		},
	}

	stack, err := Convert(composeFile, "test")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	web := stack.Services["web"]
	if len(web.DependsOn) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(web.DependsOn))
	}
}

func TestConvert_WithNamedVolumes(t *testing.T) {
	composeFile := &compose.ComposeFile{
		Version: "3.8",
		Services: map[string]compose.Service{
			"db": {
				Image:   "postgres",
				Volumes: []string{"db-data:/var/lib/postgresql/data"},
			},
		},
		Volumes: map[string]compose.Volume{
			"db-data": {},
		},
	}

	stack, err := Convert(composeFile, "test")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	db := stack.Services["db"]
	if len(db.Volumes) != 1 {
		t.Errorf("expected 1 volume, got %d", len(db.Volumes))
	}
	if db.Volumes[0] != "db-data:/var/lib/postgresql/data" {
		t.Errorf("unexpected volume: %s", db.Volumes[0])
	}

	// Check volume definition
	if len(stack.Volumes) != 1 {
		t.Errorf("expected 1 volume definition, got %d", len(stack.Volumes))
	}
	if _, exists := stack.Volumes["db-data"]; !exists {
		t.Error("expected db-data volume definition")
	}
}

func TestConvert_WithCommand(t *testing.T) {
	composeFile := &compose.ComposeFile{
		Version: "3.8",
		Services: map[string]compose.Service{
			"app": {
				Image:   "myapp",
				Command: []interface{}{"npm", "start"},
			},
		},
	}

	stack, err := Convert(composeFile, "test")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	app := stack.Services["app"]
	if len(app.Args) != 2 || app.Args[0] != "npm" || app.Args[1] != "start" {
		t.Errorf("unexpected args: %v", app.Args)
	}
}

func TestConvert_WithEntrypoint(t *testing.T) {
	composeFile := &compose.ComposeFile{
		Version: "3.8",
		Services: map[string]compose.Service{
			"app": {
				Image:      "myapp",
				Entrypoint: []interface{}{"/app/entrypoint.sh", "arg1"},
			},
		},
	}

	stack, err := Convert(composeFile, "test")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	app := stack.Services["app"]
	if len(app.Entrypoint) != 2 || app.Entrypoint[0] != "/app/entrypoint.sh" {
		t.Errorf("unexpected entrypoint: %v", app.Entrypoint)
	}
}

func TestConvert_ErrorOnBuild(t *testing.T) {
	composeFile := &compose.ComposeFile{
		Version: "3.8",
		Services: map[string]compose.Service{
			"app": {
				Build: &compose.BuildConfig{
					Context: "./app",
				},
				Image: "myapp:latest",
			},
		},
	}

	_, err := Convert(composeFile, "test")
	if err == nil {
		t.Fatal("expected error for build directive, got nil")
	}

	convErr, ok := err.(*ConversionError)
	if !ok {
		t.Fatalf("expected ConversionError, got %T", err)
	}

	if convErr.Service != "app" {
		t.Errorf("expected service 'app', got %s", convErr.Service)
	}

	if !strings.Contains(err.Error(), "build") {
		t.Errorf("expected error to mention 'build', got: %s", err.Error())
	}

	if !strings.Contains(err.Error(), "Fledge") {
		t.Errorf("expected error to mention 'Fledge', got: %s", err.Error())
	}
}

func TestConvert_ErrorOnBindMount(t *testing.T) {
	tests := []struct {
		name   string
		volume string
	}{
		{"relative path", "./data:/data"},
		{"absolute path", "/host/data:/data"},
		{"home directory", "~/data:/data"},
		{"parent directory", "../data:/data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			composeFile := &compose.ComposeFile{
				Version: "3.8",
				Services: map[string]compose.Service{
					"app": {
						Image:   "myapp",
						Volumes: []string{tt.volume},
					},
				},
			}

			_, err := Convert(composeFile, "test")
			if err == nil {
				t.Fatalf("expected error for bind mount %s, got nil", tt.volume)
			}

			_, ok := err.(*ConversionError)
			if !ok {
				t.Fatalf("expected ConversionError, got %T", err)
			}

			if !strings.Contains(err.Error(), "bind mount") {
				t.Errorf("expected error to mention 'bind mount', got: %s", err.Error())
			}
		})
	}
}

func TestConvert_ErrorOnReplicas(t *testing.T) {
	composeFile := &compose.ComposeFile{
		Version: "3.8",
		Services: map[string]compose.Service{
			"app": {
				Image: "myapp",
				Deploy: &compose.DeployConfig{
					Replicas: 3,
				},
			},
		},
	}

	_, err := Convert(composeFile, "test")
	if err == nil {
		t.Fatal("expected error for replicas > 1, got nil")
	}

	_, ok := err.(*ConversionError)
	if !ok {
		t.Fatalf("expected ConversionError, got %T", err)
	}

	if !strings.Contains(err.Error(), "replicas") {
		t.Errorf("expected error to mention 'replicas', got: %s", err.Error())
	}
}

func TestConvert_ComplexStack(t *testing.T) {
	composeFile := &compose.ComposeFile{
		Version: "3.8",
		Services: map[string]compose.Service{
			"web": {
				Image: "nginx:latest",
				Ports: []string{"8080:80", "8443:443"},
				Environment: map[string]interface{}{
					"NGINX_HOST": "example.com",
					"NGINX_PORT": "80",
				},
				Volumes:   []string{"web-data:/var/www/html"},
				DependsOn: []interface{}{"db", "cache"},
			},
			"db": {
				Image: "postgres:15",
				Environment: []interface{}{
					"POSTGRES_DB=mydb",
					"POSTGRES_USER=user",
					"POSTGRES_PASSWORD=secret",
				},
				Volumes: []string{"db-data:/var/lib/postgresql/data"},
			},
			"cache": {
				Image:   "redis:7-alpine",
				Command: "redis-server --appendonly yes",
			},
		},
		Volumes: map[string]compose.Volume{
			"web-data": {},
			"db-data":  {},
		},
	}

	stack, err := Convert(composeFile, "my-stack")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// Validate structure
	if stack.Name != "my-stack" {
		t.Errorf("expected name my-stack, got %s", stack.Name)
	}

	if len(stack.Services) != 3 {
		t.Errorf("expected 3 services, got %d", len(stack.Services))
	}

	if len(stack.Volumes) != 2 {
		t.Errorf("expected 2 volumes, got %d", len(stack.Volumes))
	}

	// Check web service
	web := stack.Services["web"]
	if len(web.Ports) != 2 {
		t.Errorf("expected 2 ports, got %d", len(web.Ports))
	}
	if len(web.DependsOn) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(web.DependsOn))
	}
	if web.Env["NGINX_HOST"] != "example.com" {
		t.Errorf("expected NGINX_HOST=example.com, got %s", web.Env["NGINX_HOST"])
	}

	// Check db service
	db := stack.Services["db"]
	if db.Env["POSTGRES_DB"] != "mydb" {
		t.Errorf("expected POSTGRES_DB=mydb, got %s", db.Env["POSTGRES_DB"])
	}

	// Check cache service
	cache := stack.Services["cache"]
	if len(cache.Args) == 0 {
		t.Error("expected cache to have args from command")
	}
}

func TestDeriveStackName(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/home/user/myapp/docker-compose.yml", "myapp"},
		{"/home/user/My App/docker-compose.yml", "my-app"},
		{"./docker-compose.yml", "app"},
		{"/home/user/project_name/docker-compose.yml", "project-name"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := DeriveStackName(tt.path)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestIsBindMount(t *testing.T) {
	tests := []struct {
		source   string
		expected bool
	}{
		{"./data", true},
		{"../config", true},
		{"/var/data", true},
		{"~/documents", true},
		{"volume-name", false},
		{"my-data", false},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			result := isBindMount(tt.source)
			if result != tt.expected {
				t.Errorf("isBindMount(%s) = %v, expected %v", tt.source, result, tt.expected)
			}
		})
	}
}
