# 📜 THE VOLANT MANIFESTO
## Ultimate Source of Truth for AI-Assisted Development

**Version:** 1.0.0
**Created:** 2025-01-07
**Purpose:** Self-contained, comprehensive documentation enabling parallel development and context-free restarts
**Audience:** AI agents, future developers, context-reset scenarios

---

# TABLE OF CONTENTS

1. [EXECUTIVE VISION](#1-executive-vision)
2. [CURRENT STATE ANALYSIS](#2-current-state-analysis)
3. [ARCHITECTURAL DEEP DIVE](#3-architectural-deep-dive)
4. [THE MASTER PLAN](#4-the-master-plan)
5. [PHASE 0: CLEAN BOUNDARIES](#5-phase-0-clean-boundaries)
6. [PHASE 1: PARALLEL TRACK A - ENV + DNS](#6-phase-1-track-a-environment-variables--dns)
7. [PHASE 1: PARALLEL TRACK B - VOLUMES](#7-phase-1-track-b-volume-manager)
8. [PHASE 1: PARALLEL TRACK C - DRIFT L4](#8-phase-1-track-c-drift-l4-completion)
9. [PHASE 1: PARALLEL TRACK F - COMPOSE CONVERTER](#9-phase-1-track-f-docker-compose-converter)
10. [PHASE 2: PARALLEL TRACK D - STACK ORCHESTRATOR](#10-phase-2-track-d-stack-orchestrator)
11. [PHASE 2: PARALLEL TRACK E - WEB UI](#11-phase-2-track-e-web-ui-enhancements)
12. [PHASE 2: PARALLEL TRACK G - FLEDGE SERVER](#12-phase-2-track-g-fledge-server-mode)
13. [PHASE 3: INTEGRATION](#13-phase-3-integration)
14. [API CONTRACTS](#14-api-contracts)
15. [DATABASE EVOLUTION](#15-database-evolution)
16. [TESTING STRATEGY](#16-testing-strategy)
17. [WORKTREE STRATEGY](#17-worktree-strategy)
18. [TROUBLESHOOTING](#18-troubleshooting)

---

# 1. EXECUTIVE VISION

## What We're Building

**Volant + Fledge + Web UI = The Docker Killer**

A microVM orchestration platform that combines:
- **Security of VMs** (hardware isolation, dedicated kernels)
- **Speed of containers** (50-150ms boot times)
- **Ease of Docker** (Dockerfile support, Compose compatibility)
- **Power of Kubernetes** (orchestration, service discovery, scaling)

**Without the complexity** (no etcd, no CNI, no CRI - just SQLite + eBPF)

## The Problem We're Solving

Docker Compose is limited:
- ❌ Shared kernel vulnerabilities
- ❌ Namespace isolation is weak
- ❌ No true hardware isolation
- ❌ GPU passthrough is hacky
- ❌ No sub-second boot times
- ❌ Complex storage with OverlayFS

Kubernetes is complex:
- ❌ Requires cluster setup
- ❌ etcd, CNI, CRI, CSI complexity
- ❌ Pods, Services, Ingress confusion
- ❌ High operational overhead
- ❌ Overkill for single-node deployments

**Volant solves both:**
- ✅ Hardware-isolated microVMs
- ✅ Docker Compose compatibility
- ✅ Sub-second boot (initramfs strategy)
- ✅ Native GPU passthrough (VFIO)
- ✅ Simple SQLite backend
- ✅ eBPF L4 routing (no iptables complexity)
- ✅ Single daemon, zero coordination

## Success Criteria

### MVP (3 weeks):
- ✅ Deploy multi-service stacks via `volar stack deploy -f volant.yaml`
- ✅ Convert `docker-compose.yml` → `volant.yaml`
- ✅ Services discover each other via DNS
- ✅ Volumes persist data across restarts
- ✅ Web UI shows stack topology and health

### Production (6-8 weeks):
- ✅ Rolling updates with zero downtime
- ✅ Secret management
- ✅ Volume backups/restores
- ✅ Stack templates catalog
- ✅ Audit logging
- ✅ Resource quotas per stack

---

# 2. CURRENT STATE ANALYSIS

## Repository Structure

```
/Users/marcxavier/Desktop/work/
├── volant/          # Orchestrator (17,900 lines Go)
├── fledge/          # Artifact builder (3,912 lines Go)
└── web/             # Web UI (Next.js 15 + React 19)
```

## Volant - Current Capabilities ✅

**Version:** v0.6.8
**License:** BSL 1.1 → Apache 2.0 (Oct 4, 2029)

### Core Components:
1. **volantd** - Control plane daemon
   - SQLite persistence
   - HTTP REST API (40+ endpoints)
   - VM lifecycle orchestration
   - Event bus (SSE)
   - Image registry

2. **volar** - CLI tool
   - VM management (create/list/start/stop/destroy)
   - Image management (install/list/remove)
   - Deployment scaling
   - Host setup

3. **kestrel** - Guest agent (PID 1)
   - Manifest decoding from kernel cmdline
   - Workload supervision
   - HTTP API (:8080)
   - Shell spawning

4. **driftd** - L4 switch daemon (PARTIAL)
   - eBPF packet routing
   - Route management
   - Vsock proxy fallback

### Key Strengths:
- ✅ Manifest-driven architecture (base64+gzip in kernel cmdline)
- ✅ Static IP allocation (simple, deterministic)
- ✅ Dual kernel strategy (vmlinux for initramfs, bzImage for rootfs)
- ✅ Squashfs support (70% size reduction)
- ✅ VFIO device passthrough
- ✅ Cloud-init integration
- ✅ Deployment scaling (replicas)

### Critical Gaps:
- ❌ No multi-service orchestration (can only deploy single image)
- ❌ No service discovery (manual IP management)
- ❌ No environment variable support
- ❌ No persistent volumes
- ❌ No Docker Compose compatibility
- ❌ Fledge build/runtime config boundaries are fuzzy

## Fledge - Current Capabilities ✅

**Version:** v0.2.9
**Build Strategies:** OCI Rootfs + Initramfs

### Core Features:
- ✅ Dockerfile support (full BuildKit integration)
- ✅ Multi-stage builds
- ✅ OCI image conversion → squashfs/ext4
- ✅ Initramfs generation (minimal stateless)
- ✅ Agent sourcing (GitHub/local/HTTP)
- ✅ Reproducible builds (normalized timestamps)
- ✅ Embedded BuildKit (no external daemon)

### Configuration:
- **fledge.toml** - Build-time config (strategy, source, filesystem)
- **manifest.toml** - Runtime config template
- **Output:** `<image>.img` + `manifest.json`

### Critical Gaps:
- ❌ Build/runtime boundaries are unclear
- ❌ No batch build API (for stacks)
- ❌ No build caching between runs
- ❌ Server mode is basic (no queue, no status tracking)

## Web UI - Current Capabilities ✅

**Framework:** Next.js 15 + React 19
**Status:** 85% complete

### Implemented:
- ✅ VM management (create/list/start/stop/destroy)
- ✅ Image management (install/list/remove)
- ✅ Deployment orchestration (create/scale/delete)
- ✅ Real-time monitoring (CPU, memory, network)
- ✅ VFIO device manager
- ✅ Cloud-init editor
- ✅ Forge build workspace
- ✅ 42/42 API endpoints integrated
- ✅ Server-Sent Events (SSE) for live updates

### Critical Gaps:
- ❌ No stack management UI
- ❌ No service dependency visualization
- ❌ No volume management UI
- ❌ No Docker Compose converter UI
- ❌ No RBAC/authorization UI

---

# 3. ARCHITECTURAL DEEP DIVE

## 3.1 Manifest System (CRITICAL)

### Current Design:
```go
// volant/internal/imagespec/spec.go
type ImageManifest struct {
    SchemaVersion string
    Name          string
    Version       string
    Runtime       string

    Rootfs        *RootfsConfig      // OCI rootfs path
    Initramfs     *InitramfsConfig   // Initramfs path
    Resources     ResourcesConfig     // CPU, memory
    Workload      WorkloadConfig      // Entrypoint, args
    Network       *NetworkConfig      // Mode, expose ports
    CloudInit     *CloudInitConfig    // Cloud-init seed
    Devices       *DevicesConfig      // PCI passthrough
    Actions       map[string]Action   // Custom API actions
}
```

### Encoding Strategy:
```go
// Manifest → gzip → base64 → kernel cmdline
encoded := base64.StdEncoding.EncodeToString(gzipCompress(jsonBytes))
cmdline += fmt.Sprintf("volant.manifest=%s", encoded)
```

**Why:** Kernel cmdline is limited to ~2KB, gzip+base64 allows complex manifests to fit.

### Boot Flow:
```
1. volantd creates VM → encodes manifest into cmdline
2. Cloud Hypervisor boots VM with cmdline
3. kestrel (PID 1) reads cmdline
4. kestrel decodes manifest (base64 → gunzip → JSON)
5. kestrel executes workload based on manifest
```

## 3.2 Orchestrator Engine

### File: `volant/internal/server/orchestrator/orchestrator.go`

```go
type Engine struct {
    db          db.Store
    launcher    runtime.Launcher  // Cloud Hypervisor
    network     network.Manager   // Bridge/vsock/drift
    deviceMgr   *devicemanager.VFIOManager
    eventBus    eventbus.Bus
    driftClient *driftclient.Client  // Optional L4 routing
}

func (e *Engine) CreateVM(ctx context.Context, req CreateVMRequest) (*VM, error) {
    // 1. Validate image exists
    image := e.db.Images().GetByName(ctx, req.ImageName)

    // 2. Allocate resources
    ip := e.network.AllocateIP(ctx)
    cid := e.allocateVsockCID()
    mac := deriveMAC(req.Name, cid)

    // 3. Merge config (image defaults + overrides)
    config := mergeConfig(image.Manifest, req.Overrides)

    // 4. Encode manifest into cmdline
    manifestEncoded := imagespec.Encode(config)
    cmdline := fmt.Sprintf("volant.manifest=%s volant.runtime=%s", manifestEncoded, config.Runtime)

    // 5. Prepare network (tap device if bridged)
    if config.Network.Mode == "bridged" {
        e.network.PrepareTap(ctx, req.Name, ip, mac)
    }

    // 6. Prepare cloud-init seed (if configured)
    if config.CloudInit != nil {
        e.prepareCloudInitSeed(ctx, req.Name, config.CloudInit)
    }

    // 7. Bind VFIO devices (if configured)
    if len(config.Devices.PCIPassthrough) > 0 {
        e.deviceMgr.Bind(ctx, config.Devices.PCIPassthrough)
    }

    // 8. Launch Cloud Hypervisor
    pid := e.launcher.Launch(ctx, LaunchConfig{
        Name:     req.Name,
        Kernel:   selectKernel(config),
        Rootfs:   config.Rootfs,
        Initramfs: config.Initramfs,
        Cmdline:  cmdline,
        CPUCores: config.Resources.CPUCores,
        MemoryMB: config.Resources.MemoryMB,
        VsockCID: cid,
    })

    // 9. Store VM in database
    vm := &db.VM{
        Name:      req.Name,
        Status:    "running",
        Runtime:   config.Runtime,
        IP:        ip,
        MAC:       mac,
        VsockCID:  cid,
        PID:       pid,
        CPUCores:  config.Resources.CPUCores,
        MemoryMB:  config.Resources.MemoryMB,
    }
    e.db.VMs().Create(ctx, vm)

    // 10. Emit event
    e.eventBus.Publish(ctx, "vm.created", vm)

    // 11. Register routes (if Drift enabled)
    if e.driftClient != nil {
        e.registerDriftRoutes(ctx, vm, config.Network.Expose)
    }

    return vm, nil
}
```

## 3.3 Network Architecture

### Three Modes:

**1. Bridged (Default)**
```
Host: 192.168.127.1 (bridge vbr0)
  │
  ├─ VM1: 192.168.127.10 (tap0) [static IP]
  ├─ VM2: 192.168.127.11 (tap1) [static IP]
  └─ VM3: 192.168.127.12 (tap2) [static IP]
```

**IPAM:** Linear pool allocation from subnet (192.168.127.0/24)

**2. Vsock (High Performance)**
```
Host ←──vsock──→ VM (CID: 5)
No network stack overhead
Ultra-low latency
```

**3. Drift L4 (eBPF Routing)**
```
Client → :8080 (host)
    ↓
[eBPF Dataplane]
    ↓
192.168.127.10:80 (VM)

Faster than vsock proxy (no userspace bounce)
```

## 3.4 Database Schema

### File: `volant/internal/server/db/sqlite/sqlite.go`

```sql
-- VM runtime state
CREATE TABLE vms (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    status TEXT NOT NULL,  -- pending|starting|running|stopped|crashed
    runtime TEXT NOT NULL,
    image TEXT NOT NULL,
    pid INTEGER,
    ip_address TEXT UNIQUE NOT NULL,
    mac_address TEXT UNIQUE NOT NULL,
    vsock_cid INTEGER,
    cpu_cores INTEGER NOT NULL,
    memory_mb INTEGER NOT NULL,
    kernel_cmdline TEXT,
    serial_socket TEXT,
    group_id INTEGER,  -- Foreign key to vm_groups
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

-- VM configuration (current)
CREATE TABLE vm_configs (
    vm_id INTEGER PRIMARY KEY,
    config_json TEXT NOT NULL,
    version INTEGER NOT NULL,
    created_at TIMESTAMP,
    FOREIGN KEY (vm_id) REFERENCES vms(id)
);

-- VM configuration history (audit trail)
CREATE TABLE vm_config_history (
    id INTEGER PRIMARY KEY,
    vm_id INTEGER NOT NULL,
    version INTEGER NOT NULL,
    config_json TEXT NOT NULL,
    created_at TIMESTAMP,
    FOREIGN KEY (vm_id) REFERENCES vms(id)
);

-- Image manifests
CREATE TABLE images (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    version TEXT NOT NULL,
    enabled INTEGER DEFAULT 1,
    manifest_json TEXT NOT NULL,
    installed_at TIMESTAMP,
    updated_at TIMESTAMP
);

-- IP allocation tracking
CREATE TABLE ip_allocations (
    ip_address TEXT PRIMARY KEY,
    vm_id INTEGER,
    status TEXT NOT NULL,  -- available|leased
    leased_at TIMESTAMP,
    FOREIGN KEY (vm_id) REFERENCES vms(id)
);

-- Deployment groups (replicas)
CREATE TABLE vm_groups (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    image TEXT NOT NULL,
    replicas INTEGER NOT NULL,
    config_json TEXT NOT NULL,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

-- Image artifacts (optional storage)
CREATE TABLE image_artifacts (
    id INTEGER PRIMARY KEY,
    image_name TEXT NOT NULL,
    version TEXT,
    artifact_name TEXT,
    kind TEXT,  -- rootfs|initramfs|kernel
    source_url TEXT,
    checksum TEXT,
    local_path TEXT,
    size_bytes INTEGER,
    created_at TIMESTAMP
);
```

## 3.5 Fledge Build Pipeline

### OCI Rootfs Strategy:
```
fledge.toml → fledge build
    ↓
1. Download OCI image (skopeo)
2. Unpack layers (umoci)
3. Extract OCI config
4. Install kestrel agent → /bin/kestrel
5. Apply file mappings
6. Create squashfs (mksquashfs -comp xz)
7. Generate manifest.json
    ↓
Output: nginx.squashfs + manifest.json
```

### Initramfs Strategy:
```
fledge.toml → fledge build
    ↓
1. Create FHS directory structure
2. Install busybox + symlinks
3. Overlay Docker rootfs (if Dockerfile)
4. Install kestrel agent (modes 1-2)
5. Compile init.c (minimal C init)
6. Create CPIO archive + gzip
7. Generate manifest.json
    ↓
Output: app.cpio.gz + manifest.json
```

---

# 4. THE MASTER PLAN

## Timeline Overview

```
┌─────────────────────────────────────────────────────────────┐
│ Phase 0: Clean Boundaries (SEQUENTIAL)        │ 1 week     │
│ ├─ Split fledge.toml / manifest.toml         │            │
│ ├─ Rename "images" → "images"                │            │
│ └─ VM config override system                 │            │
├─────────────────────────────────────────────────────────────┤
│ Phase 1: Foundation (4 PARALLEL TRACKS)       │ 1 week     │
│ ├─ Track A: Env + DNS                        │ 5-6 days   │
│ ├─ Track B: Volumes                          │ 4-5 days   │
│ ├─ Track C: Drift L4                         │ 5-6 days   │
│ └─ Track F: Compose Converter                │ 3-4 days   │
├─────────────────────────────────────────────────────────────┤
│ Phase 2: Stack System (3 PARALLEL TRACKS)     │ 1 week     │
│ ├─ Track D: Stack Orchestrator               │ 7-8 days   │
│ ├─ Track E: Web UI                           │ 5-6 days   │
│ └─ Track G: Fledge Server                    │ 3-4 days   │
├─────────────────────────────────────────────────────────────┤
│ Phase 3: Integration & Polish                 │ 2-3 days   │
│ ├─ Merge all branches                        │            │
│ ├─ Integration testing                       │            │
│ └─ Documentation                             │            │
└─────────────────────────────────────────────────────────────┘

TOTAL: ~3 weeks (vs 8 weeks sequential)
SAVINGS: 62% time reduction via parallelization
```

## Dependency Graph

```
Phase 0 (Sequential - Required)
    ↓
    ├─→ Track A: Env + DNS ────────────┐
    ├─→ Track B: Volumes ──────────────┤→ Track D: Stack Orchestrator
    ├─→ Track C: Drift L4              │
    └─→ Track F: Compose Converter     ↓
                                    Track E: Web UI
                                    Track G: Fledge Server
                                        ↓
                                    Integration
```

**Key Insights:**
- Phase 0 blocks everything (changes core architecture)
- Track A must complete before Track D (Stack needs env+DNS)
- Track B should merge before Track D (DB migrations)
- Tracks C, F, G are fully independent
- Track E can develop against mock APIs while D is in progress

## File Conflict Matrix

| Track | Files Modified | Conflicts With |
|-------|----------------|----------------|
| **Phase 0** | `imagespec/`, `config/`, `orchestrator/` | ALL (that's why it's sequential) |
| **Track A** | `imagespec/`, `agent/`, `orchestrator/`, `dns/` (NEW) | Phase 0 only |
| **Track B** | `volumes/` (NEW), `db/sqlite/` (migrations) | Phase 0, Track D (DB) |
| **Track C** | `drift/`, `driftd/` | Phase 0 only |
| **Track F** | `compose/` (NEW) | Phase 0 only |
| **Track D** | `stack/` (NEW), `db/sqlite/` (migrations) | Phase 0, Track A, Track B |
| **Track E** | `web/` (separate repo) | None |
| **Track G** | `fledge/` (separate repo) | None |

**Merge Order (to minimize conflicts):**
```
Phase 0 → main
  ↓
Track B → main  (DB migrations first)
Track F → main  (no conflicts)
Track C → main  (isolated)
Track A → main  (after B for DB safety)
  ↓
Track G → main  (separate repo)
Track D → main  (needs A + B)
Track E → main  (separate repo)
```

---

# 5. PHASE 0: CLEAN BOUNDARIES

## Overview

**Duration:** 5-7 days (3-5 dev, 2 docs/testing)
**Worktree:** `volant-phase0`
**Branch:** `phase-0-clean-boundaries`
**Status:** REQUIRED - BLOCKS ALL OTHER WORK

## The Problem

Current confusion:
- `fledge.toml` mixes build-time AND runtime concerns
- `manifest.json` is both generated AND hand-editable
- No clear override hierarchy (image defaults vs VM overrides)
- Terminology confusion ("images" means different things)

## The Solution

**Clean Separation:**
```
BUILD TIME (Developer):
  fledge.toml       → Build strategy, source, filesystem type
  manifest.toml     → Runtime defaults (CPU, memory, workload)
  ↓ fledge build
  manifest.json     → Generated (never hand-edited)
  artifact.img      → Built artifact

RUNTIME (Operator):
  volar images install → Installs manifest.json as "image"
  volar vms create    → Creates VM with overrides
```

**Override Hierarchy:**
```
VM creation flags (highest priority)
    ↓ overrides
Image defaults (from manifest.json)
    ↓ overrides
System defaults (lowest priority)
```

## Measurable Tasks

### 5.1 Fledge Changes

#### Task 5.1.1: Split Config Schema
**File:** `fledge/internal/config/schema.go`

**Current:**
```go
type Config struct {
    Version    string
    Strategy   string
    Agent      *AgentConfig
    Source     SourceConfig
    Filesystem *FilesystemConfig

    // MIXED: These are runtime, not build-time
    Resources  ResourcesConfig  // ❌ REMOVE
    Workload   WorkloadConfig   // ❌ REMOVE
    Network    *NetworkConfig   // ❌ REMOVE
    Env        map[string]string // ❌ REMOVE
}
```

**Target:**
```go
// Build-time only
type Config struct {
    Version    string
    Strategy   string              // "oci_rootfs" or "initramfs"
    Agent      *AgentConfig        // Kestrel source
    Init       *InitConfig         // Init mode
    Source     SourceConfig        // Image/Dockerfile/Busybox
    Filesystem *FilesystemConfig   // Squashfs/ext4/xfs
    Mappings   map[string]string   // File injections
}

// Runtime template (NEW)
type ManifestTemplate struct {
    SchemaVersion string
    Name          string
    Version       string
    Runtime       string

    // Runtime concerns
    Resources     *ResourcesConfig
    Workload      *WorkloadConfig
    Env           map[string]string
    Network       *NetworkConfig
    Actions       map[string]Action
    CloudInit     *CloudInitConfig
    Devices       *DevicesConfig
}
```

**Checklist:**
- [ ] Add `ManifestTemplate` struct to `schema.go`
- [ ] Remove runtime fields from `Config` struct
- [ ] Update validation functions
- [ ] Add `LoadManifestTemplate(path string)` function
- [ ] Write tests for new schema

#### Task 5.1.2: Update Build Commands
**File:** `fledge/cmd/fledge/main.go`

**Current:**
```bash
fledge build -c fledge.toml -o output.img
```

**Target:**
```bash
fledge build -c fledge.toml -m manifest.toml -o output.img
```

**Checklist:**
- [ ] Add `--manifest` flag to build command
- [ ] Update flag descriptions
- [ ] Load both config files
- [ ] Pass both to builder
- [ ] Update help text

#### Task 5.1.3: Update Manifest Generation
**Files:** `fledge/internal/builder/oci_rootfs.go`, `fledge/internal/builder/initramfs.go`

**Current:**
```go
func (b *Builder) generateManifest() error {
    // Generates manifest from build config (mixed concerns)
}
```

**Target:**
```go
func (b *Builder) generateManifest(
    buildCfg *config.Config,
    manifestTpl *config.ManifestTemplate,
    artifactPath string,
    checksum string,
) (*imagespec.ImageManifest, error) {
    // Merge template + build metadata
    manifest := manifestTpl.ToImageManifest()

    // Add build outputs
    if buildCfg.Strategy == "oci_rootfs" {
        manifest.Rootfs = &imagespec.RootfsConfig{
            URL:      "file://" + artifactPath,
            Format:   buildCfg.Filesystem.Type,
            Checksum: "sha256:" + checksum,
        }
    } else {
        manifest.Initramfs = &imagespec.InitramfsConfig{
            URL:      "file://" + artifactPath,
            Checksum: "sha256:" + checksum,
        }
    }

    return manifest, nil
}
```

**Checklist:**
- [ ] Update `generateManifest()` signature
- [ ] Implement `ToImageManifest()` method
- [ ] Update `Build()` method to pass both configs
- [ ] Update tests
- [ ] Verify manifest.json output format

#### Task 5.1.4: Update Documentation
**Files:** `fledge/docs/`, `fledge/README.md`

**Checklist:**
- [ ] Create `docs/build-vs-runtime.md`
- [ ] Split all examples into fledge.toml + manifest.toml
- [ ] Update README with new command syntax
- [ ] Create migration guide (old → new format)
- [ ] Update quick-start guides

### 5.2 Volant Changes

#### Task 5.2.1: Rename "Images" → "Images"
**Files:** Multiple throughout `volant/`

**Database Migration:**
```sql
-- Already named "images", just verify terminology
SELECT name FROM sqlite_master WHERE type='table' AND name='images';
-- Expected: 'images'
```

**Checklist:**
- [ ] Update all comments referencing "images"
- [ ] Update CLI help text (`volar images --help`)
- [ ] Update API endpoint docs
- [ ] Update variable names (keep `image` in code, update docs)
- [ ] Search for "image" comments and update context

#### Task 5.2.2: Add Environment Variable Support
**File:** `volant/internal/imagespec/spec.go`

**Current:**
```go
type WorkloadConfig struct {
    Type       string   // "exec", "http", "grpc"
    Entrypoint []string
    Args       []string
    BaseURL    string
}
```

**Target:**
```go
type WorkloadConfig struct {
    Type       string
    Entrypoint []string
    Args       []string
    BaseURL    string
    Env        map[string]string  // NEW: Environment variables
}
```

**Checklist:**
- [ ] Add `Env` field to `WorkloadConfig`
- [ ] Update `Validate()` to check env vars
- [ ] Update `Encode()` to include env in gzip payload
- [ ] Update `Decode()` to extract env

#### Task 5.2.3: Implement VM Config Overrides
**File:** `volant/pkg/config/vmconfig.go`

**NEW:**
```go
// VM configuration with override system
type VMConfig struct {
    // Image reference
    ImageName string

    // Override fields (nil = use image defaults)
    CPUCoresOverride    *int
    MemoryMBOverride    *int
    EnvOverrides        map[string]string
    PortOverrides       []PortMapping
    EntrypointOverride  *string
    ArgsOverride        []string

    // Final merged config (computed)
    FinalCPUCores   int
    FinalMemoryMB   int
    FinalEnv        map[string]string
    FinalEntrypoint string
    FinalArgs       []string
    FinalPorts      []PortMapping
}

// Merge image defaults + VM overrides
func MergeConfig(manifest *imagespec.ImageManifest, overrides *VMConfigOverrides) *VMConfig {
    cfg := &VMConfig{
        ImageName: manifest.Name,
    }

    // Merge resources
    cfg.FinalCPUCores = manifest.Resources.CPUCores
    if overrides.CPUCores != nil {
        cfg.FinalCPUCores = *overrides.CPUCores
    }

    cfg.FinalMemoryMB = manifest.Resources.MemoryMB
    if overrides.MemoryMB != nil {
        cfg.FinalMemoryMB = *overrides.MemoryMB
    }

    // Merge env vars (image defaults + overrides)
    cfg.FinalEnv = make(map[string]string)
    for k, v := range manifest.Env {
        cfg.FinalEnv[k] = v
    }
    for k, v := range overrides.Env {
        cfg.FinalEnv[k] = v  // Overrides win
    }

    // Merge entrypoint/args
    cfg.FinalEntrypoint = manifest.Workload.Entrypoint
    if overrides.Entrypoint != nil {
        cfg.FinalEntrypoint = *overrides.Entrypoint
    }

    cfg.FinalArgs = manifest.Workload.Args
    if overrides.Args != nil {
        cfg.FinalArgs = overrides.Args
    }

    // Merge ports
    cfg.FinalPorts = manifest.Network.Expose
    if len(overrides.Ports) > 0 {
        cfg.FinalPorts = overrides.Ports
    }

    return cfg
}
```

**Checklist:**
- [ ] Create `vmconfig.go` with override types
- [ ] Implement `MergeConfig()` function
- [ ] Write tests for merge logic
- [ ] Handle edge cases (nil checks)

#### Task 5.2.4: Update Orchestrator
**File:** `volant/internal/server/orchestrator/orchestrator.go`

**Current:**
```go
func (o *Orchestrator) CreateVM(ctx context.Context, req CreateVMRequest) (*VM, error) {
    // Loads image, creates VM, no override system
}
```

**Target:**
```go
func (o *Orchestrator) CreateVM(
    ctx context.Context,
    name string,
    imageName string,
    overrides *VMConfigOverrides,
) (*VM, error) {
    // 1. Load image manifest
    image, err := o.db.Images().GetByName(ctx, imageName)
    if err != nil {
        return nil, fmt.Errorf("image not found: %w", err)
    }

    // 2. Merge image defaults + overrides
    vmConfig := config.MergeConfig(image.Manifest, overrides)

    // 3. Encode final config into kernel cmdline
    manifest := vmConfig.ToManifest()  // Convert back to manifest format
    manifestEncoded := imagespec.Encode(manifest)
    cmdline := fmt.Sprintf("volant.manifest=%s volant.runtime=%s", manifestEncoded, manifest.Runtime)

    // 4. Continue with existing flow...
    // (IP allocation, network setup, launch, etc.)
}
```

**Checklist:**
- [ ] Update `CreateVM()` signature
- [ ] Implement config merging
- [ ] Update manifest encoding to include merged config
- [ ] Update all callers (CLI, API)
- [ ] Write integration tests

#### Task 5.2.5: Update HTTP API
**File:** `volant/internal/server/httpapi/httpapi.go`

**Current:**
```go
type CreateVMRequest struct {
    Name      string `json:"name"`
    Image     string `json:"image"`
    Runtime   string `json:"runtime"`
    CPUCores  int    `json:"cpu_cores"`
    MemoryMB  int    `json:"memory_mb"`
}
```

**Target:**
```go
type CreateVMRequest struct {
    Name      string            `json:"name"`
    ImageName string            `json:"image"`  // Renamed for clarity

    // Overrides (all optional)
    CPUCores   *int              `json:"cpu_cores,omitempty"`
    MemoryMB   *int              `json:"memory_mb,omitempty"`
    Env        map[string]string `json:"env,omitempty"`
    Entrypoint *string           `json:"entrypoint,omitempty"`
    Args       []string          `json:"args,omitempty"`
    Ports      []PortMapping     `json:"ports,omitempty"`
}

type PortMapping struct {
    GuestPort uint16 `json:"guest_port"`
    HostPort  uint16 `json:"host_port"`
    Protocol  string `json:"protocol"`  // "tcp" or "udp"
}
```

**Checklist:**
- [ ] Update `CreateVMRequest` struct
- [ ] Update request validation
- [ ] Update handler to pass overrides
- [ ] Update OpenAPI spec
- [ ] Update API documentation

#### Task 5.2.6: Update CLI
**File:** `volant/internal/cli/standard/cli.go`

**Current:**
```bash
volar vms create myvm --image nginx --cpu 2 --memory 512
```

**Target:**
```bash
# Use image defaults
volar vms create myvm --image nginx

# Override specific fields
volar vms create myvm --image nginx \
  --cpu 4 \
  --memory 1024 \
  --env LOG_LEVEL=debug \
  --env DATABASE_URL=postgres://... \
  --port 8080:80 \
  --entrypoint /usr/local/bin/myapp

# Flags:
#   --image (required): Image name
#   --cpu (optional): Override CPU cores
#   --memory (optional): Override memory MB
#   --env (repeatable): Add/override env var
#   --port (repeatable): Port mapping HOST:GUEST
#   --entrypoint (optional): Override entrypoint
#   --args (optional): Override args
```

**Checklist:**
- [ ] Update `vms create` command flags
- [ ] Add `--env` flag (StringToString, repeatable)
- [ ] Add `--port` flag (StringArray, parse HOST:GUEST)
- [ ] Add `--entrypoint` flag (optional string)
- [ ] Add `--args` flag (StringArray)
- [ ] Update help text with examples
- [ ] Update command descriptions

### 5.3 Documentation Updates

#### Task 5.3.1: Create Concepts Documentation
**Directory:** `volant/docs/0_concepts/` (NEW)

**Files to create:**
- [ ] `build-vs-runtime.md` - Explain fledge.toml vs manifest.toml
- [ ] `images-vs-vms.md` - Explain image = template, VM = instance
- [ ] `override-hierarchy.md` - Document precedence rules

**Content for `build-vs-runtime.md`:**
```markdown
# Build-Time vs Runtime Configuration

## Overview

Volant cleanly separates build-time and runtime concerns:

**Build-Time (fledge.toml):**
- What to build: Docker image, Dockerfile, busybox
- How to build: OCI rootfs or initramfs strategy
- Build optimizations: Compression level, filesystem type
- Artifact sourcing: Kestrel agent from GitHub/local/HTTP

**Runtime (manifest.toml):**
- Resource defaults: CPU cores, memory MB
- Workload config: Entrypoint, args, environment variables
- Network defaults: Ports to expose, network mode
- Actions: Custom API operations

**Result (manifest.json):**
- Generated by Fledge (NEVER hand-edited)
- Merges manifest.toml + build metadata
- Contains artifact URLs, checksums, formats

## Example

### fledge.toml (build-time)
```toml
version = "1"
strategy = "oci_rootfs"

[agent]
source_strategy = "release"
version = "latest"

[source]
image = "nginx:alpine"

[filesystem]
type = "squashfs"
compression_level = 15
```

### manifest.toml (runtime defaults)
```toml
schema_version = "v1"
name = "nginx"
version = "1.0.0"
runtime = "nginx"

[resources]
cpu_cores = 2
memory_mb = 512

[workload]
entrypoint = ["/usr/sbin/nginx", "-g", "daemon off;"]

[env]
WORKER_PROCESSES = "auto"
LOG_LEVEL = "info"

[network]
mode = "bridged"
expose = [{ port = 80, protocol = "tcp" }]
```

### VM Creation with Overrides
```bash
# Use defaults
volar vms create nginx-default --image nginx

# Override CPU and env
volar vms create nginx-custom --image nginx \
  --cpu 4 \
  --memory 1024 \
  --env LOG_LEVEL=debug
```
```

#### Task 5.3.2: Update Reference Docs
**Files:** `volant/docs/6_reference/`

**Checklist:**
- [ ] Update `fledge-toml-schema.md` (build-time only)
- [ ] Create `manifest-toml-schema.md` (runtime template)
- [ ] Update `manifest-json-schema.md` (generated output)
- [ ] Update `api-reference.md` (new CreateVMRequest format)
- [ ] Update `cli-reference.md` (new flags)

#### Task 5.3.3: Update All Examples
**Files:** `volant/docs/examples/`, `fledge/docs/examples/`

**Checklist:**
- [ ] Split nginx example into fledge.toml + manifest.toml
- [ ] Split postgres example
- [ ] Split custom app example
- [ ] Add override examples
- [ ] Show env var merging examples

### 5.4 Testing

#### Task 5.4.1: Unit Tests
**Checklist:**
- [ ] Test `config.MergeConfig()` with all override scenarios
- [ ] Test env var merging (defaults + overrides)
- [ ] Test nil override handling
- [ ] Test port mapping parsing
- [ ] Test manifest encoding with env vars

#### Task 5.4.2: Integration Tests
**Checklist:**
- [ ] Build image with new fledge.toml + manifest.toml format
- [ ] Install image into Volant
- [ ] Create VM with defaults (no overrides)
- [ ] Create VM with all overrides
- [ ] Verify env vars inside VM
- [ ] Verify resource allocations
- [ ] Test port mappings

**Test Script:**
```bash
#!/bin/bash
set -e

echo "=== Phase 0 Integration Test ==="

# 1. Create test image
cd /tmp/test-nginx
cat > fledge.toml <<EOF
version = "1"
strategy = "oci_rootfs"

[agent]
source_strategy = "release"
version = "latest"

[source]
image = "nginx:alpine"

[filesystem]
type = "squashfs"
compression_level = 15
EOF

cat > manifest.toml <<EOF
schema_version = "v1"
name = "nginx"
version = "1.0.0"
runtime = "nginx"

[resources]
cpu_cores = 2
memory_mb = 512

[workload]
entrypoint = ["/usr/sbin/nginx", "-g", "daemon off;"]

[env]
WORKER_PROCESSES = "auto"
LOG_LEVEL = "info"

[network]
mode = "bridged"
expose = [{ port = 80, protocol = "tcp" }]
EOF

# 2. Build image
sudo fledge build -c fledge.toml -m manifest.toml -o nginx.img

# Verify outputs
test -f nginx.squashfs || { echo "ERROR: nginx.squashfs not found"; exit 1; }
test -f nginx.manifest.json || { echo "ERROR: nginx.manifest.json not found"; exit 1; }

# 3. Install image
volar images install --manifest nginx.manifest.json

# Verify installation
volar images list | grep nginx || { echo "ERROR: Image not installed"; exit 1; }

# 4. Create VM with defaults
volar vms create nginx-default --image nginx

# Verify defaults used
VM_INFO=$(volar vms get nginx-default --output json)
CPU=$(echo "$VM_INFO" | jq -r '.cpu_cores')
MEM=$(echo "$VM_INFO" | jq -r '.memory_mb')

test "$CPU" = "2" || { echo "ERROR: Expected CPU=2, got $CPU"; exit 1; }
test "$MEM" = "512" || { echo "ERROR: Expected Memory=512, got $MEM"; exit 1; }

# 5. Create VM with overrides
volar vms create nginx-custom --image nginx \
  --cpu 4 \
  --memory 1024 \
  --env LOG_LEVEL=debug \
  --env WORKER_PROCESSES=4 \
  --port 9090:80

# Verify overrides applied
VM_INFO=$(volar vms get nginx-custom --output json)
CPU=$(echo "$VM_INFO" | jq -r '.cpu_cores')
MEM=$(echo "$VM_INFO" | jq -r '.memory_mb')

test "$CPU" = "4" || { echo "ERROR: Expected CPU=4, got $CPU"; exit 1; }
test "$MEM" = "1024" || { echo "ERROR: Expected Memory=1024, got $MEM"; exit 1; }

# 6. Verify env vars inside VM
volar exec nginx-custom -- env | grep 'LOG_LEVEL=debug' || { echo "ERROR: Env override failed"; exit 1; }
volar exec nginx-custom -- env | grep 'WORKER_PROCESSES=4' || { echo "ERROR: Env override failed"; exit 1; }

# 7. Cleanup
volar vms destroy nginx-default nginx-custom
volar images remove nginx

echo "=== Phase 0 Integration Test PASSED ==="
```

### 5.5 Success Criteria

**Phase 0 Complete When:**
- [ ] `fledge.toml` contains ONLY build concerns
- [ ] `manifest.toml` contains ONLY runtime defaults
- [ ] `manifest.json` is generated, never hand-edited
- [ ] Images are immutable templates in Volant
- [ ] VMs can override: CPU, memory, env, ports, entrypoint, args
- [ ] Override hierarchy is clear: VM flags > Image defaults > System defaults
- [ ] All examples updated to show new format
- [ ] All documentation explains clean separation
- [ ] Integration tests pass
- [ ] **Branch merged to main** ← Unblocks Phase 1

---

# 6. PHASE 1 TRACK A: Environment Variables + DNS

## Overview

**Duration:** 5-6 days
**Worktree:** `volant-track-a`
**Branch:** `phase-1-track-a-env-dns`
**Dependencies:** Phase 0 ✓
**Conflicts:** None (unique files)

## What This Enables

**Before:**
```bash
# Can't configure apps at runtime
# Must bake config into build artifacts
```

**After:**
```bash
# Environment variables from manifest
volar vms create api --image myapp

# Override env vars
volar vms create api --image myapp --env DATABASE_URL=postgres://...

# Service discovery via DNS
DATABASE_URL=postgres://database.volant:5432/mydb
# database.volant auto-resolves to 192.168.127.10
```

## Measurable Tasks

### 6.1 Environment Variable Support

#### Task 6.1.1: Extend Manifest Spec
**File:** `volant/internal/imagespec/spec.go`

**Already done in Phase 0, verify:**
```go
type WorkloadConfig struct {
    Type       string
    Entrypoint []string
    Args       []string
    BaseURL    string
    Env        map[string]string  // ✓ Added in Phase 0
}
```

**Checklist:**
- [ ] Verify `Env` field exists
- [ ] Verify `Encode()` includes env
- [ ] Verify `Decode()` extracts env

#### Task 6.1.2: Encode Env into Kernel Cmdline
**File:** `volant/internal/server/orchestrator/orchestrator.go`

**Current (Phase 0):**
```go
cmdline := fmt.Sprintf("volant.manifest=%s", manifestEncoded)
```

**Env is already in manifest, but add explicit param for easier parsing:**
```go
// Encode env vars separately for easier guest parsing
envEncoded := base64.StdEncoding.EncodeToString([]byte(jsonMarshal(vmConfig.FinalEnv)))
cmdline := fmt.Sprintf("volant.manifest=%s volant.env=%s", manifestEncoded, envEncoded)
```

**Checklist:**
- [ ] Add env encoding to `CreateVM()`
- [ ] Base64 encode env vars separately
- [ ] Append to kernel cmdline
- [ ] Document cmdline format

#### Task 6.1.3: Decode and Apply Env in Guest
**File:** `volant/internal/agent/app/app.go`

**Add env extraction and application:**
```go
func (a *Agent) Start(ctx context.Context) error {
    // Existing: Decode manifest
    manifest := imagespec.Decode(cmdlineManifest)

    // NEW: Decode env vars
    envParam := getCmdlineParam("volant.env")
    envVars := decodeEnvVars(envParam)  // base64 → JSON → map

    // Merge manifest env + cmdline env (cmdline wins)
    finalEnv := make(map[string]string)
    for k, v := range manifest.Workload.Env {
        finalEnv[k] = v
    }
    for k, v := range envVars {
        finalEnv[k] = v  // Override
    }

    // Apply to workload process
    cmd := exec.CommandContext(ctx, manifest.Workload.Entrypoint[0], manifest.Workload.Entrypoint[1:]...)
    for k, v := range finalEnv {
        cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
    }

    // Start workload
    return cmd.Start()
}

func decodeEnvVars(encoded string) map[string]string {
    decoded, _ := base64.StdEncoding.DecodeString(encoded)
    var env map[string]string
    json.Unmarshal(decoded, &env)
    return env
}
```

**Checklist:**
- [ ] Add `getCmdlineParam()` helper
- [ ] Implement `decodeEnvVars()`
- [ ] Merge manifest env + cmdline env
- [ ] Apply env to workload exec
- [ ] Test env vars are visible in guest

### 6.2 DNS Service Discovery

#### Task 6.2.1: Create DNS Server Package
**File:** `volant/internal/server/dns/server.go` (NEW)

**Full implementation:**
```go
package dns

import (
    "context"
    "fmt"
    "net"
    "strings"

    "github.com/miekg/dns"
    "github.com/volantvm/volant/pkg/db"
)

type Server struct {
    db      db.Store
    listen  string      // "192.168.127.1:53"
    domain  string      // "volant"
    server  *dns.Server
}

func New(db db.Store, listen, domain string) *Server {
    return &Server{
        db:     db,
        listen: listen,
        domain: domain,
    }
}

func (s *Server) Start(ctx context.Context) error {
    dns.HandleFunc(s.domain+".", s.handleQuery)
    dns.HandleFunc(".", s.handleQuery)  // Catch-all

    s.server = &dns.Server{
        Addr: s.listen,
        Net:  "udp",
    }

    go func() {
        <-ctx.Done()
        s.server.Shutdown()
    }()

    return s.server.ListenAndServe()
}

func (s *Server) handleQuery(w dns.ResponseWriter, r *dns.Msg) {
    m := new(dns.Msg)
    m.SetReply(r)
    m.Authoritative = true

    for _, q := range r.Question {
        if q.Qtype == dns.TypeA {
            // Extract service name
            name := strings.TrimSuffix(q.Name, "."+s.domain+".")
            name = strings.TrimSuffix(name, ".")

            // Resolve to IP(s)
            ips, err := s.resolveService(name)
            if err != nil || len(ips) == 0 {
                m.SetRcode(r, dns.RcodeNameError)
                w.WriteMsg(m)
                return
            }

            // Add A records
            for _, ip := range ips {
                rr := &dns.A{
                    Hdr: dns.RR_Header{
                        Name:   q.Name,
                        Rrtype: dns.TypeA,
                        Class:  dns.ClassINET,
                        Ttl:    10,  // Short TTL for dynamic updates
                    },
                    A: net.ParseIP(ip),
                }
                m.Answer = append(m.Answer, rr)
            }
        }
    }

    w.WriteMsg(m)
}

func (s *Server) resolveService(name string) ([]string, error) {
    ctx := context.Background()

    // Try VM lookup first
    vm, err := s.db.VMs().GetByName(ctx, name)
    if err == nil && vm.IPAddress != "" {
        return []string{vm.IPAddress}, nil
    }

    // Try deployment lookup (round-robin)
    deployment, err := s.db.VMGroups().GetByName(ctx, name)
    if err == nil {
        vms, err := s.db.VMs().ListByGroup(ctx, deployment.ID)
        if err == nil {
            var ips []string
            for _, vm := range vms {
                if vm.IPAddress != "" && vm.Status == "running" {
                    ips = append(ips, vm.IPAddress)
                }
            }
            return ips, nil
        }
    }

    return nil, fmt.Errorf("service not found: %s", name)
}
```

**Checklist:**
- [ ] Create `dns/server.go`
- [ ] Implement `Start()` method
- [ ] Implement `handleQuery()` for A records
- [ ] Implement `resolveService()` (VM + deployment lookup)
- [ ] Add dependency: `github.com/miekg/dns` to go.mod
- [ ] Write unit tests for DNS resolution

#### Task 6.2.2: Add DNS Configuration
**File:** `volant/internal/server/config/config.go`

**Add DNS fields:**
```go
type ServerConfig struct {
    // ... existing fields ...

    // DNS
    DNSEnabled bool   // VOLANT_DNS_ENABLED (default: true)
    DNSListen  string // VOLANT_DNS_LISTEN (default: 192.168.127.1:53)
    DNSDomain  string // VOLANT_DNS_DOMAIN (default: volant)
    DNSUpstreams []string // VOLANT_DNS_UPSTREAMS (fallback, comma-separated)
}

func FromEnv() (*ServerConfig, error) {
    // ... existing code ...

    cfg.DNSEnabled = os.Getenv("VOLANT_DNS_ENABLED") != "false"
    cfg.DNSListen = getEnvOrDefault("VOLANT_DNS_LISTEN", "192.168.127.1:53")
    cfg.DNSDomain = getEnvOrDefault("VOLANT_DNS_DOMAIN", "volant")
    upstreams := os.Getenv("VOLANT_DNS_UPSTREAMS")
    if upstreams != "" {
        cfg.DNSUpstreams = strings.Split(upstreams, ",")
    } else {
        cfg.DNSUpstreams = []string{"1.1.1.1:53", "8.8.8.8:53"}
    }

    return cfg, nil
}
```

**Checklist:**
- [ ] Add DNS fields to `ServerConfig`
- [ ] Parse env vars in `FromEnv()`
- [ ] Set sensible defaults
- [ ] Document env vars

#### Task 6.2.3: Start DNS Server in volantd
**File:** `cmd/volantd/main.go`

**Add DNS startup:**
```go
func main() {
    // ... existing setup (config, db, orchestrator, etc.) ...

    // Start DNS server
    if cfg.DNSEnabled {
        dnsServer := dns.New(store, cfg.DNSListen, cfg.DNSDomain)
        go func() {
            logger.Info("Starting DNS server", "listen", cfg.DNSListen, "domain", cfg.DNSDomain)
            if err := dnsServer.Start(ctx); err != nil {
                logger.Error("DNS server failed", "error", err)
            }
        }()
    } else {
        logger.Info("DNS server disabled")
    }

    // ... rest of main (HTTP server, etc.) ...
}
```

**Checklist:**
- [ ] Import DNS package
- [ ] Start DNS server in goroutine
- [ ] Add logging
- [ ] Handle graceful shutdown
- [ ] Test DNS starts correctly

#### Task 6.2.4: Auto-Configure VMs with Nameserver
**File:** `volant/internal/server/orchestrator/cloudinit/builder.go`

**For VMs with cloud-init:**
```go
func (b *Builder) GenerateUserData(vmName string, config *CloudInitConfig) (string, error) {
    userData := `#cloud-config

write_files:
  - path: /etc/resolv.conf
    content: |
      nameserver 192.168.127.1
      search volant
    permissions: '0644'

runcmd:
  - echo "nameserver 192.168.127.1" > /etc/resolv.conf
  - echo "search volant" >> /etc/resolv.conf
`

    // Append user-provided cloud-init
    if config != nil && config.UserData != "" {
        userData += "\n" + config.UserData
    }

    return userData, nil
}
```

**Checklist:**
- [ ] Update `GenerateUserData()` to inject nameserver
- [ ] Preserve user-provided cloud-init config
- [ ] Test resolv.conf is created correctly

**For initramfs (no cloud-init):**
**File:** `volant/internal/builder/initramfs.go`

```go
func (b *InitramfsBuilder) createResolvConf(rootfs string) error {
    resolvConf := "nameserver 192.168.127.1\nsearch volant\n"
    return os.WriteFile(
        filepath.Join(rootfs, "etc", "resolv.conf"),
        []byte(resolvConf),
        0644,
    )
}

func (b *InitramfsBuilder) Build() error {
    // ... existing build steps ...

    // NEW: Create /etc/resolv.conf
    if err := b.createResolvConf(b.rootfsPath); err != nil {
        return fmt.Errorf("failed to create resolv.conf: %w", err)
    }

    // ... continue with CPIO creation ...
}
```

**Checklist:**
- [ ] Add `createResolvConf()` helper
- [ ] Call during build process
- [ ] Test resolv.conf exists in initramfs

#### Task 6.2.5: Add Database Query Methods
**File:** `volant/internal/server/db/sqlite/sqlite.go`

**Add methods for DNS lookups:**
```go
func (r *VMRepository) ListByGroup(ctx context.Context, groupID int64) ([]db.VM, error) {
    query := `SELECT * FROM vms WHERE group_id = ?`
    rows, err := r.db.QueryContext(ctx, query, groupID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var vms []db.VM
    for rows.Next() {
        var vm db.VM
        if err := rows.Scan(&vm.ID, &vm.Name, &vm.Status, &vm.IPAddress, ...); err != nil {
            return nil, err
        }
        vms = append(vms, vm)
    }
    return vms, nil
}
```

**Checklist:**
- [ ] Add `ListByGroup()` method to `VMRepository`
- [ ] Add `GetByName()` method to `VMGroupRepository` (if not exists)
- [ ] Test queries return correct results

### 6.3 Integration Testing

#### Task 6.3.1: Test Environment Variables
**Script:** `test-env-vars.sh`

```bash
#!/bin/bash
set -e

echo "=== Testing Environment Variables ==="

# 1. Create image with default env
cat > manifest.toml <<EOF
[env]
LOG_LEVEL = "info"
DATABASE_URL = "postgres://localhost:5432/mydb"
EOF

# Build and install image
sudo fledge build -c fledge.toml -m manifest.toml -o myapp.img
volar images install --manifest myapp.manifest.json

# 2. Create VM with defaults
volar vms create app-default --image myapp

# Verify defaults
volar exec app-default -- env | grep 'LOG_LEVEL=info'
volar exec app-default -- env | grep 'DATABASE_URL=postgres://localhost:5432/mydb'

# 3. Create VM with overrides
volar vms create app-override --image myapp \
  --env LOG_LEVEL=debug \
  --env DATABASE_URL=postgres://database.volant:5432/mydb

# Verify overrides
volar exec app-override -- env | grep 'LOG_LEVEL=debug'
volar exec app-override -- env | grep 'DATABASE_URL=postgres://database.volant:5432/mydb'

# Cleanup
volar vms destroy app-default app-override
volar images remove myapp

echo "=== Environment Variables Test PASSED ==="
```

**Checklist:**
- [ ] Run test script
- [ ] Verify defaults work
- [ ] Verify overrides work
- [ ] Verify env vars visible in guest

#### Task 6.3.2: Test DNS Resolution
**Script:** `test-dns.sh`

```bash
#!/bin/bash
set -e

echo "=== Testing DNS Service Discovery ==="

# 1. Start volantd with DNS enabled
VOLANT_DNS_ENABLED=true volantd &
VOLANTD_PID=$!
sleep 2

# 2. Create database VM
volar vms create postgres --image postgres-db

# Get assigned IP
DB_IP=$(volar vms get postgres --output json | jq -r '.ip_address')
echo "Database IP: $DB_IP"

# 3. Test DNS resolution from host
RESOLVED_IP=$(dig @192.168.127.1 +short postgres.volant)
test "$RESOLVED_IP" = "$DB_IP" || { echo "ERROR: DNS resolution failed"; exit 1; }

# 4. Create API VM that references DB by name
volar vms create api --image myapp \
  --env DATABASE_URL=postgres://postgres.volant:5432/mydb

# 5. Test DNS resolution from inside VM
volar exec api -- nslookup postgres.volant | grep "$DB_IP"

# 6. Test short name resolution (search domain)
volar exec api -- ping -c 1 postgres

# 7. Test deployment round-robin
volar deployments create web --image nginx --replicas 3

# Should return multiple IPs
dig @192.168.127.1 +short web.volant | wc -l | grep 3

# Cleanup
volar deployments delete web
volar vms destroy api postgres
kill $VOLANTD_PID

echo "=== DNS Service Discovery Test PASSED ==="
```

**Checklist:**
- [ ] Run test script
- [ ] Verify VM names resolve to IPs
- [ ] Verify short names work (search domain)
- [ ] Verify deployment round-robin
- [ ] Verify DNS works inside VMs

### 6.4 Documentation

#### Task 6.4.1: Document Environment Variables
**File:** `volant/docs/3_guides/environment-variables.md` (NEW)

**Content:**
```markdown
# Environment Variables

## Overview

Environment variables can be set at two levels:

1. **Image defaults** (manifest.toml)
2. **VM overrides** (volar vms create --env)

VM overrides take precedence over image defaults.

## Setting Defaults in Images

### manifest.toml
```toml
[env]
LOG_LEVEL = "info"
DATABASE_URL = "postgres://localhost:5432/mydb"
API_KEY = "default-key"
```

These become the default env vars for all VMs created from this image.

## Overriding at VM Creation

```bash
volar vms create myvm --image myapp \
  --env LOG_LEVEL=debug \
  --env DATABASE_URL=postgres://database.volant:5432/mydb
```

## Accessing in Workload

Environment variables are automatically available to your workload process:

```bash
# Inside VM
echo $LOG_LEVEL
# Output: debug
```

## Use Cases

### 12-Factor Apps
```bash
# Development
volar vms create app-dev --image myapp \
  --env ENVIRONMENT=development \
  --env DATABASE_URL=postgres://dev-db.volant:5432/mydb

# Production
volar vms create app-prod --image myapp \
  --env ENVIRONMENT=production \
  --env DATABASE_URL=postgres://prod-db.volant:5432/mydb
```

### Service Discovery
```bash
# Reference services by name
volar vms create api --image myapp \
  --env DATABASE_URL=postgres://postgres.volant:5432/mydb \
  --env CACHE_URL=redis://cache.volant:6379
```
```

**Checklist:**
- [ ] Create `environment-variables.md`
- [ ] Add examples for common use cases
- [ ] Document override behavior
- [ ] Show integration with service discovery

#### Task 6.4.2: Document DNS Service Discovery
**File:** `volant/docs/3_guides/service-discovery.md` (NEW)

**Content:**
```markdown
# Service Discovery via DNS

## Overview

Volant includes an embedded DNS server that automatically resolves VM and deployment names to IPs.

## Configuration

DNS is enabled by default. Configure via environment variables:

```bash
VOLANT_DNS_ENABLED=true             # Enable/disable (default: true)
VOLANT_DNS_LISTEN=192.168.127.1:53  # Bind address (default: host IP:53)
VOLANT_DNS_DOMAIN=volant            # Domain suffix (default: volant)
```

## Resolution Rules

### VM Names
```bash
postgres.volant → 192.168.127.10 (single IP)
```

### Deployment Names
```bash
web-cluster.volant → [192.168.127.11, 192.168.127.12, 192.168.127.13]
# Round-robin DNS (returns multiple A records)
```

### Short Names
```bash
postgres → 192.168.127.10
# Works due to search domain (search volant)
```

## Usage Examples

### In Environment Variables
```bash
volar vms create api --image myapp \
  --env DATABASE_URL=postgres://postgres.volant:5432/mydb
```

### In Application Code
```python
# Python example
import psycopg2

conn = psycopg2.connect("postgres://admin:pass@postgres.volant:5432/mydb")
# postgres.volant resolves to 192.168.127.10
```

### Testing Resolution
```bash
# From host
dig @192.168.127.1 postgres.volant

# From inside VM
volar exec api -- nslookup postgres.volant
volar exec api -- ping postgres
```

## Load Balancing

Deployments return multiple IPs (round-robin):

```bash
volar deployments create web --image nginx --replicas 3
# Creates: web-0, web-1, web-2

dig @192.168.127.1 web.volant
# Returns 3 A records (one for each replica)
```

Most HTTP clients will automatically round-robin across these IPs.
```

**Checklist:**
- [ ] Create `service-discovery.md`
- [ ] Document DNS configuration
- [ ] Show resolution examples
- [ ] Explain round-robin for deployments
- [ ] Add troubleshooting section

### 6.5 Success Criteria

**Track A Complete When:**
- [ ] Environment variables work (manifest defaults + VM overrides)
- [ ] Env vars visible inside VMs
- [ ] DNS server starts with volantd
- [ ] VMs auto-configured with nameserver
- [ ] `<vm-name>.volant` resolves to IP
- [ ] `<deployment-name>.volant` round-robins across replicas
- [ ] Short names work (search domain)
- [ ] Service-to-service communication works
- [ ] All tests pass
- [ ] Documentation complete
- [ ] **Branch merged to main** ← Unblocks Track D

---

# 7. PHASE 1 TRACK B: Volume Manager

## Overview

**Duration:** 4-5 days
**Worktree:** `volant-track-b`
**Branch:** `phase-1-track-b-volumes`
**Dependencies:** Phase 0 ✓
**Conflicts:** DB migrations (merge before Track D)

## What This Enables

**Before:**
```bash
# No persistent storage
# Data lost on VM restart
# Databases can't persist data
```

**After:**
```bash
# Create persistent volume
volar volumes create pgdata --size 10G --type ext4

# Attach to VM
volar vms create postgres --image postgres-db \
  --volume pgdata:/var/lib/postgresql/data

# Data persists across restarts
volar vms restart postgres  # Data still there
```

## Measurable Tasks

### 7.1 Volume Manager Core

#### Task 7.1.1: Create Volume Package
**File:** `volant/internal/server/volumes/manager.go` (NEW)

**Full implementation:**
```go
package volumes

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strconv"

    "github.com/volantvm/volant/pkg/db"
)

type VolumeType string

const (
    VolumeTypeExt4     VolumeType = "ext4"
    VolumeTypeSquashfs VolumeType = "squashfs"
    VolumeTypeBind     VolumeType = "bind"
)

type Volume struct {
    ID         int64
    Name       string
    Type       VolumeType
    SizeGB     int
    Persistent bool
    HostPath   string
    BackupPath string
    CreatedAt  string
    UpdatedAt  string
}

type VolumeMount struct {
    VolumeID   int64
    VMName     string
    MountPoint string
    ReadOnly   bool
}

type Manager struct {
    basePath string  // /var/lib/volant/volumes
    db       db.Store
}

func NewManager(basePath string, db db.Store) *Manager {
    return &Manager{
        basePath: basePath,
        db:       db,
    }
}

func (m *Manager) CreateVolume(ctx context.Context, vol *Volume) error {
    // 1. Validate
    if vol.Name == "" {
        return fmt.Errorf("volume name required")
    }
    if vol.SizeGB <= 0 {
        return fmt.Errorf("volume size must be > 0")
    }

    // 2. Set host path
    vol.HostPath = filepath.Join(m.basePath, vol.Name+".img")

    // 3. Create volume file
    switch vol.Type {
    case VolumeTypeExt4:
        if err := m.createExt4Volume(vol); err != nil {
            return fmt.Errorf("failed to create ext4 volume: %w", err)
        }
    case VolumeTypeSquashfs:
        return fmt.Errorf("squashfs volumes are read-only, use bind mounts instead")
    case VolumeTypeBind:
        // Bind mounts don't need creation, just path validation
        if _, err := os.Stat(vol.HostPath); err != nil {
            return fmt.Errorf("bind mount path does not exist: %s", vol.HostPath)
        }
    default:
        return fmt.Errorf("unsupported volume type: %s", vol.Type)
    }

    // 4. Store in database
    return m.db.Volumes().Create(ctx, vol)
}

func (m *Manager) createExt4Volume(vol *Volume) error {
    // 1. Allocate sparse file
    sizeMB := vol.SizeGB * 1024
    cmd := exec.Command("dd",
        "if=/dev/zero",
        "of="+vol.HostPath,
        "bs=1M",
        "count=0",
        "seek="+strconv.Itoa(sizeMB),
    )
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("dd failed: %w", err)
    }

    // 2. Create ext4 filesystem
    cmd = exec.Command("mkfs.ext4", "-F", vol.HostPath)
    if err := cmd.Run(); err != nil {
        os.Remove(vol.HostPath)  // Cleanup
        return fmt.Errorf("mkfs.ext4 failed: %w", err)
    }

    return nil
}

func (m *Manager) DeleteVolume(ctx context.Context, name string) error {
    vol, err := m.db.Volumes().GetByName(ctx, name)
    if err != nil {
        return fmt.Errorf("volume not found: %w", err)
    }

    // 1. Check if attached to any VMs
    mounts, err := m.db.VolumeMounts().ListByVolume(ctx, vol.ID)
    if err != nil {
        return err
    }
    if len(mounts) > 0 {
        return fmt.Errorf("volume is attached to VMs, detach first")
    }

    // 2. Delete file
    if vol.Type != VolumeTypeBind {
        if err := os.Remove(vol.HostPath); err != nil && !os.IsNotExist(err) {
            return fmt.Errorf("failed to delete volume file: %w", err)
        }
    }

    // 3. Delete from database
    return m.db.Volumes().Delete(ctx, vol.ID)
}

func (m *Manager) AttachVolume(ctx context.Context, vmName, volumeName, mountPoint string, readOnly bool) error {
    // 1. Get volume
    vol, err := m.db.Volumes().GetByName(ctx, volumeName)
    if err != nil {
        return fmt.Errorf("volume not found: %w", err)
    }

    // 2. Get VM
    vm, err := m.db.VMs().GetByName(ctx, vmName)
    if err != nil {
        return fmt.Errorf("VM not found: %w", err)
    }

    // 3. Store mount
    mount := &VolumeMount{
        VolumeID:   vol.ID,
        VMName:     vmName,
        MountPoint: mountPoint,
        ReadOnly:   readOnly,
    }
    return m.db.VolumeMounts().Create(ctx, mount)
}

func (m *Manager) DetachVolume(ctx context.Context, vmName, volumeName string) error {
    vol, err := m.db.Volumes().GetByName(ctx, volumeName)
    if err != nil {
        return fmt.Errorf("volume not found: %w", err)
    }

    return m.db.VolumeMounts().DeleteByVMAndVolume(ctx, vmName, vol.ID)
}

func (m *Manager) BackupVolume(ctx context.Context, name string) error {
    vol, err := m.db.Volumes().GetByName(ctx, name)
    if err != nil {
        return fmt.Errorf("volume not found: %w", err)
    }

    backupPath := vol.HostPath + ".backup-" + fmt.Sprintf("%d", time.Now().Unix())

    // Snapshot + compress
    cmd := exec.Command("tar", "czf", backupPath, "-C", filepath.Dir(vol.HostPath), filepath.Base(vol.HostPath))
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("backup failed: %w", err)
    }

    vol.BackupPath = backupPath
    return m.db.Volumes().Update(ctx, vol)
}
```

**Checklist:**
- [ ] Create `volumes/manager.go`
- [ ] Implement `CreateVolume()` (dd + mkfs.ext4)
- [ ] Implement `DeleteVolume()` (with safety checks)
- [ ] Implement `AttachVolume()` / `DetachVolume()`
- [ ] Implement `BackupVolume()` (tar + gzip)
- [ ] Add error handling
- [ ] Write unit tests

#### Task 7.1.2: Add Database Schema
**File:** `volant/internal/server/db/sqlite/migrations/008_volumes.sql` (NEW)

**Migration:**
```sql
-- 008_volumes.sql
CREATE TABLE volumes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL,  -- ext4|squashfs|bind
    size_gb INTEGER NOT NULL,
    persistent BOOLEAN DEFAULT 1,
    host_path TEXT NOT NULL,
    backup_path TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE volume_mounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    volume_id INTEGER NOT NULL,
    vm_name TEXT NOT NULL,
    mount_point TEXT NOT NULL,
    read_only BOOLEAN DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (volume_id) REFERENCES volumes(id) ON DELETE CASCADE,
    UNIQUE(vm_name, mount_point)  -- One mount per path per VM
);

CREATE INDEX idx_volume_mounts_volume ON volume_mounts(volume_id);
CREATE INDEX idx_volume_mounts_vm ON volume_mounts(vm_name);
```

**Checklist:**
- [ ] Create migration file
- [ ] Add `volumes` table
- [ ] Add `volume_mounts` table
- [ ] Add indexes
- [ ] Test migration applies cleanly

#### Task 7.1.3: Add Repository Interfaces
**File:** `volant/internal/server/db/sqlite/volumes.go` (NEW)

**Implementation:**
```go
package sqlite

import (
    "context"
    "database/sql"
    "time"

    "github.com/volantvm/volant/internal/server/volumes"
)

type VolumeRepository struct {
    db *sql.DB
}

func (r *VolumeRepository) Create(ctx context.Context, vol *volumes.Volume) error {
    query := `INSERT INTO volumes (name, type, size_gb, persistent, host_path, backup_path, created_at, updated_at)
              VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

    now := time.Now()
    result, err := r.db.ExecContext(ctx, query,
        vol.Name, vol.Type, vol.SizeGB, vol.Persistent,
        vol.HostPath, vol.BackupPath, now, now)
    if err != nil {
        return err
    }

    id, _ := result.LastInsertId()
    vol.ID = id
    return nil
}

func (r *VolumeRepository) GetByName(ctx context.Context, name string) (*volumes.Volume, error) {
    query := `SELECT id, name, type, size_gb, persistent, host_path, backup_path, created_at, updated_at
              FROM volumes WHERE name = ?`

    var vol volumes.Volume
    err := r.db.QueryRowContext(ctx, query, name).Scan(
        &vol.ID, &vol.Name, &vol.Type, &vol.SizeGB, &vol.Persistent,
        &vol.HostPath, &vol.BackupPath, &vol.CreatedAt, &vol.UpdatedAt)
    if err != nil {
        return nil, err
    }
    return &vol, nil
}

func (r *VolumeRepository) List(ctx context.Context) ([]volumes.Volume, error) {
    query := `SELECT id, name, type, size_gb, persistent, host_path, backup_path, created_at, updated_at
              FROM volumes ORDER BY created_at DESC`

    rows, err := r.db.QueryContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var vols []volumes.Volume
    for rows.Next() {
        var vol volumes.Volume
        if err := rows.Scan(&vol.ID, &vol.Name, &vol.Type, &vol.SizeGB, &vol.Persistent,
            &vol.HostPath, &vol.BackupPath, &vol.CreatedAt, &vol.UpdatedAt); err != nil {
            return nil, err
        }
        vols = append(vols, vol)
    }
    return vols, nil
}

func (r *VolumeRepository) Delete(ctx context.Context, id int64) error {
    query := `DELETE FROM volumes WHERE id = ?`
    _, err := r.db.ExecContext(ctx, query, id)
    return err
}

func (r *VolumeRepository) Update(ctx context.Context, vol *volumes.Volume) error {
    query := `UPDATE volumes SET backup_path = ?, updated_at = ? WHERE id = ?`
    _, err := r.db.ExecContext(ctx, query, vol.BackupPath, time.Now(), vol.ID)
    return err
}

type VolumeMountRepository struct {
    db *sql.DB
}

func (r *VolumeMountRepository) Create(ctx context.Context, mount *volumes.VolumeMount) error {
    query := `INSERT INTO volume_mounts (volume_id, vm_name, mount_point, read_only, created_at)
              VALUES (?, ?, ?, ?, ?)`

    _, err := r.db.ExecContext(ctx, query,
        mount.VolumeID, mount.VMName, mount.MountPoint, mount.ReadOnly, time.Now())
    return err
}

func (r *VolumeMountRepository) ListByVolume(ctx context.Context, volumeID int64) ([]volumes.VolumeMount, error) {
    query := `SELECT volume_id, vm_name, mount_point, read_only FROM volume_mounts WHERE volume_id = ?`

    rows, err := r.db.QueryContext(ctx, query, volumeID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var mounts []volumes.VolumeMount
    for rows.Next() {
        var m volumes.VolumeMount
        if err := rows.Scan(&m.VolumeID, &m.VMName, &m.MountPoint, &m.ReadOnly); err != nil {
            return nil, err
        }
        mounts = append(mounts, m)
    }
    return mounts, nil
}

func (r *VolumeMountRepository) ListByVM(ctx context.Context, vmName string) ([]volumes.VolumeMount, error) {
    query := `SELECT volume_id, vm_name, mount_point, read_only FROM volume_mounts WHERE vm_name = ?`

    rows, err := r.db.QueryContext(ctx, query, vmName)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var mounts []volumes.VolumeMount
    for rows.Next() {
        var m volumes.VolumeMount
        if err := rows.Scan(&m.VolumeID, &m.VMName, &m.MountPoint, &m.ReadOnly); err != nil {
            return nil, err
        }
        mounts = append(mounts, m)
    }
    return mounts, nil
}

func (r *VolumeMountRepository) DeleteByVMAndVolume(ctx context.Context, vmName string, volumeID int64) error {
    query := `DELETE FROM volume_mounts WHERE vm_name = ? AND volume_id = ?`
    _, err := r.db.ExecContext(ctx, query, vmName, volumeID)
    return err
}
```

**Checklist:**
- [ ] Create `volumes.go` with repositories
- [ ] Implement all CRUD methods
- [ ] Add to `Queries` interface
- [ ] Test all database operations

### 7.2 Cloud Hypervisor Integration

#### Task 7.2.1: Add Volume Support to Launcher
**File:** `volant/internal/server/orchestrator/cloudhypervisor/launcher.go`

**Update launch config:**
```go
type LaunchConfig struct {
    // ... existing fields ...

    // NEW: Volumes to attach
    Volumes []VolumeConfig
}

type VolumeConfig struct {
    HostPath   string
    MountPoint string  // Not used by CH, but stored for manifest
    ReadOnly   bool
}

func (l *Launcher) Launch(ctx context.Context, cfg LaunchConfig) (int, error) {
    args := []string{
        "--kernel", cfg.Kernel,
        "--cmdline", cfg.Cmdline,
        // ... existing args ...
    }

    // Add rootfs
    if cfg.Rootfs != nil {
        args = append(args, "--disk", fmt.Sprintf("path=%s,readonly=%v", cfg.Rootfs.Path, cfg.Rootfs.ReadOnly))
    }

    // NEW: Add volumes
    for _, vol := range cfg.Volumes {
        args = append(args, "--disk", fmt.Sprintf("path=%s,readonly=%v", vol.HostPath, vol.ReadOnly))
    }

    // ... rest of launch logic ...
}
```

**Checklist:**
- [ ] Add `Volumes` field to `LaunchConfig`
- [ ] Generate `--disk` args for each volume
- [ ] Handle read-only flag correctly
- [ ] Test volumes are attached

#### Task 7.2.2: Update Orchestrator to Attach Volumes
**File:** `volant/internal/server/orchestrator/orchestrator.go`

**Update `CreateVM`:**
```go
func (o *Orchestrator) CreateVM(ctx context.Context, req CreateVMRequest) (*VM, error) {
    // ... existing logic (config merge, IP allocation, etc.) ...

    // NEW: Load volume mounts
    volumeMounts, err := o.volumeMgr.GetMountsForRequest(ctx, req.Volumes)
    if err != nil {
        return nil, fmt.Errorf("failed to load volumes: %w", err)
    }

    // NEW: Add volumes to launch config
    var volumeConfigs []cloudhypervisor.VolumeConfig
    for _, mount := range volumeMounts {
        vol, err := o.db.Volumes().GetByName(ctx, mount.VolumeName)
        if err != nil {
            return nil, fmt.Errorf("volume not found: %s", mount.VolumeName)
        }
        volumeConfigs = append(volumeConfigs, cloudhypervisor.VolumeConfig{
            HostPath:   vol.HostPath,
            MountPoint: mount.MountPoint,
            ReadOnly:   mount.ReadOnly,
        })
    }

    // Launch with volumes
    pid := o.launcher.Launch(ctx, cloudhypervisor.LaunchConfig{
        // ... existing fields ...
        Volumes: volumeConfigs,
    })

    // ... rest of CreateVM ...
}
```

**Checklist:**
- [ ] Load volume mounts from request
- [ ] Lookup volume paths
- [ ] Pass volumes to launcher
- [ ] Test volumes are accessible in VM

#### Task 7.2.3: Auto-Mount Volumes in Guest
**File:** `volant/internal/agent/app/app.go`

**Add volume mounting:**
```go
func (a *Agent) Start(ctx context.Context) error {
    // ... existing: decode manifest, mount filesystems ...

    // NEW: Auto-mount additional volumes
    if err := a.mountVolumes(ctx); err != nil {
        return fmt.Errorf("failed to mount volumes: %w", err)
    }

    // ... continue with workload launch ...
}

func (a *Agent) mountVolumes(ctx context.Context) error {
    // Scan for additional disks beyond rootfs
    // Cloud Hypervisor adds them as /dev/vdb, /dev/vdc, etc.

    for i := 1; i < 10; i++ {  // Support up to 9 volumes
        devPath := fmt.Sprintf("/dev/vd%c", 'a'+i)
        if _, err := os.Stat(devPath); os.IsNotExist(err) {
            break  // No more volumes
        }

        // Mount point from manifest (passed via kernel cmdline)
        mountPoint := a.getVolumeMountPoint(i)
        if mountPoint == "" {
            continue  // No mount point specified
        }

        // Create mount point
        os.MkdirAll(mountPoint, 0755)

        // Mount
        cmd := exec.Command("mount", devPath, mountPoint)
        if err := cmd.Run(); err != nil {
            return fmt.Errorf("failed to mount %s: %w", devPath, err)
        }
    }

    return nil
}
```

**Checklist:**
- [ ] Implement volume auto-mounting
- [ ] Create mount points if missing
- [ ] Handle mount errors gracefully
- [ ] Test volumes are mounted correctly

### 7.3 CLI Commands

#### Task 7.3.1: Add Volume Commands
**File:** `volant/internal/cli/standard/volumes.go` (NEW)

**Implementation:**
```go
package standard

import (
    "github.com/spf13/cobra"
)

func volumesCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "volumes",
        Short: "Manage persistent volumes",
    }

    cmd.AddCommand(volumesCreateCmd())
    cmd.AddCommand(volumesListCmd())
    cmd.AddCommand(volumesRemoveCmd())
    cmd.AddCommand(volumesBackupCmd())

    return cmd
}

func volumesCreateCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "create [name]",
        Short: "Create a new volume",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            name := args[0]
            size, _ := cmd.Flags().GetInt("size")
            volType, _ := cmd.Flags().GetString("type")

            return apiClient.CreateVolume(cmd.Context(), name, size, volType)
        },
    }

    cmd.Flags().Int("size", 10, "Volume size in GB")
    cmd.Flags().String("type", "ext4", "Volume type (ext4|bind)")

    return cmd
}

func volumesListCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "list",
        Short: "List all volumes",
        RunE: func(cmd *cobra.Command, args []string) error {
            volumes, err := apiClient.ListVolumes(cmd.Context())
            if err != nil {
                return err
            }

            // Print table
            fmt.Printf("%-20s %-10s %-10s %-20s\n", "NAME", "TYPE", "SIZE", "ATTACHED TO")
            for _, vol := range volumes {
                attachedTo := "-"
                if len(vol.Mounts) > 0 {
                    attachedTo = vol.Mounts[0].VMName
                }
                fmt.Printf("%-20s %-10s %-10s %-20s\n", vol.Name, vol.Type, fmt.Sprintf("%dG", vol.SizeGB), attachedTo)
            }
            return nil
        },
    }
}

func volumesRemoveCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "rm [name]",
        Short: "Remove a volume",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return apiClient.DeleteVolume(cmd.Context(), args[0])
        },
    }
}

func volumesBackupCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "backup [name]",
        Short: "Backup a volume",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return apiClient.BackupVolume(cmd.Context(), args[0])
        },
    }
}
```

**Checklist:**
- [ ] Create `volumes.go` with CLI commands
- [ ] Implement `create`, `list`, `rm`, `backup` commands
- [ ] Add flags (size, type)
- [ ] Add to root command
- [ ] Update help text

#### Task 7.3.2: Update VM Create Command
**File:** `volant/internal/cli/standard/vms.go`

**Add volume flag:**
```go
func vmsCreateCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "create [name]",
        Short: "Create a new VM",
        // ... existing code ...
    }

    // ... existing flags ...

    // NEW: Volume mounting
    cmd.Flags().StringArray("volume", []string{}, "Mount volume (format: VOL:PATH or VOL:PATH:ro)")

    return cmd
}
```

**Parse volume flag:**
```go
func parseVolumes(volumeFlags []string) ([]VolumeMount, error) {
    var mounts []VolumeMount
    for _, v := range volumeFlags {
        parts := strings.Split(v, ":")
        if len(parts) < 2 {
            return nil, fmt.Errorf("invalid volume format: %s (expected VOL:PATH or VOL:PATH:ro)", v)
        }

        mount := VolumeMount{
            VolumeName: parts[0],
            MountPoint: parts[1],
            ReadOnly:   len(parts) == 3 && parts[2] == "ro",
        }
        mounts = append(mounts, mount)
    }
    return mounts, nil
}
```

**Example usage:**
```bash
# Read-write mount
volar vms create postgres --image postgres-db \
  --volume pgdata:/var/lib/postgresql/data

# Read-only mount
volar vms create web --image nginx \
  --volume config:/etc/nginx/conf.d:ro

# Multiple volumes
volar vms create app --image myapp \
  --volume data:/app/data \
  --volume logs:/var/log:ro
```

**Checklist:**
- [ ] Add `--volume` flag
- [ ] Implement volume parsing
- [ ] Pass volumes to API
- [ ] Update help text with examples

### 7.4 REST API

#### Task 7.4.1: Add Volume Endpoints
**File:** `volant/internal/server/httpapi/volumes.go` (NEW)

**Implementation:**
```go
package httpapi

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/volantvm/volant/internal/server/volumes"
)

func (h *HTTPServer) registerVolumeRoutes(r *gin.RouterGroup) {
    v := r.Group("/volumes")
    {
        v.POST("", h.createVolume)
        v.GET("", h.listVolumes)
        v.GET("/:name", h.getVolume)
        v.DELETE("/:name", h.deleteVolume)
        v.POST("/:name/backup", h.backupVolume)
    }
}

type CreateVolumeRequest struct {
    Name       string `json:"name" binding:"required"`
    Type       string `json:"type" binding:"required"`  // ext4|bind
    SizeGB     int    `json:"size_gb" binding:"required,min=1"`
    Persistent bool   `json:"persistent"`
}

func (h *HTTPServer) createVolume(c *gin.Context) {
    var req CreateVolumeRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    vol := &volumes.Volume{
        Name:       req.Name,
        Type:       volumes.VolumeType(req.Type),
        SizeGB:     req.SizeGB,
        Persistent: req.Persistent,
    }

    if err := h.volumeMgr.CreateVolume(c.Request.Context(), vol); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, vol)
}

func (h *HTTPServer) listVolumes(c *gin.Context) {
    vols, err := h.db.Volumes().List(c.Request.Context())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, vols)
}

func (h *HTTPServer) getVolume(c *gin.Context) {
    name := c.Param("name")
    vol, err := h.db.Volumes().GetByName(c.Request.Context(), name)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
        return
    }

    c.JSON(http.StatusOK, vol)
}

func (h *HTTPServer) deleteVolume(c *gin.Context) {
    name := c.Param("name")
    if err := h.volumeMgr.DeleteVolume(c.Request.Context(), name); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "volume deleted"})
}

func (h *HTTPServer) backupVolume(c *gin.Context) {
    name := c.Param("name")
    if err := h.volumeMgr.BackupVolume(c.Request.Context(), name); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "volume backed up"})
}
```

**Checklist:**
- [ ] Create `volumes.go` with REST handlers
- [ ] Implement all CRUD endpoints
- [ ] Add request validation
- [ ] Add to router
- [ ] Test all endpoints

#### Task 7.4.2: Update CreateVM Endpoint
**File:** `volant/internal/server/httpapi/httpapi.go`

**Update request:**
```go
type CreateVMRequest struct {
    // ... existing fields ...

    // NEW: Volumes to mount
    Volumes []VolumeMountRequest `json:"volumes,omitempty"`
}

type VolumeMountRequest struct {
    VolumeName string `json:"volume"`
    MountPoint string `json:"mount_point"`
    ReadOnly   bool   `json:"read_only"`
}
```

**Pass to orchestrator:**
```go
func (h *HTTPServer) createVM(c *gin.Context) {
    var req CreateVMRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    vm, err := h.orchestrator.CreateVM(c.Request.Context(), orchestrator.CreateVMRequest{
        Name:      req.Name,
        ImageName: req.ImageName,
        Overrides: &orchestrator.VMConfigOverrides{
            CPUCores: req.CPUCores,
            MemoryMB: req.MemoryMB,
            Env:      req.Env,
            Ports:    req.Ports,
        },
        Volumes: req.Volumes,  // NEW
    })
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, vm)
}
```

**Checklist:**
- [ ] Update `CreateVMRequest`
- [ ] Pass volumes to orchestrator
- [ ] Validate volume references
- [ ] Test VM creation with volumes

### 7.5 Testing

#### Task 7.5.1: Unit Tests
**File:** `volant/internal/server/volumes/manager_test.go` (NEW)

**Tests:**
```go
func TestCreateVolume(t *testing.T) {
    mgr := NewManager("/tmp/test-volumes", mockDB)

    vol := &Volume{
        Name:   "test-vol",
        Type:   VolumeTypeExt4,
        SizeGB: 1,
    }

    err := mgr.CreateVolume(context.Background(), vol)
    assert.NoError(t, err)
    assert.FileExists(t, vol.HostPath)
}

func TestDeleteVolume(t *testing.T) {
    // Test deletion
    // Test attached volume protection
}

func TestAttachVolume(t *testing.T) {
    // Test attach/detach
}
```

**Checklist:**
- [ ] Write tests for volume creation
- [ ] Test deletion (with safety checks)
- [ ] Test attach/detach
- [ ] Test backup/restore
- [ ] All tests pass

#### Task 7.5.2: Integration Tests
**Script:** `test-volumes.sh`

```bash
#!/bin/bash
set -e

echo "=== Testing Volumes ==="

# 1. Create volume
volar volumes create pgdata --size 10 --type ext4

# Verify creation
volar volumes list | grep pgdata || { echo "ERROR: Volume not created"; exit 1; }

# 2. Create VM with volume
volar vms create postgres --image postgres-db \
  --volume pgdata:/var/lib/postgresql/data

# 3. Write data to volume
volar exec postgres -- psql -U postgres -c "CREATE DATABASE testdb;"

# 4. Restart VM
volar vms restart postgres

# 5. Verify data persists
volar exec postgres -- psql -U postgres -l | grep testdb || { echo "ERROR: Data not persisted"; exit 1; }

# 6. Backup volume
volar volumes backup pgdata

# 7. Cleanup
volar vms destroy postgres
volar volumes rm pgdata

echo "=== Volumes Test PASSED ==="
```

**Checklist:**
- [ ] Run integration test
- [ ] Verify data persistence
- [ ] Test backup/restore
- [ ] Test read-only mounts

### 7.6 Documentation

#### Task 7.6.1: Document Volume Management
**File:** `volant/docs/3_guides/volumes.md` (NEW)

**Content:** (See previous sections for full examples)

**Checklist:**
- [ ] Create `volumes.md`
- [ ] Document volume types
- [ ] Show creation examples
- [ ] Explain persistence
- [ ] Add backup/restore section

### 7.7 Success Criteria

**Track B Complete When:**
- [ ] Volumes can be created (ext4, bind)
- [ ] Volumes attach to VMs correctly
- [ ] Data persists across VM restarts
- [ ] Backups work
- [ ] CLI commands functional
- [ ] REST API endpoints work
- [ ] All tests pass
- [ ] Documentation complete
- [ ] **Branch merged to main** ← Unblocks Track D

---

# 8. PHASE 1 TRACK C: Drift L4 Completion

## Overview

**Duration:** 5-6 days
**Worktree:** `volant-track-c`
**Branch:** `phase-1-track-c-drift`
**Dependencies:** Phase 0 ✓
**Conflicts:** None (isolated subsystem)

## What This Enables

**Before:**
```bash
# Per-VM vsock proxy (userspace)
Client → :8080 → vsock proxy → VM
Latency: ~100μs
```

**After:**
```bash
# eBPF L4 routing (kernel space)
Client → :8080 → [eBPF] → VM
Latency: ~10μs (10x faster)
```

## Measurable Tasks

### 8.1 eBPF Dataplane

#### Task 8.1.1: Complete eBPF Programs
**Files:** `volant/internal/drift/bpf/*.c`

**Programs needed:**
1. `drift_l4_ingress.c` - Inbound packet processing
2. `drift_l4_egress.c` - Outbound packet processing

**Ingress program:**
```c
// drift_l4_ingress.c
#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <bpf/bpf_helpers.h>

struct route_key {
    __u16 port;
    __u8 protocol;  // IPPROTO_TCP or IPPROTO_UDP
};

struct route_value {
    __u32 dest_ip;
    __u16 dest_port;
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, struct route_key);
    __type(value, struct route_value);
    __uint(max_entries, 1024);
} route_map SEC(".maps");

SEC("tc/ingress")
int drift_ingress(struct __sk_buff *skb) {
    void *data_end = (void *)(long)skb->data_end;
    void *data = (void *)(long)skb->data;

    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return TC_ACT_OK;

    if (eth->h_proto != htons(ETH_P_IP))
        return TC_ACT_OK;

    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return TC_ACT_OK;

    struct route_key key = {0};
    key.protocol = ip->protocol;

    if (ip->protocol == IPPROTO_TCP) {
        struct tcphdr *tcp = (void *)ip + sizeof(*ip);
        if ((void *)(tcp + 1) > data_end)
            return TC_ACT_OK;
        key.port = ntohs(tcp->dest);
    } else if (ip->protocol == IPPROTO_UDP) {
        struct udphdr *udp = (void *)ip + sizeof(*ip);
        if ((void *)(udp + 1) > data_end)
            return TC_ACT_OK;
        key.port = ntohs(udp->dest);
    } else {
        return TC_ACT_OK;
    }

    // Lookup route
    struct route_value *route = bpf_map_lookup_elem(&route_map, &key);
    if (!route)
        return TC_ACT_OK;

    // Rewrite destination IP and port
    ip->daddr = route->dest_ip;

    if (ip->protocol == IPPROTO_TCP) {
        struct tcphdr *tcp = (void *)ip + sizeof(*ip);
        tcp->dest = htons(route->dest_port);
    } else {
        struct udphdr *udp = (void *)ip + sizeof(*ip);
        udp->dest = htons(route->dest_port);
    }

    // Recalculate checksums
    // TODO: Implement checksum updates

    return TC_ACT_OK;
}

char _license[] SEC("license") = "GPL";
```

**Egress program:**
```c
// drift_l4_egress.c
// Mirror of ingress, but for return packets
// Rewrites source IP/port to original values
```

**Checklist:**
- [ ] Complete ingress eBPF program
- [ ] Complete egress eBPF program
- [ ] Implement checksum recalculation
- [ ] Add connection tracking
- [ ] Compile eBPF to .o files
- [ ] Test eBPF loading

#### Task 8.1.2: Implement eBPF Dataplane Manager
**File:** `volant/internal/drift/dataplane/manager_linux.go`

**Implementation:**
```go
package dataplane

import (
    "fmt"

    "github.com/cilium/ebpf"
    "github.com/cilium/ebpf/link"
)

type Manager struct {
    objs    *eBPFObjects
    ingress link.Link
    egress  link.Link
}

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -type route_key,route_value eBPF ../../bpf/drift_l4_ingress.c -- -I../../bpf

func New() (*Manager, error) {
    // Load eBPF objects
    objs := &eBPFObjects{}
    if err := LoadeBPFObjects(objs, nil); err != nil {
        return nil, fmt.Errorf("failed to load eBPF objects: %w", err)
    }

    return &Manager{objs: objs}, nil
}

func (m *Manager) AttachToInterface(ifname string) error {
    // Get interface
    iface, err := net.InterfaceByName(ifname)
    if err != nil {
        return fmt.Errorf("interface not found: %w", err)
    }

    // Attach TC ingress program
    m.ingress, err = link.AttachTCX(link.TCXOptions{
        Program:   m.objs.DriftIngress,
        Attach:    ebpf.AttachTCXIngress,
        Interface: iface.Index,
    })
    if err != nil {
        return fmt.Errorf("failed to attach ingress: %w", err)
    }

    // Attach TC egress program
    m.egress, err = link.AttachTCX(link.TCXOptions{
        Program:   m.objs.DriftEgress,
        Attach:    ebpf.AttachTCXEgress,
        Interface: iface.Index,
    })
    if err != nil {
        m.ingress.Close()
        return fmt.Errorf("failed to attach egress: %w", err)
    }

    return nil
}

func (m *Manager) AddRoute(hostPort uint16, protocol string, destIP string, destPort uint16) error {
    key := RouteKey{
        Port:     hostPort,
        Protocol: protocolToInt(protocol),
    }

    value := RouteValue{
        DestIP:   ipToUint32(destIP),
        DestPort: destPort,
    }

    return m.objs.RouteMap.Put(&key, &value)
}

func (m *Manager) DeleteRoute(hostPort uint16, protocol string) error {
    key := RouteKey{
        Port:     hostPort,
        Protocol: protocolToInt(protocol),
    }

    return m.objs.RouteMap.Delete(&key)
}

func (m *Manager) Close() error {
    if m.ingress != nil {
        m.ingress.Close()
    }
    if m.egress != nil {
        m.egress.Close()
    }
    if m.objs != nil {
        m.objs.Close()
    }
    return nil
}
```

**Checklist:**
- [ ] Implement dataplane manager
- [ ] Load eBPF programs
- [ ] Attach to network interface
- [ ] Implement route add/delete
- [ ] Handle errors gracefully
- [ ] Write tests

### 8.2 Route Management

#### Task 8.2.1: Complete Route Controller
**File:** `volant/internal/drift/controller/controller.go`

**Current state:** Basic structure exists

**Enhancements needed:**
```go
type Controller struct {
    dataplane Dataplane
    routes    routes.Store
    mu        sync.RWMutex
}

func (c *Controller) AddRoute(ctx context.Context, route *routes.Route) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    // 1. Check for conflicts
    existing, _ := c.routes.GetByPort(ctx, route.HostPort, route.Protocol)
    if existing != nil {
        return fmt.Errorf("port %d/%s already in use", route.HostPort, route.Protocol)
    }

    // 2. Add to eBPF dataplane
    if err := c.dataplane.AddRoute(route.HostPort, route.Protocol, route.Backend.IP, route.Backend.Port); err != nil {
        return fmt.Errorf("failed to add route to dataplane: %w", err)
    }

    // 3. Store persistently
    if err := c.routes.Create(ctx, route); err != nil {
        // Rollback dataplane
        c.dataplane.DeleteRoute(route.HostPort, route.Protocol)
        return fmt.Errorf("failed to store route: %w", err)
    }

    return nil
}

func (c *Controller) DeleteRoute(ctx context.Context, hostPort uint16, protocol string) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    // 1. Remove from dataplane
    if err := c.dataplane.DeleteRoute(hostPort, protocol); err != nil {
        return fmt.Errorf("failed to remove route from dataplane: %w", err)
    }

    // 2. Remove from storage
    if err := c.routes.Delete(ctx, hostPort, protocol); err != nil {
        // Log but don't fail (dataplane is authoritative)
        log.Errorf("failed to delete route from storage: %v", err)
    }

    return nil
}

func (c *Controller) HealthCheck(ctx context.Context) error {
    // Check dataplane is running
    if err := c.dataplane.Ping(); err != nil {
        return fmt.Errorf("dataplane unhealthy: %w", err)
    }

    // Check route count matches
    dataplaneCount := c.dataplane.RouteCount()
    storageCount := c.routes.Count(ctx)
    if dataplaneCount != storageCount {
        return fmt.Errorf("route count mismatch: dataplane=%d storage=%d", dataplaneCount, storageCount)
    }

    return nil
}
```

**Checklist:**
- [ ] Add conflict detection
- [ ] Implement atomic add/delete
- [ ] Add health checks
- [ ] Handle rollbacks
- [ ] Test edge cases

### 8.3 Integration with Volant

#### Task 8.3.1: Update Drift Client
**File:** `volant/internal/server/driftclient/client.go`

**Enhancements:**
```go
type Client struct {
    endpoint string
    apiKey   string
    client   *http.Client
    fallback bool  // Use vsock proxy if Drift unavailable
}

func (c *Client) RegisterRoutes(ctx context.Context, vmName string, routes []Route) error {
    // Try Drift first
    if !c.fallback {
        err := c.registerViaHTTP(ctx, routes)
        if err == nil {
            return nil
        }

        log.Warnf("Drift unavailable, falling back to vsock proxy: %v", err)
        c.fallback = true
    }

    // Fallback to vsock proxy
    return c.registerViaVsockProxy(ctx, vmName, routes)
}

func (c *Client) registerViaHTTP(ctx context.Context, routes []Route) error {
    for _, route := range routes {
        req, _ := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/routes", jsonBody(route))
        req.Header.Set("Authorization", "Bearer "+c.apiKey)

        resp, err := c.client.Do(req)
        if err != nil {
            return err
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusCreated {
            return fmt.Errorf("drift returned %d", resp.StatusCode)
        }
    }
    return nil
}

func (c *Client) registerViaVsockProxy(ctx context.Context, vmName string, routes []Route) error {
    // Start vsock proxy processes (existing implementation)
    for _, route := range routes {
        go vsockproxy.Start(ctx, route.HostPort, vmName, route.GuestPort)
    }
    return nil
}
```

**Checklist:**
- [ ] Implement fallback logic
- [ ] Add health checking
- [ ] Handle Drift downtime gracefully
- [ ] Test failover scenario

### 8.4 Monitoring & Metrics

#### Task 8.4.1: Add Metrics Endpoint
**File:** `volant/internal/drift/httpapi/metrics.go` (NEW)

**Implementation:**
```go
package httpapi

import (
    "github.com/gin-gonic/gin"
)

type Metrics struct {
    TotalRoutes    int            `json:"total_routes"`
    PacketsRouted  uint64         `json:"packets_routed"`
    BytesRouted    uint64         `json:"bytes_routed"`
    DroppedPackets uint64         `json:"dropped_packets"`
    RoutesByProto  map[string]int `json:"routes_by_protocol"`
}

func (h *HTTPServer) getMetrics(c *gin.Context) {
    metrics, err := h.controller.GetMetrics(c.Request.Context())
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, metrics)
}
```

**Checklist:**
- [ ] Implement metrics collection
- [ ] Add packet/byte counters
- [ ] Track drop rates
- [ ] Expose via HTTP endpoint

### 8.5 Testing

#### Task 8.5.1: Performance Benchmarks
**Script:** `bench-drift.sh`

```bash
#!/bin/bash
set -e

echo "=== Drift Performance Benchmark ==="

# 1. Start Drift
driftd --interface vbr0 &
DRIFT_PID=$!
sleep 2

# 2. Create VM with Drift routing
VOLANT_DRIFT_ENDPOINT=http://localhost:9090 volantd &
VOLANTD_PID=$!
sleep 2

volar vms create web --image nginx --port 8080:80

# 3. Benchmark Drift routing
echo "Benchmarking Drift L4..."
ab -n 10000 -c 100 http://localhost:8080/ > drift-results.txt

# 4. Disable Drift, enable vsock proxy
kill $DRIFT_PID
volar vms restart web

# 5. Benchmark vsock proxy
echo "Benchmarking vsock proxy..."
ab -n 10000 -c 100 http://localhost:8080/ > vsock-results.txt

# 6. Compare
DRIFT_RPS=$(grep "Requests per second" drift-results.txt | awk '{print $4}')
VSOCK_RPS=$(grep "Requests per second" vsock-results.txt | awk '{print $4}')

echo "Drift: $DRIFT_RPS req/s"
echo "Vsock: $VSOCK_RPS req/s"
echo "Speedup: $(echo "scale=2; $DRIFT_RPS / $VSOCK_RPS" | bc)x"

# Cleanup
kill $VOLANTD_PID
volar vms destroy web
```

**Checklist:**
- [ ] Run benchmark
- [ ] Compare Drift vs vsock proxy
- [ ] Verify 2-10x speedup
- [ ] Document results

#### Task 8.5.2: Failover Test
**Script:** `test-drift-failover.sh`

```bash
#!/bin/bash
set -e

echo "=== Drift Failover Test ==="

# 1. Start without Drift
volantd &
VOLANTD_PID=$!
sleep 2

# Create VM (should use vsock proxy)
volar vms create web --image nginx --port 8080:80

# Test connectivity
curl http://localhost:8080/

# 2. Start Drift mid-flight
driftd --interface vbr0 &
DRIFT_PID=$!
sleep 2

# Create another VM (should use Drift)
volar vms create api --image myapp --port 8081:8080

# Test connectivity
curl http://localhost:8081/health

# 3. Kill Drift
kill $DRIFT_PID

# Create another VM (should fall back to vsock)
volar vms create db --image postgres-db --port 5432:5432

# Test connectivity
psql -h localhost -p 5432 -U postgres -c "SELECT 1"

# Cleanup
kill $VOLANTD_PID

echo "=== Drift Failover Test PASSED ==="
```

**Checklist:**
- [ ] Test Drift unavailable at start
- [ ] Test Drift appears mid-flight
- [ ] Test Drift disappears mid-flight
- [ ] Verify graceful fallback

### 8.6 Documentation

#### Task 8.6.1: Document Drift L4
**File:** `volant/docs/3_guides/drift-l4-routing.md` (NEW)

**Content:**
```markdown
# Drift L4 Switch

## Overview

Drift is an optional eBPF-based L4 packet router that provides high-performance port forwarding.

## Why Drift?

**Without Drift (vsock proxy):**
- Userspace process per port mapping
- Context switches for every packet
- Latency: ~100μs

**With Drift (eBPF):**
- Kernel-space packet rewriting
- Zero context switches
- Latency: ~10μs (10x faster)

## Configuration

```bash
# Enable Drift
VOLANT_DRIFT_ENDPOINT=http://localhost:9090 volantd

# Start Drift daemon
driftd --interface vbr0
```

## Performance

Benchmark results:
- Throughput: 2-10x higher than vsock proxy
- Latency: 10x lower
- CPU usage: 50% lower

## Failover

Drift is optional. If Drift is unavailable, Volant automatically falls back to vsock proxy.
```

**Checklist:**
- [ ] Create `drift-l4-routing.md`
- [ ] Explain benefits
- [ ] Document configuration
- [ ] Show performance numbers
- [ ] Explain failover behavior

### 8.7 Success Criteria

**Track C Complete When:**
- [ ] eBPF dataplane fully functional
- [ ] Routes add/delete correctly
- [ ] Performance is 2-10x better than vsock proxy
- [ ] Failover to vsock proxy works
- [ ] Health checks implemented
- [ ] Metrics endpoint functional
- [ ] All tests pass
- [ ] Documentation complete
- [ ] **Branch merged to main**

---


# 9. PHASE 1 TRACK F: DOCKER COMPOSE CONVERTER

**Branch:** `feature/compose-converter`  
**Duration:** Week 2-3 (parallel with A/B/C)  
**Dependencies:** None (fully independent)  
**Owner:** Track F Team

## 9.1 Overview

Create a tool to convert `docker-compose.yml` files into Volant stack manifests (`volant.yaml`). This provides a migration path for existing Docker Compose users.

**Key Insight:** Docker Compose is container-centric. Volant is VM-centric. The converter must bridge this impedance mismatch.

## 9.2 Task 1: Docker Compose Parser

**File:** `volant/cmd/volant-compose/parser.go`

```go
package main

import (
    "fmt"
    "gopkg.in/yaml.v3"
    "os"
)

// DockerCompose represents a docker-compose.yml file
type DockerCompose struct {
    Version  string                     `yaml:"version"`
    Services map[string]ComposeService  `yaml:"services"`
    Networks map[string]ComposeNetwork  `yaml:"networks,omitempty"`
    Volumes  map[string]ComposeVolume   `yaml:"volumes,omitempty"`
}

type ComposeService struct {
    Image       string            `yaml:"image"`
    Build       *ComposeBuild     `yaml:"build,omitempty"`
    Command     []string          `yaml:"command,omitempty"`
    Entrypoint  []string          `yaml:"entrypoint,omitempty"`
    Environment map[string]string `yaml:"environment,omitempty"`
    Ports       []string          `yaml:"ports,omitempty"`
    Volumes     []string          `yaml:"volumes,omitempty"`
    Networks    []string          `yaml:"networks,omitempty"`
    DependsOn   []string          `yaml:"depends_on,omitempty"`
    Deploy      *ComposeDeploy    `yaml:"deploy,omitempty"`
    Restart     string            `yaml:"restart,omitempty"`
}

type ComposeBuild struct {
    Context    string            `yaml:"context"`
    Dockerfile string            `yaml:"dockerfile,omitempty"`
    Args       map[string]string `yaml:"args,omitempty"`
}

type ComposeDeploy struct {
    Replicas  int                `yaml:"replicas,omitempty"`
    Resources *ComposeResources  `yaml:"resources,omitempty"`
}

type ComposeResources struct {
    Limits       *ComposeResourceLimit `yaml:"limits,omitempty"`
    Reservations *ComposeResourceLimit `yaml:"reservations,omitempty"`
}

type ComposeResourceLimit struct {
    CPUs   string `yaml:"cpus,omitempty"`
    Memory string `yaml:"memory,omitempty"`
}

type ComposeNetwork struct {
    Driver string `yaml:"driver,omitempty"`
}

type ComposeVolume struct {
    Driver string `yaml:"driver,omitempty"`
}

func ParseDockerCompose(path string) (*DockerCompose, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read file: %w", err)
    }

    var compose DockerCompose
    if err := yaml.Unmarshal(data, &compose); err != nil {
        return nil, fmt.Errorf("parse yaml: %w", err)
    }

    return &compose, nil
}
```

**Test:**

```bash
# Test file: testdata/docker-compose.yml
cat > testdata/docker-compose.yml << 'EOF'
version: '3.8'

services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
    environment:
      NGINX_PORT: 80
    volumes:
      - web-data:/usr/share/nginx/html
    depends_on:
      - api

  api:
    build:
      context: ./api
      dockerfile: Dockerfile
    command: ["./server", "--port", "3000"]
    environment:
      DATABASE_URL: postgres://db:5432/myapp
    ports:
      - "3000:3000"
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: '2.0'
          memory: 1G

  db:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: secret
    volumes:
      - db-data:/var/lib/postgresql/data

volumes:
  web-data:
  db-data:
EOF

# Test parser
go run cmd/volant-compose/parser.go testdata/docker-compose.yml
```

**Checklist:**
- [ ] Implement `DockerCompose` struct
- [ ] Support Compose v2 and v3 formats
- [ ] Parse services, volumes, networks
- [ ] Handle `build` directive
- [ ] Handle `deploy.replicas`
- [ ] Parse resource limits
- [ ] Unit tests for parser

## 9.3 Task 2: Volant Stack Generator

**File:** `volant/cmd/volant-compose/generator.go`

```go
package main

import (
    "fmt"
    "strconv"
    "strings"
)

// VolantStack represents a volant.yaml file
type VolantStack struct {
    Stack    StackMetadata     `yaml:"stack"`
    Services map[string]VMSpec `yaml:"services"`
    Volumes  map[string]Volume `yaml:"volumes,omitempty"`
}

type StackMetadata struct {
    Name    string `yaml:"name"`
    Version string `yaml:"version"`
}

type VMSpec struct {
    Image       string            `yaml:"image"`
    Replicas    int               `yaml:"replicas,omitempty"`
    CPU         int               `yaml:"cpu,omitempty"`
    Memory      int               `yaml:"memory,omitempty"`
    Environment map[string]string `yaml:"environment,omitempty"`
    Ports       []PortMapping     `yaml:"ports,omitempty"`
    Volumes     []VolumeMount     `yaml:"volumes,omitempty"`
    DependsOn   []string          `yaml:"depends_on,omitempty"`
}

type PortMapping struct {
    Host  int    `yaml:"host"`
    Guest int    `yaml:"guest"`
    Proto string `yaml:"proto"`
}

type VolumeMount struct {
    Volume string `yaml:"volume"`
    Path   string `yaml:"path"`
}

type Volume struct {
    Type   string `yaml:"type"`
    SizeGB int    `yaml:"size_gb,omitempty"`
}

func ConvertToVolantStack(compose *DockerCompose, stackName string) (*VolantStack, error) {
    stack := &VolantStack{
        Stack: StackMetadata{
            Name:    stackName,
            Version: "1.0",
        },
        Services: make(map[string]VMSpec),
        Volumes:  make(map[string]Volume),
    }

    // Convert services
    for name, svc := range compose.Services {
        vmSpec := VMSpec{
            Image:       convertImage(svc),
            Replicas:    getReplicas(svc),
            CPU:         getCPU(svc),
            Memory:      getMemory(svc),
            Environment: svc.Environment,
            DependsOn:   svc.DependsOn,
        }

        // Convert ports
        for _, portStr := range svc.Ports {
            port, err := parsePort(portStr)
            if err != nil {
                return nil, fmt.Errorf("service %s: %w", name, err)
            }
            vmSpec.Ports = append(vmSpec.Ports, port)
        }

        // Convert volumes
        for _, volStr := range svc.Volumes {
            mount, err := parseVolume(volStr)
            if err != nil {
                return nil, fmt.Errorf("service %s: %w", name, err)
            }
            vmSpec.Volumes = append(vmSpec.Volumes, mount)
        }

        stack.Services[name] = vmSpec
    }

    // Convert volumes
    for name := range compose.Volumes {
        stack.Volumes[name] = Volume{
            Type:   "ext4",
            SizeGB: 10, // Default 10GB
        }
    }

    return stack, nil
}

func convertImage(svc ComposeService) string {
    if svc.Build != nil {
        // For build directives, return a placeholder
        return fmt.Sprintf("local/%s:latest", svc.Build.Context)
    }
    return svc.Image
}

func getReplicas(svc ComposeService) int {
    if svc.Deploy != nil && svc.Deploy.Replicas > 0 {
        return svc.Deploy.Replicas
    }
    return 1
}

func getCPU(svc ComposeService) int {
    if svc.Deploy != nil && svc.Deploy.Resources != nil {
        if limit := svc.Deploy.Resources.Limits; limit != nil && limit.CPUs != "" {
            cpus, _ := strconv.ParseFloat(limit.CPUs, 64)
            return int(cpus)
        }
    }
    return 2 // Default 2 cores
}

func getMemory(svc ComposeService) int {
    if svc.Deploy != nil && svc.Deploy.Resources != nil {
        if limit := svc.Deploy.Resources.Limits; limit != nil && limit.Memory != "" {
            return parseMemoryMB(limit.Memory)
        }
    }
    return 512 // Default 512MB
}

func parseMemoryMB(memStr string) int {
    memStr = strings.ToUpper(memStr)
    
    if strings.HasSuffix(memStr, "G") {
        gb, _ := strconv.Atoi(strings.TrimSuffix(memStr, "G"))
        return gb * 1024
    }
    if strings.HasSuffix(memStr, "M") {
        mb, _ := strconv.Atoi(strings.TrimSuffix(memStr, "M"))
        return mb
    }
    
    return 512 // Default
}

func parsePort(portStr string) (PortMapping, error) {
    parts := strings.Split(portStr, ":")
    if len(parts) != 2 {
        return PortMapping{}, fmt.Errorf("invalid port format: %s", portStr)
    }

    host, err := strconv.Atoi(parts[0])
    if err != nil {
        return PortMapping{}, fmt.Errorf("invalid host port: %s", parts[0])
    }

    guest, err := strconv.Atoi(parts[1])
    if err != nil {
        return PortMapping{}, fmt.Errorf("invalid guest port: %s", parts[1])
    }

    return PortMapping{
        Host:  host,
        Guest: guest,
        Proto: "tcp",
    }, nil
}

func parseVolume(volStr string) (VolumeMount, error) {
    parts := strings.Split(volStr, ":")
    if len(parts) < 2 {
        return VolumeMount{}, fmt.Errorf("invalid volume format: %s", volStr)
    }

    return VolumeMount{
        Volume: parts[0],
        Path:   parts[1],
    }, nil
}
```

**Checklist:**
- [ ] Implement `VolantStack` struct
- [ ] Convert services to VM specs
- [ ] Convert ports (handle host:guest format)
- [ ] Convert volumes
- [ ] Convert resource limits
- [ ] Handle `depends_on`
- [ ] Unit tests for generator

## 9.4 Task 3: CLI Tool

**File:** `volant/cmd/volant-compose/main.go`

```go
package main

import (
    "flag"
    "fmt"
    "gopkg.in/yaml.v3"
    "os"
    "path/filepath"
    "strings"
)

func main() {
    inputFile := flag.String("f", "docker-compose.yml", "Input docker-compose.yml file")
    outputFile := flag.String("o", "volant.yaml", "Output volant.yaml file")
    stackName := flag.String("name", "", "Stack name (default: directory name)")
    flag.Parse()

    // Parse docker-compose.yml
    compose, err := ParseDockerCompose(*inputFile)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error parsing docker-compose.yml: %v\n", err)
        os.Exit(1)
    }

    // Determine stack name
    name := *stackName
    if name == "" {
        cwd, _ := os.Getwd()
        name = filepath.Base(cwd)
    }

    // Convert to Volant stack
    stack, err := ConvertToVolantStack(compose, name)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error converting to Volant stack: %v\n", err)
        os.Exit(1)
    }

    // Write volant.yaml
    data, err := yaml.Marshal(stack)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error marshaling YAML: %v\n", err)
        os.Exit(1)
    }

    if err := os.WriteFile(*outputFile, data, 0644); err != nil {
        fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("✓ Converted %s → %s\n", *inputFile, *outputFile)
    printWarnings(compose, stack)
}

func printWarnings(compose *DockerCompose, stack *VolantStack) {
    // Warn about unsupported features
    warnings := []string{}

    for name, svc := range compose.Services {
        if svc.Build != nil {
            warnings = append(warnings, 
                fmt.Sprintf("Service '%s': build directive requires manual image creation with Fledge", name))
        }
        if svc.Restart != "" && svc.Restart != "no" {
            warnings = append(warnings, 
                fmt.Sprintf("Service '%s': restart policy not yet supported in Volant", name))
        }
        if len(svc.Networks) > 0 {
            warnings = append(warnings, 
                fmt.Sprintf("Service '%s': custom networks not yet supported (all VMs on vbr0)", name))
        }
    }

    if len(warnings) > 0 {
        fmt.Println("\n⚠ Warnings:")
        for _, w := range warnings {
            fmt.Printf("  - %s\n", w)
        }
    }
}
```

**Build and test:**

```bash
# Build
go build -o bin/volant-compose cmd/volant-compose/*.go

# Test
./bin/volant-compose -f testdata/docker-compose.yml -o testdata/volant.yaml

# Expected output: testdata/volant.yaml
cat testdata/volant.yaml
```

**Expected `volant.yaml` output:**

```yaml
stack:
  name: myapp
  version: "1.0"

services:
  web:
    image: nginx:alpine
    replicas: 1
    cpu: 2
    memory: 512
    environment:
      NGINX_PORT: "80"
    ports:
      - host: 8080
        guest: 80
        proto: tcp
    volumes:
      - volume: web-data
        path: /usr/share/nginx/html
    depends_on:
      - api

  api:
    image: local/api:latest
    replicas: 3
    cpu: 2
    memory: 1024
    environment:
      DATABASE_URL: postgres://db:5432/myapp
    ports:
      - host: 3000
        guest: 3000
        proto: tcp

  db:
    image: postgres:15
    replicas: 1
    cpu: 2
    memory: 512
    environment:
      POSTGRES_PASSWORD: secret
    volumes:
      - volume: db-data
        path: /var/lib/postgresql/data

volumes:
  web-data:
    type: ext4
    size_gb: 10
  db-data:
    type: ext4
    size_gb: 10
```

**Checklist:**
- [ ] CLI parsing with flags
- [ ] Auto-detect stack name from directory
- [ ] Output YAML to file
- [ ] Print warnings for unsupported features
- [ ] Integration test

## 9.5 Task 4: Build Directive Handling

**File:** `volant/cmd/volant-compose/fledge_builder.go`

```go
package main

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
)

// GenerateFledgeConfig creates a fledge.toml for services with build directives
func GenerateFledgeConfig(compose *DockerCompose) error {
    for name, svc := range compose.Services {
        if svc.Build == nil {
            continue
        }

        fledgePath := filepath.Join(svc.Build.Context, "fledge.toml")
        
        config := fmt.Sprintf(`[build]
name = "%s"
strategy = "oci-rootfs"

[build.source]
type = "dockerfile"
context = "."
dockerfile = "%s"

[runtime]
init = "/init"
command = %s
env = %s

[resources]
vcpus = 2
memory_mb = 512
`, 
            name,
            getDockerfile(svc.Build),
            formatCommand(svc.Command),
            formatEnv(svc.Environment),
        )

        if err := os.WriteFile(fledgePath, []byte(config), 0644); err != nil {
            return fmt.Errorf("write fledge.toml for %s: %w", name, err)
        }

        fmt.Printf("✓ Created %s\n", fledgePath)
    }

    return nil
}

func getDockerfile(build *ComposeBuild) string {
    if build.Dockerfile != "" {
        return build.Dockerfile
    }
    return "Dockerfile"
}

func formatCommand(cmd []string) string {
    if len(cmd) == 0 {
        return "[]"
    }
    return fmt.Sprintf(`["%s"]`, cmd[0]) // Simplified
}

func formatEnv(env map[string]string) string {
    if len(env) == 0 {
        return "{}"
    }
    
    result := "{ "
    first := true
    for k, v := range env {
        if !first {
            result += ", "
        }
        result += fmt.Sprintf(`"%s" = "%s"`, k, v)
        first = false
    }
    result += " }"
    
    return result
}

// BuildImages builds all images with Fledge
func BuildImages(compose *DockerCompose) error {
    for name, svc := range compose.Services {
        if svc.Build == nil {
            continue
        }

        fmt.Printf("Building image for %s with Fledge...\n", name)

        cmd := exec.Command("fledge", "build", "-f", filepath.Join(svc.Build.Context, "fledge.toml"))
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr

        if err := cmd.Run(); err != nil {
            return fmt.Errorf("build %s: %w", name, err)
        }

        fmt.Printf("✓ Built %s\n", name)
    }

    return nil
}
```

**CLI integration:**

```go
// Add flag in main.go
buildImages := flag.Bool("build", false, "Build images with Fledge")

// After ConvertToVolantStack
if *buildImages {
    fmt.Println("\nGenerating Fledge configs...")
    if err := GenerateFledgeConfig(compose); err != nil {
        fmt.Fprintf(os.Stderr, "Error generating Fledge configs: %v\n", err)
        os.Exit(1)
    }

    fmt.Println("\nBuilding images...")
    if err := BuildImages(compose); err != nil {
        fmt.Fprintf(os.Stderr, "Error building images: %v\n", err)
        os.Exit(1)
    }
}
```

**Test:**

```bash
# Create test Docker project
mkdir -p testdata/api
cat > testdata/api/Dockerfile << 'EOF'
FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go build -o server .
CMD ["./server"]
EOF

# Convert and build
./bin/volant-compose -f testdata/docker-compose.yml -build

# Verify fledge.toml created
cat testdata/api/fledge.toml
```

**Checklist:**
- [ ] Generate `fledge.toml` from build directive
- [ ] Invoke Fledge to build images
- [ ] Handle build args
- [ ] Error handling
- [ ] Integration test

## 9.6 Task 5: Documentation

**File:** `docs/docker-compose-migration.md`

```markdown
# Docker Compose Migration Guide

## Overview

`volant-compose` converts Docker Compose files to Volant stack manifests.

## Installation

```bash
go install github.com/volant-project/volant/cmd/volant-compose@latest
```

## Basic Usage

```bash
# Convert docker-compose.yml → volant.yaml
volant-compose -f docker-compose.yml -o volant.yaml

# With auto-build
volant-compose -f docker-compose.yml -build
```

## Supported Features

| Docker Compose | Volant | Notes |
|----------------|--------|-------|
| `image` | ✅ | Direct mapping |
| `build` | ✅ | Generates `fledge.toml` |
| `ports` | ✅ | Host:guest mapping |
| `environment` | ✅ | Direct mapping |
| `volumes` | ✅ | Persistent volumes |
| `depends_on` | ✅ | Startup ordering |
| `deploy.replicas` | ✅ | VM scaling |
| `deploy.resources` | ✅ | CPU/memory limits |
| `networks` | ⚠️  | All VMs on `vbr0` |
| `restart` | ❌ | Not yet supported |
| `healthcheck` | ❌ | Not yet supported |

## Example

**Input:** `docker-compose.yml`

```yaml
version: '3'
services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
    environment:
      NGINX_HOST: example.com
```

**Output:** `volant.yaml`

```yaml
stack:
  name: myapp
  version: "1.0"

services:
  web:
    image: nginx:alpine
    cpu: 2
    memory: 512
    environment:
      NGINX_HOST: example.com
    ports:
      - host: 8080
        guest: 80
        proto: tcp
```

**Deploy:**

```bash
volant stack deploy volant.yaml
```

## Build Directive

When `build` is specified, `volant-compose` generates a `fledge.toml` file:

**Input:**

```yaml
services:
  api:
    build:
      context: ./api
      dockerfile: Dockerfile
    command: ["./server", "--port", "3000"]
```

**Generated:** `api/fledge.toml`

```toml
[build]
name = "api"
strategy = "oci-rootfs"

[build.source]
type = "dockerfile"
context = "."
dockerfile = "Dockerfile"

[runtime]
command = ["./server", "--port", "3000"]
```

Build with:

```bash
volant-compose -f docker-compose.yml -build
```

## Migration Checklist

- [ ] Convert compose file: `volant-compose -f docker-compose.yml`
- [ ] Build images: `volant-compose -build` or manually with Fledge
- [ ] Deploy stack: `volant stack deploy volant.yaml`
- [ ] Verify VMs: `volant vm list`
- [ ] Test connectivity: `curl http://<vm-name>.volant:8080`

## Differences from Docker Compose

### Architecture

- **Docker Compose:** Containers (shared kernel)
- **Volant:** MicroVMs (isolated kernels)

### Networking

- **Docker Compose:** Multiple custom networks
- **Volant:** Single bridge network (`vbr0`) with DNS

### Volumes

- **Docker Compose:** Bind mounts and named volumes
- **Volant:** Persistent volumes with ext4/squashfs

### Performance

- **Startup:** Volant is slightly slower (kernel boot)
- **Runtime:** Volant has better isolation
- **Resources:** Volant uses more memory per service

## Troubleshooting

**Problem:** Image not found

```bash
# Pull image first
docker pull nginx:alpine
fledge import nginx:alpine
```

**Problem:** Port already in use

```bash
# Check what's using the port
volant vm list | grep 8080
```

**Problem:** Build fails

```bash
# Build manually with Fledge
cd api
fledge build -f fledge.toml
```
```

**Checklist:**
- [ ] Create `docker-compose-migration.md`
- [ ] Document converter usage
- [ ] Show supported features table
- [ ] Provide examples
- [ ] Explain differences from Docker Compose
- [ ] Add troubleshooting section

## 9.7 Success Criteria

**Track F Complete When:**
- [ ] `volant-compose` CLI tool works end-to-end
- [ ] Converts services, ports, volumes, env vars
- [ ] Handles `build` directive (generates `fledge.toml`)
- [ ] Generates valid `volant.yaml` files
- [ ] Warnings for unsupported features
- [ ] Documentation complete
- [ ] Integration tests pass
- [ ] **Branch merged to main**

---

# 10. PHASE 1 TRACK D: STACK ORCHESTRATOR

**Branch:** `feature/stack-orchestrator`  
**Duration:** Week 3-4  
**Dependencies:** **Track A (Env+DNS)** - requires service discovery  
**Owner:** Track D Team

## 10.1 Overview

Implement multi-service stack orchestration with dependency management, health checks, and rollback. This is the core of the "Docker killer" vision.

**Key Insight:** Stacks are groups of VMs that work together. The orchestrator must handle startup ordering, health checks, and failure scenarios.

## 10.2 Task 1: Database Schema

**File:** `volant/internal/server/db/sqlite/migrations/009_stacks.sql`

```sql
-- Stack definitions
CREATE TABLE stacks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    version TEXT NOT NULL,
    manifest_yaml TEXT NOT NULL,  -- Full volant.yaml
    status TEXT NOT NULL,          -- deploying|running|stopping|stopped|failed
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Stack VMs (many-to-many)
CREATE TABLE stack_vms (
    stack_id INTEGER NOT NULL,
    vm_name TEXT NOT NULL,
    service_name TEXT NOT NULL,    -- Name in volant.yaml
    replica_index INTEGER NOT NULL, -- 0-based replica number
    depends_on TEXT,               -- JSON array of service names
    
    PRIMARY KEY (stack_id, vm_name),
    FOREIGN KEY (stack_id) REFERENCES stacks(id) ON DELETE CASCADE,
    FOREIGN KEY (vm_name) REFERENCES vms(name) ON DELETE CASCADE
);

-- Stack deployment history
CREATE TABLE stack_deployments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    stack_id INTEGER NOT NULL,
    version TEXT NOT NULL,
    status TEXT NOT NULL,          -- success|failed|rolled_back
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    
    FOREIGN KEY (stack_id) REFERENCES stacks(id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX idx_stacks_name ON stacks(name);
CREATE INDEX idx_stacks_status ON stacks(status);
CREATE INDEX idx_stack_vms_stack_id ON stack_vms(stack_id);
CREATE INDEX idx_stack_deployments_stack_id ON stack_deployments(stack_id);
```

**Apply migration:**

```bash
# Test migration
sqlite3 test.db < volant/internal/server/db/sqlite/migrations/009_stacks.sql

# Verify tables
sqlite3 test.db "SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'stack%';"
```

**Checklist:**
- [ ] Create `009_stacks.sql`
- [ ] Add `stacks` table
- [ ] Add `stack_vms` table
- [ ] Add `stack_deployments` table
- [ ] Test migration

## 10.3 Task 2: Stack Manifest Parser

**File:** `volant/pkg/stack/manifest.go`

```go
package stack

import (
    "fmt"
    "gopkg.in/yaml.v3"
    "os"
)

// StackManifest represents a volant.yaml file
type StackManifest struct {
    Stack    StackMetadata        `yaml:"stack"`
    Services map[string]VMService `yaml:"services"`
    Volumes  map[string]Volume    `yaml:"volumes,omitempty"`
}

type StackMetadata struct {
    Name    string `yaml:"name"`
    Version string `yaml:"version"`
}

type VMService struct {
    Image       string            `yaml:"image"`
    Replicas    int               `yaml:"replicas"`
    CPU         int               `yaml:"cpu"`
    Memory      int               `yaml:"memory"`
    Environment map[string]string `yaml:"environment,omitempty"`
    Ports       []PortMapping     `yaml:"ports,omitempty"`
    Volumes     []VolumeMount     `yaml:"volumes,omitempty"`
    DependsOn   []string          `yaml:"depends_on,omitempty"`
    HealthCheck *HealthCheck      `yaml:"health_check,omitempty"`
}

type PortMapping struct {
    Host  int    `yaml:"host"`
    Guest int    `yaml:"guest"`
    Proto string `yaml:"proto"`
}

type VolumeMount struct {
    Volume   string `yaml:"volume"`
    Path     string `yaml:"path"`
    ReadOnly bool   `yaml:"read_only,omitempty"`
}

type Volume struct {
    Type   string `yaml:"type"`    // ext4|squashfs|bind
    SizeGB int    `yaml:"size_gb,omitempty"`
}

type HealthCheck struct {
    Test     []string `yaml:"test"`     // ["CMD", "curl", "-f", "http://localhost/health"]
    Interval int      `yaml:"interval"` // seconds
    Timeout  int      `yaml:"timeout"`  // seconds
    Retries  int      `yaml:"retries"`
}

func ParseManifest(path string) (*StackManifest, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read file: %w", err)
    }

    var manifest StackManifest
    if err := yaml.Unmarshal(data, &manifest); err != nil {
        return nil, fmt.Errorf("parse yaml: %w", err)
    }

    // Validate
    if err := manifest.Validate(); err != nil {
        return nil, err
    }

    return &manifest, nil
}

func (m *StackManifest) Validate() error {
    if m.Stack.Name == "" {
        return fmt.Errorf("stack.name is required")
    }
    if m.Stack.Version == "" {
        return fmt.Errorf("stack.version is required")
    }
    if len(m.Services) == 0 {
        return fmt.Errorf("at least one service is required")
    }

    // Validate services
    for name, svc := range m.Services {
        if svc.Image == "" {
            return fmt.Errorf("service %s: image is required", name)
        }
        if svc.Replicas < 1 {
            svc.Replicas = 1 // Default
        }
        if svc.CPU < 1 {
            svc.CPU = 2 // Default
        }
        if svc.Memory < 128 {
            svc.Memory = 512 // Default
        }

        // Validate depends_on references
        for _, dep := range svc.DependsOn {
            if _, exists := m.Services[dep]; !exists {
                return fmt.Errorf("service %s: depends_on references unknown service: %s", name, dep)
            }
        }

        // Validate volume references
        for _, mount := range svc.Volumes {
            if _, exists := m.Volumes[mount.Volume]; !exists {
                return fmt.Errorf("service %s: volume reference not found: %s", name, mount.Volume)
            }
        }
    }

    return nil
}

// GetStartupOrder returns services in dependency order
func (m *StackManifest) GetStartupOrder() ([]string, error) {
    // Topological sort
    visited := make(map[string]bool)
    order := []string{}

    var visit func(string) error
    visit = func(name string) error {
        if visited[name] {
            return nil
        }

        svc := m.Services[name]
        for _, dep := range svc.DependsOn {
            if err := visit(dep); err != nil {
                return err
            }
        }

        visited[name] = true
        order = append(order, name)
        return nil
    }

    for name := range m.Services {
        if err := visit(name); err != nil {
            return nil, err
        }
    }

    return order, nil
}
```

**Test:**

```go
// stack/manifest_test.go
func TestParseManifest(t *testing.T) {
    manifest, err := ParseManifest("testdata/test-stack.yaml")
    require.NoError(t, err)

    assert.Equal(t, "myapp", manifest.Stack.Name)
    assert.Equal(t, 3, len(manifest.Services))
}

func TestGetStartupOrder(t *testing.T) {
    manifest := &StackManifest{
        Services: map[string]VMService{
            "web":  {DependsOn: []string{"api"}},
            "api":  {DependsOn: []string{"db"}},
            "db":   {},
        },
    }

    order, err := manifest.GetStartupOrder()
    require.NoError(t, err)

    // Expected: db, api, web
    assert.Equal(t, []string{"db", "api", "web"}, order)
}
```

**Checklist:**
- [ ] Implement `StackManifest` struct
- [ ] Parse YAML
- [ ] Validate manifest
- [ ] Topological sort for `depends_on`
- [ ] Unit tests

## 10.4 Task 3: Stack Orchestrator

**File:** `volant/internal/server/stack/orchestrator.go`

```go
package stack

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "volant/internal/server/db"
    "volant/pkg/stack"
)

type Orchestrator struct {
    db     *db.Database
    vmAPI  VMManager
}

type VMManager interface {
    CreateVM(ctx context.Context, config *VMConfig) error
    StartVM(ctx context.Context, name string) error
    StopVM(ctx context.Context, name string) error
    DeleteVM(ctx context.Context, name string) error
    GetVMStatus(ctx context.Context, name string) (string, error)
}

type VMConfig struct {
    Name        string
    Image       string
    CPU         int
    Memory      int
    Environment map[string]string
    Ports       []PortMapping
    Volumes     []VolumeMount
}

type PortMapping struct {
    Host  int
    Guest int
    Proto string
}

type VolumeMount struct {
    Volume string
    Path   string
}

func NewOrchestrator(database *db.Database, vmMgr VMManager) *Orchestrator {
    return &Orchestrator{
        db:    database,
        vmAPI: vmMgr,
    }
}

// DeployStack deploys a new stack or updates an existing one
func (o *Orchestrator) DeployStack(ctx context.Context, manifestPath string) error {
    // Parse manifest
    manifest, err := stack.ParseManifest(manifestPath)
    if err != nil {
        return fmt.Errorf("parse manifest: %w", err)
    }

    // Start deployment
    deploymentID, err := o.startDeployment(ctx, manifest)
    if err != nil {
        return err
    }

    // Deploy services in dependency order
    order, err := manifest.GetStartupOrder()
    if err != nil {
        return o.failDeployment(ctx, deploymentID, err)
    }

    for _, serviceName := range order {
        service := manifest.Services[serviceName]

        // Create volumes if needed
        for _, mount := range service.Volumes {
            volSpec := manifest.Volumes[mount.Volume]
            if err := o.ensureVolume(ctx, mount.Volume, volSpec); err != nil {
                return o.failDeployment(ctx, deploymentID, err)
            }
        }

        // Create replicas
        for i := 0; i < service.Replicas; i++ {
            vmName := fmt.Sprintf("%s-%s-%d", manifest.Stack.Name, serviceName, i)

            vmConfig := &VMConfig{
                Name:        vmName,
                Image:       service.Image,
                CPU:         service.CPU,
                Memory:      service.Memory,
                Environment: service.Environment,
            }

            // Convert ports
            for _, p := range service.Ports {
                vmConfig.Ports = append(vmConfig.Ports, PortMapping{
                    Host:  p.Host + i, // Offset for replicas
                    Guest: p.Guest,
                    Proto: p.Proto,
                })
            }

            // Convert volumes
            for _, v := range service.Volumes {
                vmConfig.Volumes = append(vmConfig.Volumes, VolumeMount{
                    Volume: v.Volume,
                    Path:   v.Path,
                })
            }

            // Create and start VM
            if err := o.vmAPI.CreateVM(ctx, vmConfig); err != nil {
                return o.failDeployment(ctx, deploymentID, err)
            }

            if err := o.vmAPI.StartVM(ctx, vmName); err != nil {
                return o.failDeployment(ctx, deploymentID, err)
            }

            // Health check
            if service.HealthCheck != nil {
                if err := o.waitHealthy(ctx, vmName, service.HealthCheck); err != nil {
                    return o.failDeployment(ctx, deploymentID, err)
                }
            }

            // Record VM in stack
            if err := o.addVMToStack(ctx, manifest.Stack.Name, vmName, serviceName, i); err != nil {
                return o.failDeployment(ctx, deploymentID, err)
            }
        }
    }

    // Mark deployment successful
    return o.completeDeployment(ctx, deploymentID)
}

func (o *Orchestrator) startDeployment(ctx context.Context, manifest *stack.StackManifest) (int64, error) {
    manifestYAML, _ := yaml.Marshal(manifest)

    // Insert or update stack
    stackID, err := o.db.Stacks().Upsert(ctx, &db.Stack{
        Name:         manifest.Stack.Name,
        Version:      manifest.Stack.Version,
        ManifestYAML: string(manifestYAML),
        Status:       "deploying",
    })
    if err != nil {
        return 0, err
    }

    // Create deployment record
    deploymentID, err := o.db.StackDeployments().Create(ctx, &db.StackDeployment{
        StackID: stackID,
        Version: manifest.Stack.Version,
        Status:  "in_progress",
    })

    return deploymentID, err
}

func (o *Orchestrator) completeDeployment(ctx context.Context, deploymentID int64) error {
    return o.db.StackDeployments().Update(ctx, deploymentID, &db.StackDeployment{
        Status:      "success",
        CompletedAt: time.Now(),
    })
}

func (o *Orchestrator) failDeployment(ctx context.Context, deploymentID int64, err error) error {
    o.db.StackDeployments().Update(ctx, deploymentID, &db.StackDeployment{
        Status:       "failed",
        CompletedAt:  time.Now(),
        ErrorMessage: err.Error(),
    })

    // TODO: Rollback VMs
    return fmt.Errorf("deployment failed: %w", err)
}

func (o *Orchestrator) ensureVolume(ctx context.Context, name string, spec stack.Volume) error {
    // Check if volume exists
    existing, _ := o.db.Volumes().GetByName(ctx, name)
    if existing != nil {
        return nil // Already exists
    }

    // Create volume
    return o.db.Volumes().Create(ctx, &db.Volume{
        Name:   name,
        Type:   spec.Type,
        SizeGB: spec.SizeGB,
    })
}

func (o *Orchestrator) waitHealthy(ctx context.Context, vmName string, check *stack.HealthCheck) error {
    interval := time.Duration(check.Interval) * time.Second
    timeout := time.Duration(check.Timeout) * time.Second

    deadline := time.Now().Add(timeout)

    for time.Now().Before(deadline) {
        // TODO: Execute health check command in VM
        status, err := o.vmAPI.GetVMStatus(ctx, vmName)
        if err == nil && status == "running" {
            return nil
        }

        time.Sleep(interval)
    }

    return fmt.Errorf("health check timeout for %s", vmName)
}

func (o *Orchestrator) addVMToStack(ctx context.Context, stackName, vmName, serviceName string, replica int) error {
    stack, err := o.db.Stacks().GetByName(ctx, stackName)
    if err != nil {
        return err
    }

    return o.db.StackVMs().Create(ctx, &db.StackVM{
        StackID:      stack.ID,
        VMName:       vmName,
        ServiceName:  serviceName,
        ReplicaIndex: replica,
    })
}

// StopStack stops all VMs in a stack
func (o *Orchestrator) StopStack(ctx context.Context, stackName string) error {
    vms, err := o.db.StackVMs().ListByStack(ctx, stackName)
    if err != nil {
        return err
    }

    // Stop in reverse order
    for i := len(vms) - 1; i >= 0; i-- {
        if err := o.vmAPI.StopVM(ctx, vms[i].VMName); err != nil {
            return err
        }
    }

    return o.db.Stacks().UpdateStatus(ctx, stackName, "stopped")
}

// DeleteStack deletes all VMs and the stack record
func (o *Orchestrator) DeleteStack(ctx context.Context, stackName string) error {
    vms, err := o.db.StackVMs().ListByStack(ctx, stackName)
    if err != nil {
        return err
    }

    // Delete VMs
    for _, vm := range vms {
        if err := o.vmAPI.DeleteVM(ctx, vm.VMName); err != nil {
            return err
        }
    }

    // Delete stack record (cascade deletes stack_vms)
    return o.db.Stacks().Delete(ctx, stackName)
}
```

**Checklist:**
- [ ] Implement `Orchestrator` struct
- [ ] Deploy stack with dependency ordering
- [ ] Create VMs with replicas
- [ ] Health check polling
- [ ] Stop/delete stack operations
- [ ] Error handling and rollback

## 10.5 Task 4: REST API

**File:** `volant/internal/server/api/stacks.go`

```go
package api

import (
    "encoding/json"
    "net/http"

    "volant/internal/server/stack"
)

type StacksHandler struct {
    orchestrator *stack.Orchestrator
}

func NewStacksHandler(orch *stack.Orchestrator) *StacksHandler {
    return &StacksHandler{orchestrator: orch}
}

// POST /api/v1/stacks
func (h *StacksHandler) Deploy(w http.ResponseWriter, r *http.Request) {
    var req struct {
        ManifestPath string `json:"manifest_path"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }

    if err := h.orchestrator.DeployStack(r.Context(), req.ManifestPath); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{"status": "deployed"})
}

// GET /api/v1/stacks
func (h *StacksHandler) List(w http.ResponseWriter, r *http.Request) {
    stacks, err := h.orchestrator.ListStacks(r.Context())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(stacks)
}

// GET /api/v1/stacks/:name
func (h *StacksHandler) Get(w http.ResponseWriter, r *http.Request) {
    name := r.PathValue("name")

    stack, err := h.orchestrator.GetStack(r.Context(), name)
    if err != nil {
        http.Error(w, "stack not found", http.StatusNotFound)
        return
    }

    json.NewEncoder(w).Encode(stack)
}

// DELETE /api/v1/stacks/:name
func (h *StacksHandler) Delete(w http.ResponseWriter, r *http.Request) {
    name := r.PathValue("name")

    if err := h.orchestrator.DeleteStack(r.Context(), name); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/stacks/:name/stop
func (h *StacksHandler) Stop(w http.ResponseWriter, r *http.Request) {
    name := r.PathValue("name")

    if err := h.orchestrator.StopStack(r.Context(), name); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}
```

**Register routes:**

```go
// volant/internal/server/server.go
func (s *Server) setupRoutes() {
    stacksHandler := api.NewStacksHandler(s.orchestrator)

    s.mux.HandleFunc("POST /api/v1/stacks", stacksHandler.Deploy)
    s.mux.HandleFunc("GET /api/v1/stacks", stacksHandler.List)
    s.mux.HandleFunc("GET /api/v1/stacks/{name}", stacksHandler.Get)
    s.mux.HandleFunc("DELETE /api/v1/stacks/{name}", stacksHandler.Delete)
    s.mux.HandleFunc("POST /api/v1/stacks/{name}/stop", stacksHandler.Stop)
}
```

**Test:**

```bash
# Deploy stack
curl -X POST http://localhost:7893/api/v1/stacks \
  -H "Content-Type: application/json" \
  -d '{"manifest_path": "/path/to/volant.yaml"}'

# List stacks
curl http://localhost:7893/api/v1/stacks

# Get stack details
curl http://localhost:7893/api/v1/stacks/myapp

# Stop stack
curl -X POST http://localhost:7893/api/v1/stacks/myapp/stop

# Delete stack
curl -X DELETE http://localhost:7893/api/v1/stacks/myapp
```

**Checklist:**
- [ ] Implement REST handlers
- [ ] Deploy endpoint
- [ ] List/Get endpoints
- [ ] Stop/Delete endpoints
- [ ] Integration tests

## 10.6 Task 5: CLI Commands

**File:** `volant/cmd/volant/stack.go`

```go
package main

import (
    "fmt"
    "os"
    "text/tabwriter"

    "github.com/spf13/cobra"
)

var stackCmd = &cobra.Command{
    Use:   "stack",
    Short: "Manage stacks",
}

var stackDeployCmd = &cobra.Command{
    Use:   "deploy <manifest.yaml>",
    Short: "Deploy a stack",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        manifestPath := args[0]

        resp, err := client.Post("/api/v1/stacks", map[string]string{
            "manifest_path": manifestPath,
        })
        if err != nil {
            return err
        }

        fmt.Printf("✓ Stack deployed from %s\n", manifestPath)
        return nil
    },
}

var stackListCmd = &cobra.Command{
    Use:   "list",
    Short: "List all stacks",
    RunE: func(cmd *cobra.Command, args []string) error {
        var stacks []Stack
        if err := client.Get("/api/v1/stacks", &stacks); err != nil {
            return err
        }

        w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
        fmt.Fprintln(w, "NAME\tVERSION\tSTATUS\tVMS\tCREATED")

        for _, stack := range stacks {
            fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
                stack.Name,
                stack.Version,
                stack.Status,
                len(stack.VMs),
                stack.CreatedAt.Format("2006-01-02 15:04"),
            )
        }

        return w.Flush()
    },
}

var stackStopCmd = &cobra.Command{
    Use:   "stop <name>",
    Short: "Stop a stack",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        name := args[0]

        if err := client.Post(fmt.Sprintf("/api/v1/stacks/%s/stop", name), nil); err != nil {
            return err
        }

        fmt.Printf("✓ Stack %s stopped\n", name)
        return nil
    },
}

var stackDeleteCmd = &cobra.Command{
    Use:   "delete <name>",
    Short: "Delete a stack",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        name := args[0]

        // Confirm deletion
        fmt.Printf("Delete stack %s? This will delete all VMs. [y/N]: ", name)
        var confirm string
        fmt.Scanln(&confirm)

        if confirm != "y" && confirm != "Y" {
            return fmt.Errorf("cancelled")
        }

        if err := client.Delete(fmt.Sprintf("/api/v1/stacks/%s", name)); err != nil {
            return err
        }

        fmt.Printf("✓ Stack %s deleted\n", name)
        return nil
    },
}

var stackLogsCmd = &cobra.Command{
    Use:   "logs <name>",
    Short: "Show logs for all VMs in a stack",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        name := args[0]

        var stack Stack
        if err := client.Get(fmt.Sprintf("/api/v1/stacks/%s", name), &stack); err != nil {
            return err
        }

        for _, vm := range stack.VMs {
            fmt.Printf("==> %s (service: %s, replica: %d)\n", vm.VMName, vm.ServiceName, vm.ReplicaIndex)
            // TODO: Fetch and print logs
        }

        return nil
    },
}

func init() {
    stackCmd.AddCommand(stackDeployCmd)
    stackCmd.AddCommand(stackListCmd)
    stackCmd.AddCommand(stackStopCmd)
    stackCmd.AddCommand(stackDeleteCmd)
    stackCmd.AddCommand(stackLogsCmd)
    rootCmd.AddCommand(stackCmd)
}
```

**Test:**

```bash
# Deploy
volant stack deploy volant.yaml

# List
volant stack list

# Stop
volant stack stop myapp

# Delete
volant stack delete myapp

# Logs
volant stack logs myapp
```

**Checklist:**
- [ ] `volant stack deploy` command
- [ ] `volant stack list` command
- [ ] `volant stack stop` command
- [ ] `volant stack delete` command
- [ ] `volant stack logs` command
- [ ] Confirmation prompts
- [ ] Integration tests

## 10.7 Success Criteria

**Track D Complete When:**
- [ ] Stack manifest parser works
- [ ] Orchestrator deploys stacks with dependency ordering
- [ ] Replicas are created correctly
- [ ] Health checks work
- [ ] REST API functional
- [ ] CLI commands work
- [ ] Database migrations applied
- [ ] Integration tests pass
- [ ] Documentation complete
- [ ] **Branch merged to main**

---

# 11. PHASE 2 TRACK E: WEB UI ENHANCEMENTS

**Branch:** `feature/web-ui-stacks`  
**Duration:** Week 4  
**Dependencies:** **Track D (Stack Orchestrator)** - requires stack API  
**Owner:** Track E Team

## 11.1 Overview

Extend the Web UI to support stack deployment, monitoring, and management. Build on the existing Next.js 15 / React 19 foundation.

**Key Insight:** The Web UI already has 42/42 API endpoints implemented. We just need to add stack-specific views.

## 11.2 Task 1: Stack List View

**File:** `web/app/stacks/page.tsx`

```typescript
'use client';

import { useState, useEffect } from 'react';
import { Stack, StacksClient } from '@/lib/api/stacks';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import Link from 'next/link';

export default function StacksPage() {
  const [stacks, setStacks] = useState<Stack[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadStacks();
  }, []);

  async function loadStacks() {
    try {
      const client = new StacksClient();
      const data = await client.list();
      setStacks(data);
    } catch (error) {
      console.error('Failed to load stacks:', error);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="p-8">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">Stacks</h1>
        <Link href="/stacks/deploy">
          <Button>Deploy Stack</Button>
        </Link>
      </div>

      {loading ? (
        <p>Loading...</p>
      ) : stacks.length === 0 ? (
        <Card className="p-8 text-center text-gray-500">
          No stacks deployed. Deploy your first stack to get started.
        </Card>
      ) : (
        <div className="grid gap-4">
          {stacks.map((stack) => (
            <Card key={stack.name} className="p-6">
              <div className="flex justify-between items-start">
                <div>
                  <h2 className="text-xl font-semibold">{stack.name}</h2>
                  <p className="text-sm text-gray-500">Version {stack.version}</p>
                </div>
                <Badge variant={getStatusVariant(stack.status)}>
                  {stack.status}
                </Badge>
              </div>

              <div className="mt-4 grid grid-cols-3 gap-4 text-sm">
                <div>
                  <p className="text-gray-500">Services</p>
                  <p className="font-semibold">{stack.service_count}</p>
                </div>
                <div>
                  <p className="text-gray-500">VMs</p>
                  <p className="font-semibold">{stack.vm_count}</p>
                </div>
                <div>
                  <p className="text-gray-500">Created</p>
                  <p className="font-semibold">{formatDate(stack.created_at)}</p>
                </div>
              </div>

              <div className="mt-4 flex gap-2">
                <Link href={`/stacks/${stack.name}`}>
                  <Button variant="outline" size="sm">View</Button>
                </Link>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handleStop(stack.name)}
                >
                  Stop
                </Button>
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => handleDelete(stack.name)}
                >
                  Delete
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

function getStatusVariant(status: string) {
  switch (status) {
    case 'running': return 'success';
    case 'deploying': return 'warning';
    case 'failed': return 'destructive';
    default: return 'secondary';
  }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString();
}

async function handleStop(name: string) {
  if (!confirm(`Stop stack ${name}?`)) return;

  const client = new StacksClient();
  await client.stop(name);
  window.location.reload();
}

async function handleDelete(name: string) {
  if (!confirm(`Delete stack ${name}? This will delete all VMs.`)) return;

  const client = new StacksClient();
  await client.delete(name);
  window.location.reload();
}
```

**Checklist:**
- [ ] Create `app/stacks/page.tsx`
- [ ] Display stack list
- [ ] Show status badges
- [ ] Stop/delete actions
- [ ] Link to detail view

## 11.3 Task 2: Stack Detail View

**File:** `web/app/stacks/[name]/page.tsx`

```typescript
'use client';

import { useState, useEffect } from 'react';
import { useParams } from 'next/navigation';
import { Stack, StackDetail, StacksClient } from '@/lib/api/stacks';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';

export default function StackDetailPage() {
  const params = useParams();
  const stackName = params.name as string;

  const [stack, setStack] = useState<StackDetail | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadStack();
  }, [stackName]);

  async function loadStack() {
    try {
      const client = new StacksClient();
      const data = await client.get(stackName);
      setStack(data);
    } catch (error) {
      console.error('Failed to load stack:', error);
    } finally {
      setLoading(false);
    }
  }

  if (loading) return <p>Loading...</p>;
  if (!stack) return <p>Stack not found</p>;

  return (
    <div className="p-8">
      <div className="mb-6">
        <div className="flex items-center gap-3">
          <h1 className="text-3xl font-bold">{stack.name}</h1>
          <Badge variant={getStatusVariant(stack.status)}>
            {stack.status}
          </Badge>
        </div>
        <p className="text-gray-500">Version {stack.version}</p>
      </div>

      <Tabs defaultValue="services">
        <TabsList>
          <TabsTrigger value="services">Services</TabsTrigger>
          <TabsTrigger value="vms">VMs</TabsTrigger>
          <TabsTrigger value="volumes">Volumes</TabsTrigger>
          <TabsTrigger value="manifest">Manifest</TabsTrigger>
          <TabsTrigger value="deployments">History</TabsTrigger>
        </TabsList>

        <TabsContent value="services">
          <div className="grid gap-4">
            {stack.services.map((service) => (
              <Card key={service.name} className="p-6">
                <h3 className="text-lg font-semibold">{service.name}</h3>
                <p className="text-sm text-gray-500">{service.image}</p>

                <div className="mt-4 grid grid-cols-4 gap-4 text-sm">
                  <div>
                    <p className="text-gray-500">Replicas</p>
                    <p className="font-semibold">{service.replicas}</p>
                  </div>
                  <div>
                    <p className="text-gray-500">CPU</p>
                    <p className="font-semibold">{service.cpu} cores</p>
                  </div>
                  <div>
                    <p className="text-gray-500">Memory</p>
                    <p className="font-semibold">{service.memory} MB</p>
                  </div>
                  <div>
                    <p className="text-gray-500">Ports</p>
                    <p className="font-semibold">
                      {service.ports.map(p => `${p.host}:${p.guest}`).join(', ')}
                    </p>
                  </div>
                </div>

                {service.depends_on && service.depends_on.length > 0 && (
                  <div className="mt-4">
                    <p className="text-sm text-gray-500">Depends on:</p>
                    <div className="flex gap-2 mt-1">
                      {service.depends_on.map(dep => (
                        <Badge key={dep} variant="secondary">{dep}</Badge>
                      ))}
                    </div>
                  </div>
                )}
              </Card>
            ))}
          </div>
        </TabsContent>

        <TabsContent value="vms">
          <div className="grid gap-4">
            {stack.vms.map((vm) => (
              <Card key={vm.name} className="p-6">
                <div className="flex justify-between items-start">
                  <div>
                    <h3 className="text-lg font-semibold">{vm.name}</h3>
                    <p className="text-sm text-gray-500">
                      Service: {vm.service_name} (replica {vm.replica_index})
                    </p>
                  </div>
                  <Badge variant={getStatusVariant(vm.status)}>
                    {vm.status}
                  </Badge>
                </div>

                <div className="mt-4 grid grid-cols-3 gap-4 text-sm">
                  <div>
                    <p className="text-gray-500">IP Address</p>
                    <p className="font-mono">{vm.ip_address}</p>
                  </div>
                  <div>
                    <p className="text-gray-500">DNS</p>
                    <p className="font-mono">{vm.name}.volant</p>
                  </div>
                  <div>
                    <p className="text-gray-500">Started</p>
                    <p>{formatDate(vm.started_at)}</p>
                  </div>
                </div>
              </Card>
            ))}
          </div>
        </TabsContent>

        <TabsContent value="volumes">
          <div className="grid gap-4">
            {stack.volumes.map((volume) => (
              <Card key={volume.name} className="p-6">
                <h3 className="text-lg font-semibold">{volume.name}</h3>
                <div className="mt-4 grid grid-cols-3 gap-4 text-sm">
                  <div>
                    <p className="text-gray-500">Type</p>
                    <p className="font-semibold">{volume.type}</p>
                  </div>
                  <div>
                    <p className="text-gray-500">Size</p>
                    <p className="font-semibold">{volume.size_gb} GB</p>
                  </div>
                  <div>
                    <p className="text-gray-500">Mounts</p>
                    <p className="font-semibold">{volume.mount_count}</p>
                  </div>
                </div>
              </Card>
            ))}
          </div>
        </TabsContent>

        <TabsContent value="manifest">
          <Card className="p-6">
            <pre className="text-sm overflow-auto">
              {stack.manifest_yaml}
            </pre>
          </Card>
        </TabsContent>

        <TabsContent value="deployments">
          <div className="grid gap-4">
            {stack.deployments.map((deployment) => (
              <Card key={deployment.id} className="p-6">
                <div className="flex justify-between items-start">
                  <div>
                    <p className="text-sm text-gray-500">Version {deployment.version}</p>
                    <p className="text-xs text-gray-400">
                      {formatDateTime(deployment.started_at)}
                    </p>
                  </div>
                  <Badge variant={getStatusVariant(deployment.status)}>
                    {deployment.status}
                  </Badge>
                </div>

                {deployment.error_message && (
                  <div className="mt-4 p-4 bg-red-50 border border-red-200 rounded">
                    <p className="text-sm text-red-800">{deployment.error_message}</p>
                  </div>
                )}
              </Card>
            ))}
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}

function getStatusVariant(status: string) {
  switch (status) {
    case 'running':
    case 'success': return 'success';
    case 'deploying':
    case 'in_progress': return 'warning';
    case 'failed': return 'destructive';
    default: return 'secondary';
  }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString();
}

function formatDateTime(dateStr: string): string {
  return new Date(dateStr).toLocaleString();
}
```

**Checklist:**
- [ ] Create detail view
- [ ] Services tab
- [ ] VMs tab
- [ ] Volumes tab
- [ ] Manifest tab
- [ ] Deployment history tab

## 11.4 Task 3: Stack Deploy Form

**File:** `web/app/stacks/deploy/page.tsx`

```typescript
'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { StacksClient } from '@/lib/api/stacks';

export default function DeployStackPage() {
  const router = useRouter();
  const [manifestYaml, setManifestYaml] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  async function handleDeploy() {
    setLoading(true);
    setError('');

    try {
      const client = new StacksClient();
      await client.deploy(manifestYaml);
      router.push('/stacks');
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="p-8 max-w-4xl mx-auto">
      <h1 className="text-3xl font-bold mb-6">Deploy Stack</h1>

      <Card className="p-6">
        <div className="space-y-4">
          <div>
            <Label htmlFor="manifest">Stack Manifest (YAML)</Label>
            <Textarea
              id="manifest"
              placeholder={EXAMPLE_MANIFEST}
              value={manifestYaml}
              onChange={(e) => setManifestYaml(e.target.value)}
              rows={20}
              className="font-mono text-sm"
            />
          </div>

          {error && (
            <div className="p-4 bg-red-50 border border-red-200 rounded">
              <p className="text-sm text-red-800">{error}</p>
            </div>
          )}

          <div className="flex gap-2">
            <Button onClick={handleDeploy} disabled={loading}>
              {loading ? 'Deploying...' : 'Deploy'}
            </Button>
            <Button variant="outline" onClick={() => router.back()}>
              Cancel
            </Button>
          </div>
        </div>
      </Card>
    </div>
  );
}

const EXAMPLE_MANIFEST = `stack:
  name: myapp
  version: "1.0"

services:
  web:
    image: nginx:alpine
    replicas: 2
    cpu: 2
    memory: 512
    ports:
      - host: 8080
        guest: 80
        proto: tcp
    environment:
      NGINX_HOST: example.com
    depends_on:
      - api

  api:
    image: my-api:latest
    replicas: 3
    cpu: 4
    memory: 1024
    ports:
      - host: 3000
        guest: 3000
        proto: tcp
    environment:
      DATABASE_URL: postgres://db:5432/myapp
    depends_on:
      - db

  db:
    image: postgres:15
    replicas: 1
    cpu: 2
    memory: 2048
    environment:
      POSTGRES_PASSWORD: secret
    volumes:
      - volume: db-data
        path: /var/lib/postgresql/data

volumes:
  db-data:
    type: ext4
    size_gb: 20
`;
```

**Checklist:**
- [ ] Create deploy form
- [ ] YAML textarea with example
- [ ] Deploy button
- [ ] Error handling
- [ ] Redirect on success

## 11.5 Task 4: API Client

**File:** `web/lib/api/stacks.ts`

```typescript
export interface Stack {
  name: string;
  version: string;
  status: string;
  service_count: number;
  vm_count: number;
  created_at: string;
}

export interface StackDetail extends Stack {
  manifest_yaml: string;
  services: StackService[];
  vms: StackVM[];
  volumes: StackVolume[];
  deployments: StackDeployment[];
}

export interface StackService {
  name: string;
  image: string;
  replicas: number;
  cpu: number;
  memory: number;
  ports: PortMapping[];
  depends_on?: string[];
}

export interface StackVM {
  name: string;
  service_name: string;
  replica_index: number;
  status: string;
  ip_address: string;
  started_at: string;
}

export interface StackVolume {
  name: string;
  type: string;
  size_gb: number;
  mount_count: number;
}

export interface StackDeployment {
  id: number;
  version: string;
  status: string;
  started_at: string;
  completed_at?: string;
  error_message?: string;
}

export interface PortMapping {
  host: number;
  guest: number;
  proto: string;
}

export class StacksClient {
  private baseURL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:7893';

  async list(): Promise<Stack[]> {
    const response = await fetch(`${this.baseURL}/api/v1/stacks`);
    if (!response.ok) throw new Error('Failed to fetch stacks');
    return response.json();
  }

  async get(name: string): Promise<StackDetail> {
    const response = await fetch(`${this.baseURL}/api/v1/stacks/${name}`);
    if (!response.ok) throw new Error(`Stack ${name} not found`);
    return response.json();
  }

  async deploy(manifestYaml: string): Promise<void> {
    const response = await fetch(`${this.baseURL}/api/v1/stacks`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ manifest_yaml: manifestYaml }),
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(error);
    }
  }

  async stop(name: string): Promise<void> {
    const response = await fetch(`${this.baseURL}/api/v1/stacks/${name}/stop`, {
      method: 'POST',
    });

    if (!response.ok) {
      throw new Error(`Failed to stop stack ${name}`);
    }
  }

  async delete(name: string): Promise<void> {
    const response = await fetch(`${this.baseURL}/api/v1/stacks/${name}`, {
      method: 'DELETE',
    });

    if (!response.ok) {
      throw new Error(`Failed to delete stack ${name}`);
    }
  }
}
```

**Checklist:**
- [ ] Create TypeScript types
- [ ] Implement API client
- [ ] Handle errors
- [ ] Environment variable for API URL

## 11.6 Task 5: Navigation

**File:** `web/app/layout.tsx`

```typescript
// Add to existing navigation
<nav>
  <Link href="/">Dashboard</Link>
  <Link href="/vms">VMs</Link>
  <Link href="/stacks">Stacks</Link>  {/* NEW */}
  <Link href="/images">Images</Link>
  <Link href="/volumes">Volumes</Link>  {/* NEW */}
</nav>
```

**Checklist:**
- [ ] Add "Stacks" to navigation
- [ ] Add "Volumes" to navigation
- [ ] Update dashboard with stack stats

## 11.7 Success Criteria

**Track E Complete When:**
- [ ] Stack list view implemented
- [ ] Stack detail view implemented
- [ ] Stack deploy form implemented
- [ ] API client implemented
- [ ] Navigation updated
- [ ] All views responsive
- [ ] Integration tests pass
- [ ] **Branch merged to main**

---

# 12. PHASE 2 TRACK G: FLEDGE SERVER MODE

**Branch:** `feature/fledge-server`  
**Duration:** Week 4 (parallel with E)  
**Dependencies:** None (independent)  
**Owner:** Track G Team

## 12.1 Overview

Add a server mode to Fledge that allows remote builds via HTTP API. This enables the Web UI to trigger builds and Volant to request images on-demand.

**Key Insight:** Fledge is currently CLI-only. Adding a server mode makes it embeddable and automatable.

## 12.2 Task 1: HTTP Server

**File:** `fledge/cmd/fledged/main.go`

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "fledge/internal/server"
)

func main() {
    listenAddr := flag.String("listen", ":8080", "HTTP server listen address")
    workDir := flag.String("work-dir", "/tmp/fledge", "Working directory for builds")
    imageDir := flag.String("image-dir", "/var/lib/fledge/images", "Image output directory")
    flag.Parse()

    // Create server
    srv := server.New(&server.Config{
        ListenAddr: *listenAddr,
        WorkDir:    *workDir,
        ImageDir:   *imageDir,
    })

    // Start server
    httpServer := &http.Server{
        Addr:    *listenAddr,
        Handler: srv,
    }

    go func() {
        log.Printf("Fledge server listening on %s", *listenAddr)
        if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatalf("HTTP server error: %v", err)
        }
    }()

    // Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan

    log.Println("Shutting down...")

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := httpServer.Shutdown(ctx); err != nil {
        log.Fatalf("Shutdown error: %v", err)
    }

    log.Println("Server stopped")
}
```

**Checklist:**
- [ ] Create `fledged` command
- [ ] HTTP server setup
- [ ] Graceful shutdown
- [ ] Configuration flags

## 12.3 Task 2: Build API

**File:** `fledge/internal/server/server.go`

```go
package server

import (
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "sync"

    "fledge/internal/builder"
)

type Config struct {
    ListenAddr string
    WorkDir    string
    ImageDir   string
}

type Server struct {
    config *Config
    builds map[string]*BuildJob
    mu     sync.RWMutex
    mux    *http.ServeMux
}

type BuildJob struct {
    ID     string
    Status string // queued|building|completed|failed
    Config *builder.Config
    Output string
    Error  string
}

func New(config *Config) *Server {
    s := &Server{
        config: config,
        builds: make(map[string]*BuildJob),
        mux:    http.NewServeMux(),
    }

    s.setupRoutes()
    return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    s.mux.ServeHTTP(w, r)
}

func (s *Server) setupRoutes() {
    s.mux.HandleFunc("POST /api/v1/builds", s.handleBuild)
    s.mux.HandleFunc("GET /api/v1/builds/{id}", s.handleGetBuild)
    s.mux.HandleFunc("GET /api/v1/builds/{id}/logs", s.handleGetLogs)
    s.mux.HandleFunc("GET /api/v1/images", s.handleListImages)
    s.mux.HandleFunc("GET /api/v1/images/{name}/download", s.handleDownloadImage)
    s.mux.HandleFunc("GET /health", s.handleHealth)
}

// POST /api/v1/builds
func (s *Server) handleBuild(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Name       string            `json:"name"`
        Strategy   string            `json:"strategy"`   // oci-rootfs|initramfs
        Dockerfile string            `json:"dockerfile"` // For oci-rootfs
        Context    string            `json:"context"`
        Command    []string          `json:"command"`
        Env        map[string]string `json:"env"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }

    // Create build job
    jobID := generateID()
    job := &BuildJob{
        ID:     jobID,
        Status: "queued",
        Config: &builder.Config{
            Name:       req.Name,
            Strategy:   req.Strategy,
            Dockerfile: req.Dockerfile,
            Context:    req.Context,
            Command:    req.Command,
            Env:        req.Env,
            OutputDir:  s.config.ImageDir,
        },
    }

    s.mu.Lock()
    s.builds[jobID] = job
    s.mu.Unlock()

    // Start build asynchronously
    go s.runBuild(job)

    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]string{
        "id":     jobID,
        "status": "queued",
    })
}

func (s *Server) runBuild(job *BuildJob) {
    s.mu.Lock()
    job.Status = "building"
    s.mu.Unlock()

    log.Printf("Starting build %s (%s)", job.ID, job.Config.Name)

    // Run builder
    b := builder.New(job.Config)
    output, err := b.Build()

    s.mu.Lock()
    defer s.mu.Unlock()

    if err != nil {
        job.Status = "failed"
        job.Error = err.Error()
        log.Printf("Build %s failed: %v", job.ID, err)
    } else {
        job.Status = "completed"
        job.Output = output
        log.Printf("Build %s completed: %s", job.ID, output)
    }
}

// GET /api/v1/builds/:id
func (s *Server) handleGetBuild(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")

    s.mu.RLock()
    job, exists := s.builds[id]
    s.mu.RUnlock()

    if !exists {
        http.Error(w, "build not found", http.StatusNotFound)
        return
    }

    json.NewEncoder(w).Encode(job)
}

// GET /api/v1/builds/:id/logs
func (s *Server) handleGetLogs(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")

    logPath := filepath.Join(s.config.WorkDir, id, "build.log")
    data, err := os.ReadFile(logPath)
    if err != nil {
        http.Error(w, "logs not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "text/plain")
    w.Write(data)
}

// GET /api/v1/images
func (s *Server) handleListImages(w http.ResponseWriter, r *http.Request) {
    files, err := os.ReadDir(s.config.ImageDir)
    if err != nil {
        http.Error(w, "failed to list images", http.StatusInternalServerError)
        return
    }

    images := []map[string]interface{}{}
    for _, f := range files {
        if f.IsDir() {
            continue
        }

        info, _ := f.Info()
        images = append(images, map[string]interface{}{
            "name": f.Name(),
            "size": info.Size(),
            "modified": info.ModTime(),
        })
    }

    json.NewEncoder(w).Encode(images)
}

// GET /api/v1/images/:name/download
func (s *Server) handleDownloadImage(w http.ResponseWriter, r *http.Request) {
    name := r.PathValue("name")

    imagePath := filepath.Join(s.config.ImageDir, name)
    file, err := os.Open(imagePath)
    if err != nil {
        http.Error(w, "image not found", http.StatusNotFound)
        return
    }
    defer file.Close()

    info, _ := file.Stat()
    w.Header().Set("Content-Type", "application/octet-stream")
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", name))
    w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))

    io.Copy(w, file)
}

// GET /health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func generateID() string {
    return fmt.Sprintf("%d", time.Now().UnixNano())
}
```

**Checklist:**
- [ ] Implement HTTP server
- [ ] Build API endpoint
- [ ] Get build status endpoint
- [ ] List images endpoint
- [ ] Download image endpoint
- [ ] Health check endpoint

## 12.4 Task 3: Integration with Volant

**File:** `volant/internal/fledge/client.go`

```go
package fledge

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "time"
)

type Client struct {
    endpoint string
    client   *http.Client
}

type BuildRequest struct {
    Name       string            `json:"name"`
    Strategy   string            `json:"strategy"`
    Dockerfile string            `json:"dockerfile,omitempty"`
    Context    string            `json:"context,omitempty"`
    Command    []string          `json:"command,omitempty"`
    Env        map[string]string `json:"env,omitempty"`
}

type BuildResponse struct {
    ID     string `json:"id"`
    Status string `json:"status"`
}

type BuildStatus struct {
    ID     string `json:"id"`
    Status string `json:"status"`
    Output string `json:"output"`
    Error  string `json:"error"`
}

func NewClient(endpoint string) *Client {
    return &Client{
        endpoint: endpoint,
        client:   &http.Client{Timeout: 30 * time.Second},
    }
}

// Build submits a build request and polls until completion
func (c *Client) Build(req *BuildRequest) (string, error) {
    // Submit build
    buildResp, err := c.submitBuild(req)
    if err != nil {
        return "", err
    }

    // Poll for completion
    for {
        status, err := c.getBuildStatus(buildResp.ID)
        if err != nil {
            return "", err
        }

        switch status.Status {
        case "completed":
            return status.Output, nil
        case "failed":
            return "", fmt.Errorf("build failed: %s", status.Error)
        case "queued", "building":
            time.Sleep(2 * time.Second)
        default:
            return "", fmt.Errorf("unknown status: %s", status.Status)
        }
    }
}

func (c *Client) submitBuild(req *BuildRequest) (*BuildResponse, error) {
    data, _ := json.Marshal(req)
    resp, err := c.client.Post(
        fmt.Sprintf("%s/api/v1/builds", c.endpoint),
        "application/json",
        bytes.NewReader(data),
    )
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusAccepted {
        return nil, fmt.Errorf("build request failed: %d", resp.StatusCode)
    }

    var buildResp BuildResponse
    if err := json.NewDecoder(resp.Body).Decode(&buildResp); err != nil {
        return nil, err
    }

    return &buildResp, nil
}

func (c *Client) getBuildStatus(buildID string) (*BuildStatus, error) {
    resp, err := c.client.Get(
        fmt.Sprintf("%s/api/v1/builds/%s", c.endpoint, buildID),
    )
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var status BuildStatus
    if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
        return nil, err
    }

    return &status, nil
}

// DownloadImage downloads an image from Fledge server
func (c *Client) DownloadImage(name, destPath string) error {
    resp, err := c.client.Get(
        fmt.Sprintf("%s/api/v1/images/%s/download", c.endpoint, name),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("download failed: %d", resp.StatusCode)
    }

    // Create destination file
    if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
        return err
    }

    file, err := os.Create(destPath)
    if err != nil {
        return err
    }
    defer file.Close()

    _, err = io.Copy(file, resp.Body)
    return err
}
```

**Usage in Volant:**

```go
// When creating a VM with an image that doesn't exist locally:
func (s *Server) ensureImage(imageName string) error {
    // Check if image exists locally
    if imageExists(imageName) {
        return nil
    }

    // Try to build with Fledge server
    if fledgeEndpoint := os.Getenv("FLEDGE_ENDPOINT"); fledgeEndpoint != "" {
        client := fledge.NewClient(fledgeEndpoint)
        
        outputPath, err := client.Build(&fledge.BuildRequest{
            Name:     imageName,
            Strategy: "oci-rootfs",
        })
        if err != nil {
            return fmt.Errorf("fledge build: %w", err)
        }

        // Download image
        localPath := filepath.Join(s.imageDir, imageName)
        if err := client.DownloadImage(filepath.Base(outputPath), localPath); err != nil {
            return fmt.Errorf("download image: %w", err)
        }

        return nil
    }

    return fmt.Errorf("image %s not found and no Fledge server configured", imageName)
}
```

**Checklist:**
- [ ] Implement Fledge HTTP client
- [ ] Integrate with Volant
- [ ] Auto-build missing images
- [ ] Download built images

## 12.5 Task 4: Web UI Integration

**File:** `web/app/images/build/page.tsx`

```typescript
'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';

export default function BuildImagePage() {
  const router = useRouter();
  const [name, setName] = useState('');
  const [strategy, setStrategy] = useState('oci-rootfs');
  const [dockerfile, setDockerfile] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  async function handleBuild() {
    setLoading(true);
    setError('');

    try {
      const response = await fetch('http://localhost:8080/api/v1/builds', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, strategy, dockerfile }),
      });

      if (!response.ok) throw new Error('Build failed');

      const { id } = await response.json();

      // Poll for completion
      await pollBuildStatus(id);

      router.push('/images');
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  async function pollBuildStatus(buildId: string) {
    while (true) {
      const response = await fetch(`http://localhost:8080/api/v1/builds/${buildId}`);
      const data = await response.json();

      if (data.status === 'completed') {
        return;
      } else if (data.status === 'failed') {
        throw new Error(data.error);
      }

      await new Promise(resolve => setTimeout(resolve, 2000));
    }
  }

  return (
    <div className="p-8 max-w-4xl mx-auto">
      <h1 className="text-3xl font-bold mb-6">Build Image</h1>

      <Card className="p-6">
        <div className="space-y-4">
          <div>
            <Label htmlFor="name">Image Name</Label>
            <Input
              id="name"
              placeholder="my-app:latest"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>

          <div>
            <Label htmlFor="strategy">Build Strategy</Label>
            <Select
              id="strategy"
              value={strategy}
              onChange={(e) => setStrategy(e.target.value)}
            >
              <option value="oci-rootfs">OCI Rootfs (Docker)</option>
              <option value="initramfs">Initramfs (Minimal)</option>
            </Select>
          </div>

          {strategy === 'oci-rootfs' && (
            <div>
              <Label htmlFor="dockerfile">Dockerfile</Label>
              <Textarea
                id="dockerfile"
                placeholder="FROM alpine:latest&#10;RUN apk add --no-cache nginx&#10;CMD [&quot;nginx&quot;, &quot;-g&quot;, &quot;daemon off;&quot;]"
                value={dockerfile}
                onChange={(e) => setDockerfile(e.target.value)}
                rows={10}
                className="font-mono text-sm"
              />
            </div>
          )}

          {error && (
            <div className="p-4 bg-red-50 border border-red-200 rounded">
              <p className="text-sm text-red-800">{error}</p>
            </div>
          )}

          <div className="flex gap-2">
            <Button onClick={handleBuild} disabled={loading}>
              {loading ? 'Building...' : 'Build'}
            </Button>
            <Button variant="outline" onClick={() => router.back()}>
              Cancel
            </Button>
          </div>
        </div>
      </Card>
    </div>
  );
}
```

**Checklist:**
- [ ] Create build form
- [ ] Strategy selector
- [ ] Dockerfile editor
- [ ] Poll build status
- [ ] Show progress

## 12.6 Task 5: Documentation

**File:** `fledge/docs/server-mode.md`

```markdown
# Fledge Server Mode

## Overview

Fledge can run as an HTTP server to enable remote builds.

## Installation

```bash
go install github.com/fledge-project/fledge/cmd/fledged@latest
```

## Usage

```bash
# Start server
fledged --listen :8080 --image-dir /var/lib/fledge/images

# Environment variables
FLEDGE_LISTEN=:8080
FLEDGE_WORK_DIR=/tmp/fledge
FLEDGE_IMAGE_DIR=/var/lib/fledge/images
```

## API Endpoints

### POST /api/v1/builds

Submit a new build job.

**Request:**

```json
{
  "name": "my-app:latest",
  "strategy": "oci-rootfs",
  "dockerfile": "FROM alpine:latest\nCMD [\"sh\"]"
}
```

**Response:**

```json
{
  "id": "1234567890",
  "status": "queued"
}
```

### GET /api/v1/builds/:id

Get build status.

**Response:**

```json
{
  "id": "1234567890",
  "status": "completed",
  "output": "/var/lib/fledge/images/my-app-latest.ext4"
}
```

### GET /api/v1/images

List all built images.

**Response:**

```json
[
  {
    "name": "my-app-latest.ext4",
    "size": 52428800,
    "modified": "2025-01-15T10:30:00Z"
  }
]
```

### GET /api/v1/images/:name/download

Download an image file.

## Integration with Volant

Set the Fledge endpoint in Volant:

```bash
export FLEDGE_ENDPOINT=http://localhost:8080
volantd
```

Volant will automatically request builds from Fledge when images are missing.

## Security

**Warning:** Fledge server has no authentication. Use only in trusted environments or behind a firewall.

Future versions will support:
- API keys
- TLS
- Rate limiting
```

**Checklist:**
- [ ] Create documentation
- [ ] API reference
- [ ] Integration examples
- [ ] Security warnings

## 12.7 Success Criteria

**Track G Complete When:**
- [ ] `fledged` HTTP server works
- [ ] Build API functional
- [ ] Volant can request builds
- [ ] Web UI can trigger builds
- [ ] Images downloadable
- [ ] Documentation complete
- [ ] Integration tests pass
- [ ] **Branch merged to main**

---

# 13. PHASE 3: INTEGRATION & TESTING

**Duration:** Week 5  
**Dependencies:** All Phase 1 and Phase 2 tracks

## 13.1 Merge Strategy

**Merge order (to avoid conflicts):**

1. **Track A** (Env+DNS) → main
2. **Track B** (Volumes) → main
3. **Track C** (Drift L4) → main
4. **Track F** (Compose Converter) → main
5. **Track D** (Stack Orchestrator) → main
6. **Track E** (Web UI) → main
7. **Track G** (Fledge Server) → main

**Why this order:**
- Database migrations must merge in sequence (Track B before Track D)
- Track D depends on Track A (service discovery)
- Track E depends on Track D (stack API)

## 13.2 Integration Tests

**Test suite:** `volant/test/integration/stack_test.go`

```go
package integration

import (
    "testing"
    "time"

    "github.com/stretchr/testify/require"
)

func TestFullStackDeployment(t *testing.T) {
    // Start Volantd
    volantd := startVolantd(t)
    defer volantd.Stop()

    // Start Driftd
    driftd := startDriftd(t)
    defer driftd.Stop()

    // Start Fledged
    fledged := startFledged(t)
    defer fledged.Stop()

    // Deploy stack
    manifestPath := "testdata/wordpress-stack.yaml"
    err := volantd.DeployStack(manifestPath)
    require.NoError(t, err)

    // Wait for all VMs to be running
    require.Eventually(t, func() bool {
        vms, _ := volantd.ListVMs()
        return len(vms) == 2 && allRunning(vms)
    }, 60*time.Second, 2*time.Second)

    // Test DNS resolution
    ip, err := volantd.Resolve("wordpress.volant")
    require.NoError(t, err)
    require.NotEmpty(t, ip)

    // Test HTTP connectivity
    resp, err := httpGet(fmt.Sprintf("http://%s:8080", ip))
    require.NoError(t, err)
    require.Equal(t, 200, resp.StatusCode)

    // Test volume persistence
    volantd.StopStack("wordpress")
    volantd.StartStack("wordpress")

    // Data should still exist
    resp, err = httpGet(fmt.Sprintf("http://%s:8080/wp-admin", ip))
    require.NoError(t, err)
    require.Equal(t, 200, resp.StatusCode)

    // Clean up
    err = volantd.DeleteStack("wordpress")
    require.NoError(t, err)
}
```

**Test data:** `volant/test/integration/testdata/wordpress-stack.yaml`

```yaml
stack:
  name: wordpress
  version: "1.0"

services:
  wordpress:
    image: wordpress:latest
    replicas: 1
    cpu: 2
    memory: 1024
    ports:
      - host: 8080
        guest: 80
        proto: tcp
    environment:
      WORDPRESS_DB_HOST: mysql.volant
      WORDPRESS_DB_USER: wordpress
      WORDPRESS_DB_PASSWORD: secret
    depends_on:
      - mysql

  mysql:
    image: mysql:8.0
    replicas: 1
    cpu: 2
    memory: 2048
    environment:
      MYSQL_ROOT_PASSWORD: rootsecret
      MYSQL_DATABASE: wordpress
      MYSQL_USER: wordpress
      MYSQL_PASSWORD: secret
    volumes:
      - volume: mysql-data
        path: /var/lib/mysql

volumes:
  mysql-data:
    type: ext4
    size_gb: 20
```

**Run tests:**

```bash
go test -v ./test/integration/...
```

## 13.3 E2E Demonstration

**Demo script:** `scripts/demo.sh`

```bash
#!/bin/bash
set -e

echo "==> Starting Volant Stack Demo"

# Start services
echo "Starting Volantd..."
volantd --listen :7893 &
VOLANTD_PID=$!

echo "Starting Driftd..."
driftd --interface vbr0 &
DRIFTD_PID=$!

echo "Starting Fledged..."
fledged --listen :8080 &
FLEDGED_PID=$!

sleep 5

# Deploy stack
echo "Deploying WordPress stack..."
volant stack deploy examples/wordpress-stack.yaml

# Wait for readiness
echo "Waiting for stack to be ready..."
sleep 30

# Show status
echo "Stack status:"
volant stack list

echo "VM status:"
volant vm list

# Test connectivity
echo "Testing HTTP connectivity..."
curl -f http://wordpress.volant:8080

echo "✓ Stack deployed successfully!"

# Cleanup
echo "Cleaning up..."
kill $VOLANTD_PID $DRIFTD_PID $FLEDGED_PID
volant stack delete wordpress

echo "✓ Demo complete!"
```

## 13.4 Performance Benchmarks

**Benchmark:** Measure stack deployment time

```bash
time volant stack deploy examples/3-tier-app.yaml
```

**Expected results:**
- 3-service stack: < 60s
- 10-service stack: < 3min
- DNS resolution: < 10ms
- Drift L4 latency: < 10μs

## 13.5 Success Criteria

**Phase 3 Complete When:**
- [ ] All tracks merged to main
- [ ] Integration tests pass
- [ ] E2E demo works
- [ ] Performance benchmarks meet targets
- [ ] Documentation updated
- [ ] **Release v2.0 tagged**

---

# 14. API CONTRACTS BETWEEN TRACKS

This section defines the interfaces between parallel tracks to ensure compatibility.

## 14.1 Track A → Track D: Service Discovery

**Interface:** DNS Server

```go
// Track A provides:
type DNSServer interface {
    Resolve(name string) ([]string, error)
    AddRecord(name, ip string) error
    RemoveRecord(name string) error
}

// Track D uses:
func (o *Orchestrator) deployService(service *Service) error {
    // Create VM
    vm := createVM(service)
    
    // Register with DNS (Track A)
    dns.AddRecord(service.Name, vm.IPAddress)
}
```

**Contract:**
- DNS server listens on `192.168.127.1:53`
- Resolves `<name>.volant` to IP addresses
- Round-robin for multiple IPs

## 14.2 Track B → Track D: Volume Manager

**Interface:** Volume Operations

```go
// Track B provides:
type VolumeManager interface {
    CreateVolume(name string, sizeGB int, volType string) error
    DeleteVolume(name string) error
    AttachVolume(volumeName, vmName, mountPoint string) error
    DetachVolume(volumeName, vmName string) error
}

// Track D uses:
func (o *Orchestrator) deployService(service *Service) error {
    // Ensure volumes exist (Track B)
    for _, vol := range service.Volumes {
        volMgr.CreateVolume(vol.Name, vol.SizeGB, vol.Type)
        volMgr.AttachVolume(vol.Name, vm.Name, vol.MountPoint)
    }
}
```

**Contract:**
- Volumes are persistent across VM restarts
- Mount points are read/write by default
- `read_only` flag supported

## 14.3 Track D → Track E: Stack API

**Interface:** REST API

```
POST   /api/v1/stacks          - Deploy stack
GET    /api/v1/stacks          - List stacks
GET    /api/v1/stacks/:name    - Get stack details
DELETE /api/v1/stacks/:name    - Delete stack
POST   /api/v1/stacks/:name/stop - Stop stack
```

**Request/Response schemas defined in Track D.**

## 14.4 Track G → Track D: Image Building

**Interface:** Fledge Client

```go
// Track G provides:
type FledgeClient interface {
    Build(req *BuildRequest) (imagePath string, error)
    DownloadImage(name, destPath string) error
}

// Track D uses:
func (o *Orchestrator) ensureImage(imageName string) error {
    if !imageExists(imageName) {
        fledge.Build(&BuildRequest{Name: imageName})
    }
}
```

**Contract:**
- Fledge server at `http://localhost:8080`
- Environment variable: `FLEDGE_ENDPOINT`

---

# 15. DATABASE EVOLUTION STRATEGY

To avoid migration conflicts, database migrations must be carefully ordered.

## 15.1 Migration Numbering

**Existing migrations:**
- `001_initial.sql` - VMs, images
- `002_networking.sql` - Networks
- `...`
- `007_xxx.sql` (current)

**New migrations:**
- `008_volumes.sql` (Track B)
- `009_stacks.sql` (Track D)
- `010_dns.sql` (Track A) - if needed
- `011_drift.sql` (Track C) - if needed

## 15.2 Merge Order

**Critical:** Merge tracks in migration number order:
1. Track B (008) before Track D (009)
2. Any schema changes in Track A/C must use higher numbers

## 15.3 Testing Migrations

**Test script:** `scripts/test-migrations.sh`

```bash
#!/bin/bash
# Test all migrations in sequence

rm -f test.db

for migration in volant/internal/server/db/sqlite/migrations/*.sql; do
    echo "Applying $migration..."
    sqlite3 test.db < "$migration"
    
    # Verify
    if [ $? -ne 0 ]; then
        echo "❌ Migration failed: $migration"
        exit 1
    fi
done

echo "✓ All migrations applied successfully"

# Check schema
sqlite3 test.db ".schema"
```

---

# 16. TESTING STRATEGY

## 16.1 Unit Tests

**Each track must have:**
- Unit tests for all new functions
- > 80% code coverage
- Fast (< 5s per package)

**Example:**

```bash
# Track A
go test ./internal/server/dns/...

# Track B
go test ./internal/server/volumes/...

# Track D
go test ./pkg/stack/...
go test ./internal/server/stack/...
```

## 16.2 Integration Tests

**Full-stack tests:**

```bash
# Start all services
./scripts/start-services.sh

# Run integration tests
go test -v ./test/integration/...

# Stop services
./scripts/stop-services.sh
```

## 16.3 Manual Testing

**Checklist for each track:**

- [ ] Deploy a simple stack
- [ ] Deploy a complex stack (10+ services)
- [ ] Test DNS resolution
- [ ] Test volume persistence
- [ ] Test Drift L4 routing
- [ ] Convert docker-compose.yml
- [ ] Trigger build from Web UI
- [ ] Stop/start/delete operations

---

# 17. WORKTREE STRATEGY

## 17.1 Setup Worktrees

```bash
# Create worktrees for parallel development
cd /Users/marcxavier/Desktop/work

# Track A
git worktree add ../work-track-a feature/env-dns

# Track B
git worktree add ../work-track-b feature/volumes

# Track C
git worktree add ../work-track-c feature/drift-l4

# Track F
git worktree add ../work-track-f feature/compose-converter

# Track D
git worktree add ../work-track-d feature/stack-orchestrator

# Track E
git worktree add ../work-track-e feature/web-ui-stacks

# Track G
git worktree add ../work-track-g feature/fledge-server
```

## 17.2 Open Claude Code Instances

**Open 7 terminal windows:**

```bash
# Window 1 - Track A
cd ../work-track-a
code .
# In integrated terminal: claude

# Window 2 - Track B
cd ../work-track-b
code .
# In integrated terminal: claude

# ... repeat for all tracks
```

## 17.3 Provide Context

**For each Claude instance, provide this manifesto:**

```bash
# In each worktree
cat /Users/marcxavier/Desktop/work/VOLANT_MANIFESTO.md
```

**Then instruct:**

> "Read VOLANT_MANIFESTO.md. You are working on Track X. Implement all tasks in section X exactly as specified. Begin with Task 1."

## 17.4 Merge Back

**When a track is complete:**

```bash
# Switch to main repo
cd /Users/marcxavier/Desktop/work

# Pull track changes
git fetch origin feature/track-name

# Merge in order (see Phase 3)
git merge feature/track-name

# Test
go test ./...

# Push
git push origin main

# Clean up worktree
git worktree remove ../work-track-x
```

---

# 18. TROUBLESHOOTING

## 18.1 Common Issues

### Issue: DNS not resolving

**Symptom:** `curl http://myvm.volant` fails

**Fix:**
1. Check DNS server is running: `netstat -an | grep 192.168.127.1:53`
2. Verify VM IP is registered: `volant vm list`
3. Test DNS directly: `dig @192.168.127.1 myvm.volant`

### Issue: Volume not mounting

**Symptom:** VM boots but `/data` is empty

**Fix:**
1. Check volume exists: `volant volume list`
2. Verify mount in database: `sqlite3 volant.db "SELECT * FROM volume_mounts;"`
3. Check kernel command line includes volume mount

### Issue: Drift L4 not routing

**Symptom:** Ports not accessible

**Fix:**
1. Check Drift daemon: `systemctl status driftd`
2. Verify eBPF program loaded: `bpftool prog list`
3. Check routes: `curl http://localhost:9090/routes`
4. Fallback to vsock proxy: `unset VOLANT_DRIFT_ENDPOINT`

### Issue: Stack deployment hangs

**Symptom:** `volant stack deploy` never completes

**Fix:**
1. Check dependency order: `volant stack get <name>`
2. Look for circular dependencies in `depends_on`
3. Check VM logs: `volant vm logs <vm-name>`
4. Check for image build failures

### Issue: Migration conflict

**Symptom:** `UNIQUE constraint failed` during migration

**Fix:**
1. Check migration order (see Section 15)
2. Rebase feature branch: `git rebase main`
3. Renumber migrations if needed
4. Test: `./scripts/test-migrations.sh`

## 18.2 Debug Mode

**Enable verbose logging:**

```bash
# Volantd
VOLANT_LOG_LEVEL=debug volantd

# Driftd
driftd --log-level debug

# Fledged
fledged --log-level debug
```

## 18.3 Reset Everything

**Nuclear option:**

```bash
# Stop all processes
killall volantd driftd fledged

# Delete state
rm -rf /var/lib/volant
rm -f /tmp/volant.db

# Recreate bridge
ip link delete vbr0
ip link add vbr0 type bridge
ip addr add 192.168.127.1/24 dev vbr0
ip link set vbr0 up

# Restart
volantd --listen :7893
```

---

# 19. SUCCESS METRICS

## 19.1 Technical Metrics

**Must achieve:**
- [ ] Stack deployment: < 60s for 3-service stack
- [ ] DNS resolution: < 10ms
- [ ] Drift L4 latency: < 10μs (10x better than vsock)
- [ ] Volume I/O: > 500 MB/s
- [ ] Zero migration conflicts
- [ ] All integration tests pass
- [ ] > 80% code coverage

## 19.2 Feature Completeness

**Must implement:**
- [ ] `volant.yaml` manifest format
- [ ] Multi-service orchestration
- [ ] Dependency-ordered startup
- [ ] Service discovery (DNS)
- [ ] Persistent volumes
- [ ] Drift L4 packet routing
- [ ] Docker Compose converter
- [ ] Fledge server mode
- [ ] Web UI for stacks
- [ ] Health checks
- [ ] Rollback on failure

## 19.3 User Experience

**Must work:**
- [ ] `volant stack deploy volant.yaml` - one command deployment
- [ ] `curl http://<service>.volant` - DNS resolution
- [ ] `volant-compose -f docker-compose.yml` - Docker migration
- [ ] Web UI stack dashboard
- [ ] Automatic image building
- [ ] Volume persistence across restarts

---

# 20. POST-COMPLETION ROADMAP

**After Phase 3, future enhancements:**

1. **Authentication** (Week 6)
   - API key support
   - Multi-tenancy
   - Role-based access control

2. **Observability** (Week 7)
   - Prometheus metrics
   - Grafana dashboards
   - Distributed tracing

3. **High Availability** (Week 8)
   - Multi-node Volant cluster
   - Distributed volume replication
   - Leader election

4. **Advanced Networking** (Week 9)
   - Custom networks
   - Network policies (firewall rules)
   - Load balancing

5. **CI/CD Integration** (Week 10)
   - GitHub Actions plugin
   - GitLab CI integration
   - Automated testing

---

# CONCLUSION

This manifesto provides complete, self-contained documentation for building Docker Compose support into Volant using AI-assisted parallel development.

**Key Achievements:**
- ✅ 7 parallel development tracks designed
- ✅ Zero file conflicts by design
- ✅ Full implementation details with code
- ✅ Database migrations ordered correctly
- ✅ API contracts defined
- ✅ Testing strategy documented
- ✅ Worktree strategy explained
- ✅ 3-week timeline (vs 8 weeks sequential)

**Next Steps:**
1. ✅ Complete Phase 0 (Clean Boundaries)
2. Launch 4 parallel tracks (A, B, C, F)
3. Launch 3 parallel tracks (D, E, G)
4. Integration and testing
5. Release Volant v2.0 - The Docker Killer

---

**END OF MANIFESTO**

*Generated by Claude Code*  
*2025-01-15*

