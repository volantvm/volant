# Phase 2: Native Builder Integration - CORRECT IMPLEMENTATION PLAN

**Status:** Planning (Phase 2 v1 reverted - it was fundamentally wrong)
**Date:** November 11, 2025

---

## What Went Wrong in Phase 2 v1

The initial Phase 2 implementation was architecturally incorrect:

### The Mistake
- Created `pkg/builder/container.go` that was just a Docker/Podman CLI wrapper
- Required Docker daemon to be installed
- Used `docker build` and `docker export` commands
- No actual build capability integrated into volant
- Defeated the entire purpose of being self-contained

### Why This Was Wrong
- **User still needs Docker installed** - worse than using fledge externally!
- **No standalone building** - just shells out to Docker CLI
- **Not truly integrated** - volant has no building capability of its own
- **Misunderstood the directive** - user wanted embedded BuildKit, not Docker wrapper

---

## What We Actually Need (Learned from Fledge)

### Fledge's Correct Approach

After studying fledge's implementation (`internal/buildkit/embedded/embedded_linux.go`), here's how it ACTUALLY works:

```
Dockerfile
    ↓
[Embedded BuildKit] - github.com/moby/buildkit
    ├── No Docker daemon needed!
    ├── BuildKit runs as Go library
    ├── Uses bufconn (in-memory) gRPC
    └── Exports to OCI tar format
    ↓
OCI Tar (image.tar)
    ↓
[skopeo] - converts OCI tar to OCI layout
    ↓
OCI Layout Directory
    ↓
[umoci] - unpacks OCI layers to rootfs
    ↓
Rootfs Directory (unpacked filesystem)
    ↓
[mksquashfs] - compresses to squashfs
    ↓
Final Image (image.squashfs)
```

### Key Components

1. **Embedded BuildKit**
   - Import: `github.com/moby/buildkit`
   - Runs entirely in-process as a Go library
   - No external daemon needed
   - Uses in-memory gRPC (bufconn)
   - Exports to OCI format

2. **skopeo** (external binary)
   - Converts OCI tar → OCI layout
   - Command: `skopeo copy oci-archive:image.tar oci:layout-dir:latest`
   - Why: BuildKit outputs OCI tar, umoci expects OCI layout

3. **umoci** (external binary)
   - Unpacks OCI layout → rootfs directory
   - Command: `umoci unpack --image layout-dir:latest dest-dir`
   - Why: Properly handles layer extraction, metadata, permissions

4. **mksquashfs** (external binary)
   - Compresses rootfs → squashfs
   - Command: `mksquashfs rootfs-dir output.squashfs -comp gzip`
   - Why: Creates bootable, compressed, read-only filesystem

---

## Correct Implementation Plan

### Dependencies Required

**At Build Time:**
- **No Docker daemon** - BuildKit is embedded
- `skopeo` - OCI image manipulation
- `umoci` - OCI layer unpacking
- `mksquashfs` - Squashfs creation

**Runtime (for VMs):**
- Cloud Hypervisor
- Linux kernel with KVM
- Bridge networking tools

### Architecture

```
pkg/builder/
├── builder.go           # Interfaces and types
├── buildkit.go          # Embedded BuildKit integration
├── oci.go               # skopeo wrapper for OCI operations
├── unpack.go            # umoci wrapper for layer unpacking
├── squashfs.go          # mksquashfs wrapper
├── manifest.go          # Volant manifest generation
├── progress.go          # Progress reporting
└── pipeline.go          # Orchestrates full build pipeline
```

### Implementation Steps

#### 1. BuildKit Integration (`pkg/builder/buildkit.go`)

```go
package builder

import (
    "context"
    "path/filepath"

    bkclient "github.com/moby/buildkit/client"
    "github.com/moby/buildkit/control"
    "github.com/moby/buildkit/worker"
    // ... more buildkit imports
)

type BuildkitOptions struct {
    Dockerfile string
    Context    string
    Target     string
    BuildArgs  map[string]string
    OCITarPath string  // Where to export OCI tar
}

// BuildWithEmbeddedBuildkit runs BuildKit as an embedded library
// NO Docker daemon needed!
func BuildWithEmbeddedBuildkit(ctx context.Context, opts BuildkitOptions) error {
    // 1. Create embedded BuildKit controller
    // 2. Setup worker (like fledge's microvmworker)
    // 3. Configure cache storage
    // 4. Setup session manager
    // 5. Configure dockerfile.v0 frontend
    // 6. Solve with OCI exporter
    // 7. Export to OCI tar format
}
```

#### 2. OCI Operations (`pkg/builder/oci.go`)

```go
package builder

// ConvertOCITarToLayout uses skopeo to convert OCI tar to layout
func ConvertOCITarToLayout(ctx context.Context, tarPath, layoutDir string) error {
    // skopeo copy oci-archive:tarPath oci:layoutDir:latest
}
```

#### 3. Layer Unpacking (`pkg/builder/unpack.go`)

```go
package builder

// UnpackOCILayout uses umoci to unpack layers to rootfs
func UnpackOCILayout(ctx context.Context, layoutDir, rootfsDir string) error {
    // umoci unpack --image layoutDir:latest rootfsDir
}
```

#### 4. Squashfs Creation (`pkg/builder/squashfs.go`)

```go
package builder

// CreateSquashfs compresses rootfs to squashfs format
func CreateSquashfs(ctx context.Context, rootfsDir, outputPath string) error {
    // mksquashfs rootfsDir outputPath -comp gzip -no-progress
    // Calculate SHA256 checksum
}
```

#### 5. Full Pipeline (`pkg/builder/pipeline.go`)

```go
package builder

type BuildOptions struct {
    Dockerfile string
    Context    string
    Target     string
    BuildArgs  map[string]string
    OutputPath string
    ImageName  string
    ImageVersion string
}

// Build executes the full pipeline
func Build(ctx context.Context, opts BuildOptions) (*BuildResult, error) {
    // 1. Create temp directories
    // 2. BuildWithEmbeddedBuildkit() → OCI tar
    // 3. ConvertOCITarToLayout() → OCI layout
    // 4. UnpackOCILayout() → rootfs directory
    // 5. CreateSquashfs() → final squashfs
    // 6. Generate manifest
    // 7. Cleanup temp files
    // 8. Return result with paths and checksums
}
```

#### 6. Integration with cmd/volant/build.go

Update `buildFromVolantfile()` and remove `delegateToFledge()`:

```go
func buildFromVolantfile(ctx context.Context, volantfilePath string, opts buildOptions) error {
    vf, err := config.LoadVolantfile(volantfilePath)
    if err != nil {
        return err
    }

    // Use native builder
    result, err := builder.Build(ctx, builder.BuildOptions{
        Dockerfile: vf.Build.Dockerfile,
        Context: vf.Build.Context,
        Target: vf.Build.Target,
        BuildArgs: vf.Build.Args,
        OutputPath: opts.Output,
        ImageName: vf.Image.Name,
        ImageVersion: vf.Image.Version,
    })
    if err != nil {
        return err
    }

    // Auto-install using Phase 1 pipeline
    return autoInstallImage(apiURL, result.SquashfsPath, opts.Tag)
}
```

---

## Dependencies Management

### Required External Binaries

1. **skopeo**
   - Ubuntu/Debian: `apt install skopeo`
   - Fedora/RHEL: `dnf install skopeo`
   - Arch: `pacman -S skopeo`

2. **umoci**
   - Ubuntu/Debian: `apt install umoci`
   - Fedora/RHEL: Available from copr
   - Or build from source: github.com/opencontainers/umoci

3. **mksquashfs**
   - Ubuntu/Debian: `apt install squashfs-tools`
   - Fedora/RHEL: `dnf install squashfs-tools`
   - Arch: `pacman -S squashfs-tools`

### BuildKit Go Dependencies

Add to `go.mod`:
```
github.com/moby/buildkit v0.12.5
github.com/containerd/containerd v1.7.11
go.etcd.io/bbolt v1.3.8
google.golang.org/grpc v1.60.0
```

---

## Benefits of This Approach

### ✅ Self-Contained
- No Docker daemon required
- BuildKit embedded as Go library
- Complete control over build process

### ✅ Efficient
- BuildKit is highly optimized
- Parallel layer building
- Proper caching
- Faster than Docker CLI

### ✅ Proper Layer Handling
- skopeo + umoci handle layers correctly
- Preserves permissions, ownership
- Handles whiteout files properly
- OCI spec compliant

### ✅ Unified Workflow
```bash
volant build -t myapp:v1 .
# → Builds, converts, compresses, auto-registers
# No separate install step
# No external tools
```

---

## Performance Comparison

### Old (Wrong) Approach
```
user runs: volant build
    ↓
volant shells out: docker build
    ↓
Docker daemon builds
    ↓
volant shells out: docker export
    ↓
volant runs: mksquashfs
```
**Problems:**
- Requires Docker daemon
- Extra IPC overhead
- Can't control caching
- Dependent on external tool

### New (Correct) Approach
```
user runs: volant build
    ↓
Embedded BuildKit builds (in-process)
    ↓
skopeo converts to OCI layout
    ↓
umoci unpacks layers
    ↓
mksquashfs compresses
```
**Benefits:**
- No Docker needed!
- In-process BuildKit
- Full control
- Truly integrated

---

## Testing Strategy

### Unit Tests
- Test each component in isolation
- Mock BuildKit interfaces
- Test error handling

### Integration Tests
- Full pipeline tests
- Real Dockerfiles
- Verify output squashfs
- Check manifest generation

### End-to-End Tests
```bash
# Build
volant build -t test:v1 ./test-dockerfile

# Verify registration
volant images list | grep test:v1

# Run VM
volant vms create test-vm --image test:v1

# Verify VM boots
volant vms show test-vm
```

---

## Migration Path

### Phase 2.1: Core Implementation (This PR)
- [ ] Implement embedded BuildKit
- [ ] Add skopeo integration
- [ ] Add umoci integration
- [ ] Create pipeline orchestration
- [ ] Update cmd/volant/build.go

### Phase 2.2: Polish & Documentation
- [ ] Add progress reporting
- [ ] Improve error messages
- [ ] Write user documentation
- [ ] Update architecture docs

### Phase 2.3: Advanced Features
- [ ] Build caching configuration
- [ ] Multi-platform builds
- [ ] BuildKit secrets support
- [ ] Remote cache support

---

## Success Criteria

- [ ] `volant build` works without Docker installed
- [ ] Builds produce working squashfs images
- [ ] Images auto-register with daemon
- [ ] VMs boot successfully from built images
- [ ] Build caching works properly
- [ ] Performance meets or exceeds fledge

---

## References

- Fledge BuildKit Integration: `fledge/internal/buildkit/embedded/embedded_linux.go`
- BuildKit Client: `github.com/moby/buildkit/client`
- OCI Image Spec: https://github.com/opencontainers/image-spec
- Skopeo: https://github.com/containers/skopeo
- Umoci: https://github.com/opencontainers/umoci

---

**Next Steps:**

1. Create `pkg/builder/` with correct embedded BuildKit
2. Add skopeo and umoci wrappers
3. Implement full pipeline
4. Test thoroughly
5. Update documentation

This is the **correct** approach that truly integrates building into volant.
