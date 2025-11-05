package vmconfig

import (
	"testing"

	"github.com/volantvm/volant/internal/imagespec"
)

// TestFromManifestWithOverrides_ExposePortsApplied tests that expose ports
// from the manifest are correctly applied to the VM config
func TestFromManifestWithOverrides_ExposePortsApplied(t *testing.T) {
	manifest := &imagespec.Manifest{
		SchemaVersion: "v1",
		Name:          "test",
		Version:       "1.0.0",
		Runtime:       "fc",
		Resources: imagespec.ResourceSpec{
			CPUCores: 1,
			MemoryMB: 128,
		},
		Network: &imagespec.NetworkConfig{
			Mode: imagespec.NetworkModeBridged,
			Expose: []imagespec.PortExpose{
				{Port: 3000, Protocol: "tcp", HostPort: 0},       // Auto-assign
				{Port: 8080, Protocol: "tcp", HostPort: 9000},    // Explicit host port
				{Port: 5432, Protocol: "udp", HostPort: 0},       // Auto-assign UDP
			},
		},
		Workload: imagespec.Workload{
			Type:       "exec",
			Entrypoint: []string{"/app/server"},
		},
		Initramfs: imagespec.Initramfs{
			URL: "https://example.com/initramfs.cpio.gz",
		},
	}

	overrides := Overrides{}

	cfg, err := FromManifestWithOverrides(manifest, overrides, "test-image", "fc")
	if err != nil {
		t.Fatalf("FromManifestWithOverrides failed: %v", err)
	}

	// Verify expose ports were copied from manifest
	if len(cfg.Expose) != 3 {
		t.Errorf("expected 3 expose ports, got %d", len(cfg.Expose))
	}

	// Verify port 1: 3000/tcp (auto-assign)
	if cfg.Expose[0].Port != 3000 {
		t.Errorf("expected port 3000, got %d", cfg.Expose[0].Port)
	}
	if cfg.Expose[0].Protocol != "tcp" {
		t.Errorf("expected protocol tcp, got %s", cfg.Expose[0].Protocol)
	}
	if cfg.Expose[0].HostPort != 0 {
		t.Errorf("expected host_port 0 (auto-assign), got %d", cfg.Expose[0].HostPort)
	}

	// Verify port 2: 8080/tcp with explicit host port 9000
	if cfg.Expose[1].Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Expose[1].Port)
	}
	if cfg.Expose[1].Protocol != "tcp" {
		t.Errorf("expected protocol tcp, got %s", cfg.Expose[1].Protocol)
	}
	if cfg.Expose[1].HostPort != 9000 {
		t.Errorf("expected host_port 9000, got %d", cfg.Expose[1].HostPort)
	}

	// Verify port 3: 5432/udp (auto-assign)
	if cfg.Expose[2].Port != 5432 {
		t.Errorf("expected port 5432, got %d", cfg.Expose[2].Port)
	}
	if cfg.Expose[2].Protocol != "udp" {
		t.Errorf("expected protocol udp, got %s", cfg.Expose[2].Protocol)
	}
}

// TestFromManifestWithOverrides_ExposePortsOverride tests that override ports
// take precedence over manifest ports
func TestFromManifestWithOverrides_ExposePortsOverride(t *testing.T) {
	manifest := &imagespec.Manifest{
		SchemaVersion: "v1",
		Name:          "test",
		Version:       "1.0.0",
		Runtime:       "fc",
		Resources: imagespec.ResourceSpec{
			CPUCores: 1,
			MemoryMB: 128,
		},
		Network: &imagespec.NetworkConfig{
			Mode: imagespec.NetworkModeBridged,
			Expose: []imagespec.PortExpose{
				{Port: 3000, Protocol: "tcp"},
			},
		},
		Workload: imagespec.Workload{
			Type:       "exec",
			Entrypoint: []string{"/app/server"},
		},
		Initramfs: imagespec.Initramfs{
			URL: "https://example.com/initramfs.cpio.gz",
		},
	}

	// Override with different ports
	overrides := Overrides{
		ExposePorts: []Expose{
			{Port: 8080, Protocol: "tcp", HostPort: 9999},
		},
	}

	cfg, err := FromManifestWithOverrides(manifest, overrides, "test-image", "fc")
	if err != nil {
		t.Fatalf("FromManifestWithOverrides failed: %v", err)
	}

	// Verify override ports replaced manifest ports
	if len(cfg.Expose) != 1 {
		t.Errorf("expected 1 expose port from override, got %d", len(cfg.Expose))
	}

	if cfg.Expose[0].Port != 8080 {
		t.Errorf("expected port 8080 from override, got %d", cfg.Expose[0].Port)
	}
	if cfg.Expose[0].HostPort != 9999 {
		t.Errorf("expected host_port 9999 from override, got %d", cfg.Expose[0].HostPort)
	}
}

// TestFromManifestWithOverrides_NoExposeFlag tests that empty override disables all ports
func TestFromManifestWithOverrides_NoExposeFlag(t *testing.T) {
	manifest := &imagespec.Manifest{
		SchemaVersion: "v1",
		Name:          "test",
		Version:       "1.0.0",
		Runtime:       "fc",
		Resources: imagespec.ResourceSpec{
			CPUCores: 1,
			MemoryMB: 128,
		},
		Network: &imagespec.NetworkConfig{
			Mode: imagespec.NetworkModeBridged,
			Expose: []imagespec.PortExpose{
				{Port: 3000, Protocol: "tcp"},
			},
		},
		Workload: imagespec.Workload{
			Type:       "exec",
			Entrypoint: []string{"/app/server"},
		},
		Initramfs: imagespec.Initramfs{
			URL: "https://example.com/initramfs.cpio.gz",
		},
	}

	// Override with empty slice (simulates --no-expose flag)
	overrides := Overrides{
		ExposePorts: []Expose{},
	}

	cfg, err := FromManifestWithOverrides(manifest, overrides, "test-image", "fc")
	if err != nil {
		t.Fatalf("FromManifestWithOverrides failed: %v", err)
	}

	// Verify all ports are disabled
	if len(cfg.Expose) != 0 {
		t.Errorf("expected 0 expose ports (--no-expose), got %d", len(cfg.Expose))
	}
}
