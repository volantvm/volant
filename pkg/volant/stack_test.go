package volant

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse_ValidStack(t *testing.T) {
	tmpDir := t.TempDir()
	stackPath := filepath.Join(tmpDir, "volant.yaml")

	content := `version: "1.0"
name: "test-stack"
services:
  web:
    image: nginx:latest
    memory: 512
    vcpus: 1
    ports:
      - "8080:80"
    environment:
      FOO: bar
    depends_on:
      - db
  db:
    image: postgres:15
    memory: 1024
    vcpus: 2
    environment:
      POSTGRES_PASSWORD: secret
volumes:
  db-data:
    size: "10G"
`

	if err := os.WriteFile(stackPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	stack, err := Parse(stackPath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if stack.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", stack.Version)
	}

	if stack.Name != "test-stack" {
		t.Errorf("expected name test-stack, got %s", stack.Name)
	}

	if len(stack.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(stack.Services))
	}

	// Check web service
	web := stack.Services["web"]
	if web.Image != "nginx:latest" {
		t.Errorf("expected nginx:latest, got %s", web.Image)
	}
	if web.Memory != 512 {
		t.Errorf("expected memory 512, got %d", web.Memory)
	}
	if web.VCPUs != 1 {
		t.Errorf("expected vcpus 1, got %d", web.VCPUs)
	}
	if len(web.Ports) != 1 || web.Ports[0] != "8080:80" {
		t.Errorf("unexpected ports: %v", web.Ports)
	}
	if web.Env["FOO"] != "bar" {
		t.Errorf("expected FOO=bar, got %v", web.Env)
	}
	if len(web.DependsOn) != 1 || web.DependsOn[0] != "db" {
		t.Errorf("unexpected depends_on: %v", web.DependsOn)
	}
}

func TestWriteYAML(t *testing.T) {
	tmpDir := t.TempDir()
	stackPath := filepath.Join(tmpDir, "volant.yaml")

	stack := &StackFile{
		Version: "1.0",
		Name:    "test-stack",
		Services: map[string]StackService{
			"web": {
				Image:  "nginx:latest",
				Memory: 512,
				VCPUs:  1,
				Ports:  []string{"8080:80"},
				Env: map[string]string{
					"FOO": "bar",
				},
			},
		},
	}

	if err := WriteYAML(stack, stackPath); err != nil {
		t.Fatalf("WriteYAML() error = %v", err)
	}

	// Read back and verify
	parsed, err := Parse(stackPath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if parsed.Name != stack.Name {
		t.Errorf("expected name %s, got %s", stack.Name, parsed.Name)
	}

	if len(parsed.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(parsed.Services))
	}

	web := parsed.Services["web"]
	if web.Image != "nginx:latest" {
		t.Errorf("expected nginx:latest, got %s", web.Image)
	}
}

func TestValidate_MissingVersion(t *testing.T) {
	stack := &StackFile{
		Name: "test",
		Services: map[string]StackService{
			"web": {Image: "nginx"},
		},
	}

	if err := stack.Validate(); err == nil {
		t.Error("expected error for missing version, got nil")
	}
}

func TestValidate_UnsupportedVersion(t *testing.T) {
	stack := &StackFile{
		Version: "2.0",
		Name:    "test",
		Services: map[string]StackService{
			"web": {Image: "nginx"},
		},
	}

	if err := stack.Validate(); err == nil {
		t.Error("expected error for unsupported version, got nil")
	}
}

func TestValidate_MissingName(t *testing.T) {
	stack := &StackFile{
		Version: "1.0",
		Services: map[string]StackService{
			"web": {Image: "nginx"},
		},
	}

	if err := stack.Validate(); err == nil {
		t.Error("expected error for missing name, got nil")
	}
}

func TestValidate_NoServices(t *testing.T) {
	stack := &StackFile{
		Version:  "1.0",
		Name:     "test",
		Services: map[string]StackService{},
	}

	if err := stack.Validate(); err == nil {
		t.Error("expected error for no services, got nil")
	}
}

func TestValidate_MissingImage(t *testing.T) {
	stack := &StackFile{
		Version: "1.0",
		Name:    "test",
		Services: map[string]StackService{
			"web": {Memory: 512},
		},
	}

	if err := stack.Validate(); err == nil {
		t.Error("expected error for missing image, got nil")
	}
}

func TestValidate_InvalidDependency(t *testing.T) {
	stack := &StackFile{
		Version: "1.0",
		Name:    "test",
		Services: map[string]StackService{
			"web": {
				Image:     "nginx",
				DependsOn: []string{"nonexistent"},
			},
		},
	}

	if err := stack.Validate(); err == nil {
		t.Error("expected error for invalid dependency, got nil")
	}
}

func TestValidate_NegativeMemory(t *testing.T) {
	stack := &StackFile{
		Version: "1.0",
		Name:    "test",
		Services: map[string]StackService{
			"web": {
				Image:  "nginx",
				Memory: -1,
			},
		},
	}

	if err := stack.Validate(); err == nil {
		t.Error("expected error for negative memory, got nil")
	}
}

func TestValidate_NegativeVCPUs(t *testing.T) {
	stack := &StackFile{
		Version: "1.0",
		Name:    "test",
		Services: map[string]StackService{
			"web": {
				Image: "nginx",
				VCPUs: -1,
			},
		},
	}

	if err := stack.Validate(); err == nil {
		t.Error("expected error for negative vcpus, got nil")
	}
}

func TestSetDefaults(t *testing.T) {
	stack := &StackFile{
		Version: "1.0",
		Name:    "test",
		Services: map[string]StackService{
			"web": {
				Image: "nginx",
				// No memory or vcpus specified
			},
		},
	}

	stack.SetDefaults()

	web := stack.Services["web"]
	if web.Memory != 512 {
		t.Errorf("expected default memory 512, got %d", web.Memory)
	}
	if web.VCPUs != 1 {
		t.Errorf("expected default vcpus 1, got %d", web.VCPUs)
	}

	// Check default network
	if len(stack.Networks) == 0 {
		t.Error("expected default network to be created")
	}
	defaultNet, exists := stack.Networks["default"]
	if !exists {
		t.Error("expected default network to exist")
	}
	if defaultNet.Subnet != "192.168.127.0/24" {
		t.Errorf("expected default subnet 192.168.127.0/24, got %s", defaultNet.Subnet)
	}
}

func TestComplexStack(t *testing.T) {
	stack := &StackFile{
		Version: "1.0",
		Name:    "wordpress-stack",
		Services: map[string]StackService{
			"web": {
				Image:  "wordpress:latest",
				Memory: 512,
				VCPUs:  1,
				Ports:  []string{"8080:80"},
				Env: map[string]string{
					"WORDPRESS_DB_HOST":     "db.svc.volant",
					"WORDPRESS_DB_USER":     "wordpress",
					"WORDPRESS_DB_PASSWORD": "secret",
					"WORDPRESS_DB_NAME":     "wordpress",
				},
				Volumes:   []string{"wp-data:/var/www/html"},
				DependsOn: []string{"db"},
			},
			"db": {
				Image:  "mysql:8",
				Memory: 1024,
				VCPUs:  2,
				Env: map[string]string{
					"MYSQL_ROOT_PASSWORD": "rootsecret",
					"MYSQL_DATABASE":      "wordpress",
					"MYSQL_USER":          "wordpress",
					"MYSQL_PASSWORD":      "secret",
				},
				Volumes: []string{"db-data:/var/lib/mysql"},
			},
		},
		Volumes: map[string]StackVolume{
			"wp-data": {Size: "1G"},
			"db-data": {Size: "10G"},
		},
	}

	if err := stack.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}

	// Write and read back
	tmpDir := t.TempDir()
	stackPath := filepath.Join(tmpDir, "wordpress.yaml")

	if err := WriteYAML(stack, stackPath); err != nil {
		t.Fatalf("WriteYAML() error = %v", err)
	}

	parsed, err := Parse(stackPath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if parsed.Name != "wordpress-stack" {
		t.Errorf("expected name wordpress-stack, got %s", parsed.Name)
	}

	if len(parsed.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(parsed.Services))
	}

	if len(parsed.Volumes) != 2 {
		t.Errorf("expected 2 volumes, got %d", len(parsed.Volumes))
	}
}
