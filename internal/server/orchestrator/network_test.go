// Copyright (c) 2025 HYPR. PTE. LTD.
//
// Business Source License 1.1
// See LICENSE file in the project root for details.

package orchestrator

import (
	"testing"

	"github.com/volantvm/volant/internal/imagespec"
	"github.com/volantvm/volant/internal/server/orchestrator/vmconfig"
)

func TestResolveNetworkConfig(t *testing.T) {
	tests := []struct {
		name     string
		manifest *imagespec.Manifest
		vmConfig *vmconfig.Config
		want     *imagespec.NetworkConfig
	}{
		{
			name:     "both nil returns nil",
			manifest: nil,
			vmConfig: nil,
			want:     nil,
		},
		{
			name: "manifest only",
			manifest: &imagespec.Manifest{
				Network: &imagespec.NetworkConfig{
					Mode: imagespec.NetworkModeVsock,
				},
			},
			vmConfig: nil,
			want: &imagespec.NetworkConfig{
				Mode: imagespec.NetworkModeVsock,
			},
		},
		{
			name:     "vm config only",
			manifest: nil,
			vmConfig: &vmconfig.Config{
				Network: &imagespec.NetworkConfig{
					Mode: imagespec.NetworkModeBridged,
				},
			},
			want: &imagespec.NetworkConfig{
				Mode: imagespec.NetworkModeBridged,
			},
		},
		{
			name: "vm config overrides manifest",
			manifest: &imagespec.Manifest{
				Network: &imagespec.NetworkConfig{
					Mode: imagespec.NetworkModeVsock,
				},
			},
			vmConfig: &vmconfig.Config{
				Network: &imagespec.NetworkConfig{
					Mode: imagespec.NetworkModeDHCP,
				},
			},
			want: &imagespec.NetworkConfig{
				Mode: imagespec.NetworkModeDHCP,
			},
		},
		{
			name: "vm config without network falls back to manifest",
			manifest: &imagespec.Manifest{
				Network: &imagespec.NetworkConfig{
					Mode: imagespec.NetworkModeVsock,
				},
			},
			vmConfig: &vmconfig.Config{
				Image: "test",
			},
			want: &imagespec.NetworkConfig{
				Mode: imagespec.NetworkModeVsock,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveNetworkConfig(tt.manifest, tt.vmConfig)
			if tt.want == nil && got != nil {
				t.Errorf("resolveNetworkConfig() = %v, want nil", got)
				return
			}
			if tt.want != nil && got == nil {
				t.Errorf("resolveNetworkConfig() = nil, want %v", tt.want)
				return
			}
			if tt.want != nil && got != nil && got.Mode != tt.want.Mode {
				t.Errorf("resolveNetworkConfig() mode = %v, want %v", got.Mode, tt.want.Mode)
			}
		})
	}
}

func TestNeedsIPAllocation(t *testing.T) {
	tests := []struct {
		name   string
		netCfg *imagespec.NetworkConfig
		want   bool
	}{
		{
			name:   "nil config needs IP (default)",
			netCfg: nil,
			want:   true,
		},
		{
			name: "vsock mode does not need IP",
			netCfg: &imagespec.NetworkConfig{
				Mode: imagespec.NetworkModeVsock,
			},
			want: false,
		},
		{
			name: "bridged mode needs IP",
			netCfg: &imagespec.NetworkConfig{
				Mode: imagespec.NetworkModeBridged,
			},
			want: true,
		},
		{
			name: "dhcp mode does not need host-managed IP",
			netCfg: &imagespec.NetworkConfig{
				Mode: imagespec.NetworkModeDHCP,
			},
			want: false,
		},
		{
			name: "empty mode defaults to needing IP",
			netCfg: &imagespec.NetworkConfig{
				Mode: "",
			},
			want: true,
		},
		{
			name: "uppercase vsock mode",
			netCfg: &imagespec.NetworkConfig{
				Mode: "VSOCK",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needsIPAllocation(tt.netCfg); got != tt.want {
				t.Errorf("needsIPAllocation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNeedsTapDevice(t *testing.T) {
	tests := []struct {
		name   string
		netCfg *imagespec.NetworkConfig
		want   bool
	}{
		{
			name:   "nil config needs tap (default)",
			netCfg: nil,
			want:   true,
		},
		{
			name: "vsock mode does not need tap",
			netCfg: &imagespec.NetworkConfig{
				Mode: imagespec.NetworkModeVsock,
			},
			want: false,
		},
		{
			name: "bridged mode needs tap",
			netCfg: &imagespec.NetworkConfig{
				Mode: imagespec.NetworkModeBridged,
			},
			want: true,
		},
		{
			name: "dhcp mode needs tap",
			netCfg: &imagespec.NetworkConfig{
				Mode: imagespec.NetworkModeDHCP,
			},
			want: true,
		},
		{
			name: "empty mode defaults to needing tap",
			netCfg: &imagespec.NetworkConfig{
				Mode: "",
			},
			want: true,
		},
		{
			name: "uppercase vsock mode",
			netCfg: &imagespec.NetworkConfig{
				Mode: "VSOCK",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needsTapDevice(tt.netCfg); got != tt.want {
				t.Errorf("needsTapDevice() = %v, want %v", got, tt.want)
			}
		})
	}
}
