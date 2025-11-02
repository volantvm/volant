// Copyright (c) 2025 HYPR. PTE. LTD.
//
// Business Source License 1.1
// See LICENSE file in the project root for details.

package images

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/volantvm/volant/internal/imagespec"
	"github.com/volantvm/volant/internal/server/db"
)

// Registry manages image manifests at runtime.
type Registry struct {
	mu        sync.RWMutex
	backend   db.ImageRepository
	manifests map[string]imagespec.Manifest
}

func NewRegistry(repo db.ImageRepository) *Registry {
	return &Registry{backend: repo, manifests: make(map[string]imagespec.Manifest)}
}

func (r *Registry) Register(manifest imagespec.Manifest) {
	manifest.Normalize()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.manifests[manifest.Name] = manifest
}

func (r *Registry) Get(name string) (imagespec.Manifest, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	manifest, ok := r.manifests[name]
	return manifest, ok
}

func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.manifests))
	for name := range r.manifests {
		names = append(names, name)
	}
	return names
}

func (r *Registry) ResolveAction(image, action string) (imagespec.Manifest, imagespec.Action, error) {
	manifest, ok := r.Get(image)
	if !ok {
		return imagespec.Manifest{}, imagespec.Action{}, fmt.Errorf("image %s not found", image)
	}
	actionSpec, ok := manifest.Actions[action]
	if !ok {
		return imagespec.Manifest{}, imagespec.Action{}, fmt.Errorf("action %s not found", action)
	}
	return manifest, actionSpec, nil
}

func (r *Registry) Fetch(ctx context.Context, name string) (imagespec.Manifest, error) {
	if r.backend == nil {
		return imagespec.Manifest{}, errors.New("registry backend not configured")
	}
	image, err := r.backend.GetByName(ctx, name)
	if err != nil {
		return imagespec.Manifest{}, err
	}
	if image == nil {
		return imagespec.Manifest{}, fmt.Errorf("image %s not found", name)
	}
	var manifest imagespec.Manifest
	if len(image.Metadata) > 0 {
		if err := json.Unmarshal(image.Metadata, &manifest); err != nil {
			return imagespec.Manifest{}, err
		}
	}
	manifest.Name = image.Name
	manifest.Version = image.Version
	manifest.Enabled = image.Enabled
	manifest.Normalize()
	return manifest, nil
}

func (r *Registry) Persist(ctx context.Context, manifest imagespec.Manifest, enabled bool) error {
	if r.backend == nil {
		return errors.New("registry backend not configured")
	}
	manifest.Normalize()
	if err := manifest.Validate(); err != nil {
		return err
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	return r.backend.Upsert(ctx, db.Image{
		Name:     manifest.Name,
		Version:  manifest.Version,
		Enabled:  enabled,
		Metadata: data,
	})
}

func (r *Registry) Remove(ctx context.Context, name string) error {
	if r.backend == nil {
		return errors.New("registry backend not configured")
	}
	if err := r.backend.Delete(ctx, name); err != nil {
		return err
	}
	r.mu.Lock()
	delete(r.manifests, name)
	r.mu.Unlock()
	return nil
}

func cloneLabelMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
