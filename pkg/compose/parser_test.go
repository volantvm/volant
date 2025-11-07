package compose

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse_Valid(t *testing.T) {
	// Create a temporary compose file
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "docker-compose.yml")

	content := `version: "3.8"
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
    environment:
      - FOO=bar
      - BAZ=qux
    depends_on:
      - db
  db:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: secret
`

	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Parse
	compose, err := Parse(composePath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Validate
	if compose.Version != "3.8" {
		t.Errorf("expected version 3.8, got %s", compose.Version)
	}

	if len(compose.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(compose.Services))
	}

	// Check web service
	web, ok := compose.Services["web"]
	if !ok {
		t.Fatal("web service not found")
	}

	if web.Image != "nginx:latest" {
		t.Errorf("expected nginx:latest, got %s", web.Image)
	}

	if len(web.Ports) != 1 || web.Ports[0] != "8080:80" {
		t.Errorf("unexpected ports: %v", web.Ports)
	}

	env := web.GetEnvironmentMap()
	if env["FOO"] != "bar" || env["BAZ"] != "qux" {
		t.Errorf("unexpected environment: %v", env)
	}

	deps := web.GetDependsOnList()
	if len(deps) != 1 || deps[0] != "db" {
		t.Errorf("unexpected dependencies: %v", deps)
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "docker-compose.yml")

	content := `version: "3.8"
services:
  web:
    image nginx:latest
    - invalid yaml
`

	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Parse(composePath)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestParse_MissingVersion(t *testing.T) {
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "docker-compose.yml")

	content := `services:
  web:
    image: nginx:latest
`

	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Parse(composePath)
	if err == nil {
		t.Error("expected error for missing version, got nil")
	}
}

func TestParse_UnsupportedVersion(t *testing.T) {
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "docker-compose.yml")

	content := `version: "2.1"
services:
  web:
    image: nginx:latest
`

	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Parse(composePath)
	if err == nil {
		t.Error("expected error for unsupported version, got nil")
	}
}

func TestParse_NoServices(t *testing.T) {
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "docker-compose.yml")

	content := `version: "3.8"
services: {}
`

	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Parse(composePath)
	if err == nil {
		t.Error("expected error for no services, got nil")
	}
}

func TestParse_NoImageOrBuild(t *testing.T) {
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "docker-compose.yml")

	content := `version: "3.8"
services:
  web:
    ports:
      - "8080:80"
`

	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Parse(composePath)
	if err == nil {
		t.Error("expected error for missing image/build, got nil")
	}
}

func TestGetEnvironmentMap_Array(t *testing.T) {
	service := Service{
		Environment: []interface{}{"FOO=bar", "BAZ=qux", "EMPTY="},
	}

	env := service.GetEnvironmentMap()

	if env["FOO"] != "bar" {
		t.Errorf("expected FOO=bar, got FOO=%s", env["FOO"])
	}
	if env["BAZ"] != "qux" {
		t.Errorf("expected BAZ=qux, got BAZ=%s", env["BAZ"])
	}
	if env["EMPTY"] != "" {
		t.Errorf("expected EMPTY=, got EMPTY=%s", env["EMPTY"])
	}
}

func TestGetEnvironmentMap_Map(t *testing.T) {
	service := Service{
		Environment: map[string]interface{}{
			"FOO":   "bar",
			"BAZ":   "qux",
			"NUM":   123,
			"EMPTY": nil,
		},
	}

	env := service.GetEnvironmentMap()

	if env["FOO"] != "bar" {
		t.Errorf("expected FOO=bar, got FOO=%s", env["FOO"])
	}
	if env["BAZ"] != "qux" {
		t.Errorf("expected BAZ=qux, got BAZ=%s", env["BAZ"])
	}
	if env["NUM"] != "123" {
		t.Errorf("expected NUM=123, got NUM=%s", env["NUM"])
	}
	if env["EMPTY"] != "" {
		t.Errorf("expected EMPTY=, got EMPTY=%s", env["EMPTY"])
	}
}

func TestGetDependsOnList_Array(t *testing.T) {
	service := Service{
		DependsOn: []interface{}{"db", "cache", "queue"},
	}

	deps := service.GetDependsOnList()

	if len(deps) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(deps))
	}
	if deps[0] != "db" || deps[1] != "cache" || deps[2] != "queue" {
		t.Errorf("unexpected dependencies: %v", deps)
	}
}

func TestGetDependsOnList_Map(t *testing.T) {
	service := Service{
		DependsOn: map[string]interface{}{
			"db":    map[string]interface{}{"condition": "service_healthy"},
			"cache": map[string]interface{}{"condition": "service_started"},
		},
	}

	deps := service.GetDependsOnList()

	if len(deps) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(deps))
	}

	// Note: map iteration order is not guaranteed, so just check both are present
	depsMap := make(map[string]bool)
	for _, d := range deps {
		depsMap[d] = true
	}

	if !depsMap["db"] || !depsMap["cache"] {
		t.Errorf("expected db and cache, got %v", deps)
	}
}

func TestGetCommandSlice_String(t *testing.T) {
	service := Service{
		Command: "npm start",
	}

	cmd := service.GetCommandSlice()

	if len(cmd) != 3 || cmd[0] != "/bin/sh" || cmd[1] != "-c" || cmd[2] != "npm start" {
		t.Errorf("unexpected command: %v", cmd)
	}
}

func TestGetCommandSlice_Array(t *testing.T) {
	service := Service{
		Command: []interface{}{"npm", "run", "start"},
	}

	cmd := service.GetCommandSlice()

	if len(cmd) != 3 || cmd[0] != "npm" || cmd[1] != "run" || cmd[2] != "start" {
		t.Errorf("unexpected command: %v", cmd)
	}
}

func TestGetEntrypointSlice_String(t *testing.T) {
	service := Service{
		Entrypoint: "/app/entrypoint.sh",
	}

	ep := service.GetEntrypointSlice()

	if len(ep) != 1 || ep[0] != "/app/entrypoint.sh" {
		t.Errorf("unexpected entrypoint: %v", ep)
	}
}

func TestGetEntrypointSlice_Array(t *testing.T) {
	service := Service{
		Entrypoint: []interface{}{"/app/entrypoint.sh", "arg1", "arg2"},
	}

	ep := service.GetEntrypointSlice()

	if len(ep) != 3 || ep[0] != "/app/entrypoint.sh" || ep[1] != "arg1" || ep[2] != "arg2" {
		t.Errorf("unexpected entrypoint: %v", ep)
	}
}

func TestParse_ComplexCompose(t *testing.T) {
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "docker-compose.yml")

	content := `version: "3.9"
services:
  web:
    image: nginx:latest
    container_name: my-web
    ports:
      - "8080:80"
      - "8443:443"
    environment:
      NGINX_HOST: example.com
      NGINX_PORT: 80
    volumes:
      - web-data:/var/www/html
    depends_on:
      - db
      - cache
    restart: always
    networks:
      - frontend
      - backend

  db:
    image: postgres:15
    environment:
      - POSTGRES_DB=mydb
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=secret
    volumes:
      - db-data:/var/lib/postgresql/data
    networks:
      - backend

  cache:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - cache-data:/data
    networks:
      - backend

networks:
  frontend:
  backend:

volumes:
  web-data:
  db-data:
  cache-data:
`

	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	compose, err := Parse(composePath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check structure
	if len(compose.Services) != 3 {
		t.Errorf("expected 3 services, got %d", len(compose.Services))
	}

	if len(compose.Networks) != 2 {
		t.Errorf("expected 2 networks, got %d", len(compose.Networks))
	}

	if len(compose.Volumes) != 3 {
		t.Errorf("expected 3 volumes, got %d", len(compose.Volumes))
	}

	// Check web service details
	web := compose.Services["web"]
	if web.ContainerName != "my-web" {
		t.Errorf("expected container_name my-web, got %s", web.ContainerName)
	}

	if len(web.Ports) != 2 {
		t.Errorf("expected 2 ports, got %d", len(web.Ports))
	}

	env := web.GetEnvironmentMap()
	if env["NGINX_HOST"] != "example.com" {
		t.Errorf("expected NGINX_HOST=example.com, got %s", env["NGINX_HOST"])
	}

	deps := web.GetDependsOnList()
	if len(deps) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(deps))
	}

	// Check cache service command
	cache := compose.Services["cache"]
	cmd := cache.GetCommandSlice()
	if cmd[2] != "redis-server --appendonly yes" {
		t.Errorf("unexpected command: %v", cmd)
	}
}
