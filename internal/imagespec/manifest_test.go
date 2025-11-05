package imagespec

import (
	"encoding/json"
	"testing"
)

// TestPortExposeUnmarshal tests that port expose config is correctly parsed from JSON
func TestPortExposeUnmarshal(t *testing.T) {
	manifestJSON := `{
		"schema_version": "v1",
		"name": "test",
		"version": "1.0.0",
		"runtime": "fc",
		"network": {
			"mode": "bridged",
			"expose": [
				{"port": 3000, "protocol": "tcp"},
				{"port": 8080, "protocol": "tcp", "host_port": 9000},
				{"port": 5432, "protocol": "udp"}
			]
		}
	}`

	var manifest Manifest
	if err := json.Unmarshal([]byte(manifestJSON), &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	// Normalize to apply defaults
	manifest.Normalize()

	// Verify network config exists
	if manifest.Network == nil {
		t.Fatal("expected network config to be populated")
	}

	// Verify expose ports
	if len(manifest.Network.Expose) != 3 {
		t.Errorf("expected 3 exposed ports, got %d", len(manifest.Network.Expose))
	}

	// Test port 1: 3000/tcp with auto-assigned host port
	if manifest.Network.Expose[0].Port != 3000 {
		t.Errorf("expected port 3000, got %d", manifest.Network.Expose[0].Port)
	}
	if manifest.Network.Expose[0].Protocol != "tcp" {
		t.Errorf("expected protocol tcp, got %s", manifest.Network.Expose[0].Protocol)
	}
	if manifest.Network.Expose[0].HostPort != 0 {
		t.Errorf("expected host_port 0 (auto-assign), got %d", manifest.Network.Expose[0].HostPort)
	}

	// Test port 2: 8080/tcp with explicit host port 9000
	if manifest.Network.Expose[1].Port != 8080 {
		t.Errorf("expected port 8080, got %d", manifest.Network.Expose[1].Port)
	}
	if manifest.Network.Expose[1].Protocol != "tcp" {
		t.Errorf("expected protocol tcp, got %s", manifest.Network.Expose[1].Protocol)
	}
	if manifest.Network.Expose[1].HostPort != 9000 {
		t.Errorf("expected host_port 9000, got %d", manifest.Network.Expose[1].HostPort)
	}

	// Test port 3: 5432/udp with auto-assigned host port
	if manifest.Network.Expose[2].Port != 5432 {
		t.Errorf("expected port 5432, got %d", manifest.Network.Expose[2].Port)
	}
	if manifest.Network.Expose[2].Protocol != "udp" {
		t.Errorf("expected protocol udp, got %s", manifest.Network.Expose[2].Protocol)
	}
}

// TestPortExposeNormalize tests that Normalize() applies correct defaults
func TestPortExposeNormalize(t *testing.T) {
	manifestJSON := `{
		"schema_version": "v1",
		"name": "test",
		"version": "1.0.0",
		"runtime": "fc",
		"network": {
			"mode": "bridged",
			"expose": [
				{"port": 3000}
			]
		}
	}`

	var manifest Manifest
	if err := json.Unmarshal([]byte(manifestJSON), &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	manifest.Normalize()

	// Verify default protocol is applied
	if len(manifest.Network.Expose) != 1 {
		t.Fatalf("expected 1 exposed port, got %d", len(manifest.Network.Expose))
	}

	if manifest.Network.Expose[0].Protocol != "tcp" {
		t.Errorf("expected default protocol 'tcp', got '%s'", manifest.Network.Expose[0].Protocol)
	}
}

// TestPortExposeValidation tests that invalid port configs are rejected
func TestPortExposeValidation(t *testing.T) {
	tests := []struct {
		name        string
		manifestJSON string
		expectError bool
	}{
		{
			name: "valid port",
			manifestJSON: `{
				"schema_version": "v1",
				"name": "test",
				"version": "1.0.0",
				"runtime": "fc",
				"resources": {
					"cpu_cores": 1,
					"memory_mb": 128
				},
				"workload": {
					"type": "exec",
					"entrypoint": ["/app/server"]
				},
				"initramfs": {
					"url": "https://example.com/initramfs.cpio.gz"
				},
				"network": {
					"mode": "bridged",
					"expose": [{"port": 3000, "protocol": "tcp"}]
				}
			}`,
			expectError: false,
		},
		{
			name: "invalid protocol",
			manifestJSON: `{
				"schema_version": "v1",
				"name": "test",
				"version": "1.0.0",
				"runtime": "fc",
				"network": {
					"mode": "bridged",
					"expose": [{"port": 3000, "protocol": "http"}]
				}
			}`,
			expectError: true,
		},
		{
			name: "port out of range high",
			manifestJSON: `{
				"schema_version": "v1",
				"name": "test",
				"version": "1.0.0",
				"runtime": "fc",
				"network": {
					"mode": "bridged",
					"expose": [{"port": 70000, "protocol": "tcp"}]
				}
			}`,
			expectError: true,
		},
		{
			name: "port out of range low",
			manifestJSON: `{
				"schema_version": "v1",
				"name": "test",
				"version": "1.0.0",
				"runtime": "fc",
				"network": {
					"mode": "bridged",
					"expose": [{"port": 0, "protocol": "tcp"}]
				}
			}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var manifest Manifest
			if err := json.Unmarshal([]byte(tt.manifestJSON), &manifest); err != nil {
				t.Fatalf("failed to unmarshal manifest: %v", err)
			}

			manifest.Normalize()
			err := manifest.Validate()

			if tt.expectError && err == nil {
				t.Errorf("expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no validation error, got: %v", err)
			}
		})
	}
}
