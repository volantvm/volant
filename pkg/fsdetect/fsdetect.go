package fsdetect

import (
	"strings"
)

// Format represents a filesystem format type
type Format string

const (
	// Supported filesystem formats
	FormatSquashFS Format = "squashfs" // Compressed read-only (default)
	FormatExt4     Format = "ext4"     // Standard Linux filesystem
	FormatXFS      Format = "xfs"      // High-performance filesystem
	FormatBtrfs    Format = "btrfs"    // Copy-on-write filesystem
	FormatQCOW2    Format = "qcow2"    // QEMU disk image format
	FormatRaw      Format = "raw"      // Raw disk image
)

// DetectFormat detects the filesystem format from a file path or URL.
// It performs case-insensitive extension matching and returns the detected format.
// If no format is detected, it returns the defaultFormat.
func DetectFormat(path string, defaultFormat Format) Format {
	if path == "" {
		return defaultFormat
	}

	// Normalize to lowercase for case-insensitive matching
	lowerPath := strings.ToLower(strings.TrimSpace(path))

	// Detect from file extension
	switch {
	case strings.HasSuffix(lowerPath, ".squashfs"):
		return FormatSquashFS
	case strings.HasSuffix(lowerPath, ".qcow2"):
		return FormatQCOW2
	case strings.HasSuffix(lowerPath, ".xfs"):
		return FormatXFS
	case strings.HasSuffix(lowerPath, ".btrfs"):
		return FormatBtrfs
	case strings.HasSuffix(lowerPath, ".ext4"):
		return FormatExt4
	case strings.HasSuffix(lowerPath, ".img"):
		// .img files are typically ext4
		return FormatExt4
	default:
		return defaultFormat
	}
}

// IsReadOnly returns true if the filesystem format is read-only.
// Currently only squashfs is read-only by design.
func IsReadOnly(format Format) bool {
	return format == FormatSquashFS
}

// Normalize converts a format string to a canonical Format value.
// It performs case-insensitive matching and handles common aliases.
func Normalize(formatStr string) Format {
	normalized := strings.ToLower(strings.TrimSpace(formatStr))

	switch normalized {
	case "squashfs":
		return FormatSquashFS
	case "ext4":
		return FormatExt4
	case "xfs":
		return FormatXFS
	case "btrfs":
		return FormatBtrfs
	case "qcow2":
		return FormatQCOW2
	case "raw":
		return FormatRaw
	default:
		// Return as-is for unknown formats
		return Format(normalized)
	}
}

// String returns the string representation of a Format
func (f Format) String() string {
	return string(f)
}
