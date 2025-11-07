package converter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/volantvm/volant/pkg/compose"
	"github.com/volantvm/volant/pkg/volant"
)

// TestIntegration_WordPress tests converting a WordPress stack
func TestIntegration_WordPress(t *testing.T) {
	composePath := "../../testdata/wordpress/docker-compose.yml"
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		t.Skip("Test data not found")
	}

	// Parse
	composeFile, err := compose.Parse(composePath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Convert
	stack, err := Convert(composeFile, "wordpress-stack")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// Validate structure
	if stack.Name != "wordpress-stack" {
		t.Errorf("expected name wordpress-stack, got %s", stack.Name)
	}

	if len(stack.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(stack.Services))
	}

	// Check wordpress service
	wp, exists := stack.Services["wordpress"]
	if !exists {
		t.Fatal("wordpress service not found")
	}

	if wp.Image != "wordpress:latest" {
		t.Errorf("expected wordpress:latest, got %s", wp.Image)
	}

	if len(wp.Ports) != 1 {
		t.Errorf("expected 1 port, got %d", len(wp.Ports))
	}

	if len(wp.DependsOn) != 1 || wp.DependsOn[0] != "db" {
		t.Errorf("expected depends_on: [db], got %v", wp.DependsOn)
	}

	if wp.Env["WORDPRESS_DB_HOST"] != "db" {
		t.Errorf("expected WORDPRESS_DB_HOST=db, got %s", wp.Env["WORDPRESS_DB_HOST"])
	}

	// Check db service
	db, exists := stack.Services["db"]
	if !exists {
		t.Fatal("db service not found")
	}

	if db.Image != "mysql:8.0" {
		t.Errorf("expected mysql:8.0, got %s", db.Image)
	}

	if db.Env["MYSQL_DATABASE"] != "wordpress" {
		t.Errorf("expected MYSQL_DATABASE=wordpress, got %s", db.Env["MYSQL_DATABASE"])
	}

	// Check volumes
	if len(stack.Volumes) != 2 {
		t.Errorf("expected 2 volumes, got %d", len(stack.Volumes))
	}
}

// TestIntegration_NodeJsMongo tests converting a Node.js + MongoDB stack
func TestIntegration_NodeJsMongo(t *testing.T) {
	composePath := "../../testdata/nodejs-mongo/docker-compose.yml"
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		t.Skip("Test data not found")
	}

	composeFile, err := compose.Parse(composePath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	stack, err := Convert(composeFile, "nodejs-stack")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// Check app service
	app, exists := stack.Services["app"]
	if !exists {
		t.Fatal("app service not found")
	}

	if app.Image != "node:18-alpine" {
		t.Errorf("expected node:18-alpine, got %s", app.Image)
	}

	// Check command was converted to args
	if len(app.Args) != 2 || app.Args[0] != "node" || app.Args[1] != "server.js" {
		t.Errorf("expected args: [node server.js], got %v", app.Args)
	}

	// Check environment
	if app.Env["NODE_ENV"] != "production" {
		t.Errorf("expected NODE_ENV=production, got %s", app.Env["NODE_ENV"])
	}

	// Check mongo service
	mongo, exists := stack.Services["mongo"]
	if !exists {
		t.Fatal("mongo service not found")
	}

	if mongo.Env["MONGO_INITDB_ROOT_USERNAME"] != "admin" {
		t.Errorf("expected MONGO_INITDB_ROOT_USERNAME=admin, got %s", mongo.Env["MONGO_INITDB_ROOT_USERNAME"])
	}
}

// TestIntegration_NginxProxy tests converting an Nginx reverse proxy setup
func TestIntegration_NginxProxy(t *testing.T) {
	composePath := "../../testdata/nginx-proxy/docker-compose.yml"
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		t.Skip("Test data not found")
	}

	composeFile, err := compose.Parse(composePath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	stack, err := Convert(composeFile, "nginx-stack")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	if len(stack.Services) != 3 {
		t.Errorf("expected 3 services, got %d", len(stack.Services))
	}

	// Check nginx dependencies
	nginx, exists := stack.Services["nginx"]
	if !exists {
		t.Fatal("nginx service not found")
	}

	if len(nginx.DependsOn) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(nginx.DependsOn))
	}

	depsMap := make(map[string]bool)
	for _, dep := range nginx.DependsOn {
		depsMap[dep] = true
	}

	if !depsMap["backend1"] || !depsMap["backend2"] {
		t.Errorf("expected dependencies on backend1 and backend2, got %v", nginx.DependsOn)
	}
}

// TestIntegration_ErrorOnBuild tests that build directive is rejected
func TestIntegration_ErrorOnBuild(t *testing.T) {
	composePath := "../../testdata/error-build/docker-compose.yml"
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		t.Skip("Test data not found")
	}

	composeFile, err := compose.Parse(composePath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	_, err = Convert(composeFile, "test")
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

	if convErr.Field != "build" {
		t.Errorf("expected field 'build', got %s", convErr.Field)
	}
}

// TestIntegration_ErrorOnBindMount tests that bind mounts are rejected
func TestIntegration_ErrorOnBindMount(t *testing.T) {
	composePath := "../../testdata/error-bindmount/docker-compose.yml"
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		t.Skip("Test data not found")
	}

	composeFile, err := compose.Parse(composePath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	_, err = Convert(composeFile, "test")
	if err == nil {
		t.Fatal("expected error for bind mount, got nil")
	}

	convErr, ok := err.(*ConversionError)
	if !ok {
		t.Fatalf("expected ConversionError, got %T", err)
	}

	if convErr.Service != "app" {
		t.Errorf("expected service 'app', got %s", convErr.Service)
	}

	if convErr.Field != "volumes" {
		t.Errorf("expected field 'volumes', got %s", convErr.Field)
	}
}

// TestIntegration_RoundTrip tests that we can convert and write YAML correctly
func TestIntegration_RoundTrip(t *testing.T) {
	composePath := "../../testdata/wordpress/docker-compose.yml"
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		t.Skip("Test data not found")
	}

	// Parse compose
	composeFile, err := compose.Parse(composePath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Convert
	stack, err := Convert(composeFile, "test-stack")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	// Write to temp file
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "volant.yaml")

	if err := volant.WriteYAML(stack, outputPath); err != nil {
		t.Fatalf("WriteYAML() error = %v", err)
	}

	// Read back
	parsed, err := volant.Parse(outputPath)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Validate
	if parsed.Name != "test-stack" {
		t.Errorf("expected name test-stack, got %s", parsed.Name)
	}

	if len(parsed.Services) != len(stack.Services) {
		t.Errorf("expected %d services, got %d", len(stack.Services), len(parsed.Services))
	}

	// Check service details are preserved
	for name, orig := range stack.Services {
		parsed, exists := parsed.Services[name]
		if !exists {
			t.Errorf("service %s missing after round-trip", name)
			continue
		}

		if parsed.Image != orig.Image {
			t.Errorf("service %s: expected image %s, got %s", name, orig.Image, parsed.Image)
		}
	}
}

// TestIntegration_AllExamples tests all example files can be parsed and converted
func TestIntegration_AllExamples(t *testing.T) {
	testdataDir := "../../testdata"
	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Skip("Testdata directory not found")
	}

	successCount := 0
	errorCount := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip error examples (they're supposed to fail)
		if filepath.HasPrefix(entry.Name(), "error-") {
			continue
		}

		composePath := filepath.Join(testdataDir, entry.Name(), "docker-compose.yml")
		if _, err := os.Stat(composePath); os.IsNotExist(err) {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			composeFile, err := compose.Parse(composePath)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			stack, err := Convert(composeFile, entry.Name())
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}

			// Basic validation
			if err := stack.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}

			if len(stack.Services) == 0 {
				t.Error("expected at least one service")
			}

			successCount++
		})
	}

	if successCount == 0 {
		t.Skip("No test examples found")
	}

	t.Logf("Successfully converted %d examples, %d errors", successCount, errorCount)
}
