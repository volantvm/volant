// Copyright (c) 2025 HYPR. PTE. LTD.
//
// Business Source License 1.1
// See LICENSE file in the project root for details.

package images

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/volantvm/volant/internal/imagespec"
)

// Loader handles manifest discovery and parsing.
type Loader interface {
	Load(path string) (imagespec.Manifest, error)
}

// FileLoader loads image manifests from disk.
type FileLoader struct{}

// Load reads and parses a manifest file.
func (FileLoader) Load(path string) (imagespec.Manifest, error) {
	if strings.TrimSpace(path) == "" {
		return imagespec.Manifest{}, fmt.Errorf("image loader: path required")
	}
	file, err := os.Open(path)
	if err != nil {
		return imagespec.Manifest{}, fmt.Errorf("image loader: open %s: %w", path, err)
	}
	defer file.Close()

	manifest, err := decodeManifest(file)
	if err != nil {
		return imagespec.Manifest{}, fmt.Errorf("image loader: decode %s: %w", path, err)
	}
	return manifest, nil
}

func decodeManifest(reader io.Reader) (imagespec.Manifest, error) {
	var manifest imagespec.Manifest
	if err := json.NewDecoder(reader).Decode(&manifest); err != nil {
		return imagespec.Manifest{}, err
	}
	if err := manifest.Validate(); err != nil {
		return imagespec.Manifest{}, err
	}
	return manifest, nil
}
