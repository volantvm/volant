package fsdetect

import "testing"

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		defaultFormat Format
		want          Format
	}{
		// Extension-based detection
		{
			name:          "squashfs extension",
			path:          "/path/to/rootfs.squashfs",
			defaultFormat: FormatExt4,
			want:          FormatSquashFS,
		},
		{
			name:          "qcow2 extension",
			path:          "/path/to/disk.qcow2",
			defaultFormat: FormatExt4,
			want:          FormatQCOW2,
		},
		{
			name:          "xfs extension",
			path:          "/path/to/disk.xfs",
			defaultFormat: FormatExt4,
			want:          FormatXFS,
		},
		{
			name:          "btrfs extension",
			path:          "/path/to/disk.btrfs",
			defaultFormat: FormatExt4,
			want:          FormatBtrfs,
		},
		{
			name:          "ext4 extension explicit",
			path:          "/path/to/disk.ext4",
			defaultFormat: FormatSquashFS,
			want:          FormatExt4,
		},
		{
			name:          "img extension (maps to ext4)",
			path:          "/path/to/disk.img",
			defaultFormat: FormatSquashFS,
			want:          FormatExt4,
		},

		// Case insensitivity
		{
			name:          "uppercase squashfs",
			path:          "/path/to/ROOTFS.SQUASHFS",
			defaultFormat: FormatExt4,
			want:          FormatSquashFS,
		},
		{
			name:          "mixed case",
			path:          "/path/to/Disk.QCoW2",
			defaultFormat: FormatExt4,
			want:          FormatQCOW2,
		},

		// URL paths
		{
			name:          "http url with squashfs",
			path:          "http://example.com/images/rootfs.squashfs",
			defaultFormat: FormatExt4,
			want:          FormatSquashFS,
		},
		{
			name:          "https url with img",
			path:          "https://cdn.example.com/disk.img",
			defaultFormat: FormatSquashFS,
			want:          FormatExt4,
		},

		// Default fallbacks
		{
			name:          "no extension uses default",
			path:          "/path/to/rootfs",
			defaultFormat: FormatSquashFS,
			want:          FormatSquashFS,
		},
		{
			name:          "empty path uses default",
			path:          "",
			defaultFormat: FormatSquashFS,
			want:          FormatSquashFS,
		},
		{
			name:          "unknown extension uses default",
			path:          "/path/to/disk.unknown",
			defaultFormat: FormatSquashFS,
			want:          FormatSquashFS,
		},

		// Whitespace handling
		{
			name:          "path with trailing whitespace",
			path:          "/path/to/disk.squashfs  ",
			defaultFormat: FormatExt4,
			want:          FormatSquashFS,
		},
		{
			name:          "path with leading whitespace",
			path:          "  /path/to/disk.xfs",
			defaultFormat: FormatExt4,
			want:          FormatXFS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectFormat(tt.path, tt.defaultFormat)
			if got != tt.want {
				t.Errorf("DetectFormat(%q, %q) = %q, want %q", tt.path, tt.defaultFormat, got, tt.want)
			}
		})
	}
}

func TestIsReadOnly(t *testing.T) {
	tests := []struct {
		format Format
		want   bool
	}{
		{FormatSquashFS, true},
		{FormatExt4, false},
		{FormatXFS, false},
		{FormatBtrfs, false},
		{FormatQCOW2, false},
		{FormatRaw, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			got := IsReadOnly(tt.format)
			if got != tt.want {
				t.Errorf("IsReadOnly(%q) = %v, want %v", tt.format, got, tt.want)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		input string
		want  Format
	}{
		{"squashfs", FormatSquashFS},
		{"SQUASHFS", FormatSquashFS},
		{"SquashFS", FormatSquashFS},
		{"  squashfs  ", FormatSquashFS},
		{"ext4", FormatExt4},
		{"xfs", FormatXFS},
		{"btrfs", FormatBtrfs},
		{"qcow2", FormatQCOW2},
		{"raw", FormatRaw},
		{"unknown", Format("unknown")},
		{"", Format("")},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Normalize(tt.input)
			if got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
