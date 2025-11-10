# Volant Project Context

**Last Updated:** 2025-11-05

## Overview

Volant is a lightweight, secure microVM orchestration platform built on Cloud Hypervisor. It provides an image-first architecture where workloads are packaged as declarative manifests with embedded runtime logic, eliminating the need for complex control plane orchestration.

**Key Design Principles:**
- **Image-First:** Runtime logic lives in manifests, not control plane
- **Hardware Isolation:** Each workload runs in its own KVM-isolated microVM
- **Minimal Footprint:** Ultra-fast boot times (<100ms with initramfs)
- **Declarative:** Images define resources, networking, health checks, and API surface
- **Dual Boot Path:** Support for both initramfs (fast) and rootfs (OCI-compatible)

---

## Architecture Overview

### Core Components

1. **volantd (Fledge)** - Control plane daemon
   - VM lifecycle management (create, start, stop, destroy)
   - Image registry and artifact management
   - HTTP API for external clients
   - Deployment/replica set orchestration
   - Network management (bridge, IPAM, DNS)
   - SQLite persistence layer

2. **kestrel** - Guest agent (PID 1)
   - Minimal init system written in Go
   - Workload process supervision
   - HTTP/vsock API for host communication
   - Health check execution
   - Debug shell (optional)
   - Dual boot support (initramfs/rootfs)

3. **init.c** - Embedded C init
   - Early boot setup (mount proc/sys/dev)
   - Detects and execs kestrel from initramfs
   - Fallback rootfs mounting (squashfs/ext4/xfs)
   - Overlayfs support for read-only rootfs

4. **driftd** - L4 load balancer/NAT
   - eBPF TC-based dataplane (ingress/egress)
   - Port forwarding with stateful NAT
   - Vsock proxy support
   - Auto-detection of external interface
   - Map sharing between BPF programs

5. **volar** - CLI client
   - User-friendly commands for VM/image management
   - Proxies requests to volantd HTTP API

---

## Image-First Architecture

### Migration from Plugin-First

**Historical Context:**
- Originally used "plugin" terminology (see migration `/Users/marcxavier/Desktop/work/volant/internal/server/db/sqlite/migrations/0009_rename_plugins_to_images.sql`)
- Refactored to "image-first" architecture for clarity
- Images are now reusable templates; VMs are running instances

### Image Manifest Structure

**Location:** `/Users/marcxavier/Desktop/work/volant/internal/imagespec/spec.go`

```go
type Manifest struct {
    SchemaVersion string                  // "v1alpha1"
    Name          string                  // Unique image identifier
    Version       string                  // Semantic version
    Runtime       string                  // "firecracker" or "cloud-hypervisor"

    // Boot Media
    RootFS        RootFS                  // Optional: rootfs URL, checksum, format
    Initramfs     Initramfs               // Optional: initramfs URL, checksum
    Disks         []Disk                  // Additional disks (e.g., data volumes)

    // Resource Defaults (overridable per-VM)
    Resources     ResourceSpec            // cpu_cores, memory_mb

    // Runtime Configuration
    Workload      Workload                // Type, entrypoint, env vars
    Actions       map[string]Action       // API surface exposed by image
    HealthCheck   HealthCheck             // HTTP probe configuration

    // Optional Features
    CloudInit     *CloudInit              // User-data, meta-data, network-config
    Network       *NetworkConfig          // Mode (vsock/bridged/dhcp)
    Devices       *DeviceConfig           // PCI passthrough configuration

    // Metadata
    Enabled       bool                    // Toggle for deployment eligibility
    OpenAPI       string                  // OpenAPI spec URL (future)
    Labels        map[string]string       // Arbitrary key-value pairs
}
```

### Resource Definition

**Per-VM Override Model:**
- Images define **default** resources in `ResourceSpec`
- VMs can override at creation time via `VMConfig`
- Validation occurs at both image install and VM creation
- Enforces minimum/maximum bounds (e.g., 1-64 cores, 128MB-64GB memory)

**Example:**
```yaml
# Image manifest defines defaults
resources:
  cpu_cores: 2
  memory_mb: 512

# VM creation overrides
POST /api/v1/vms
{
  "name": "web-prod",
  "image": "nginx:1.0.0",
  "resources": {
    "cpu_cores": 4,      # Override: 4 cores instead of 2
    "memory_mb": 2048    # Override: 2GB instead of 512MB
  }
}
```

**File References:**
- Default resources: `/Users/marcxavier/Desktop/work/volant/internal/imagespec/spec.go:103-106`
- VM overrides: `/Users/marcxavier/Desktop/work/volant/internal/server/orchestrator/vmconfig/config.go:22-44`
- Merge logic: `/Users/marcxavier/Desktop/work/volant/internal/server/orchestrator/orchestrator.go:196-225`

### Environment Variables

**Implementation:**
- Stored in `Workload.Env` as `map[string]string`
- Encoded as base64 JSON in kernel cmdline (`volant.env` parameter)
- Decoded by kestrel on boot from `/proc/cmdline`
- Set via `os.Setenv()` before workload execution

**Flow:**
1. User defines env vars in image manifest or VM config
2. volantd merges manifest env + VM override env
3. Encoded as `volant.env=<base64(json)>` in kernel cmdline
4. kestrel reads `/proc/cmdline`, decodes, sets environment
5. Workload process inherits environment

**Example:**
```yaml
# Image manifest
workload:
  type: exec
  entrypoint: /app/server
  env:
    LOG_LEVEL: info
    PORT: "8080"

# VM override
env:
  LOG_LEVEL: debug      # Override
  DATABASE_URL: "..."   # Add new
```

**File References:**
- Manifest env: `/Users/marcxavier/Desktop/work/volant/internal/imagespec/spec.go:172-176`
- Kernel cmdline encoding: `/Users/marcxavier/Desktop/work/volant/internal/server/orchestrator/orchestrator.go:400-420`
- Agent decoding: `/Users/marcxavier/Desktop/work/volant/internal/agent/app/app.go:505-547`

### Workload Types

**Supported Types:**
1. **exec** - Simple executable
   - `entrypoint`: Path to binary
   - `args`: Command-line arguments
   - `env`: Environment variables
   - `working_dir`: Working directory

2. **http** - HTTP service
   - `base_url`: Base URL for health checks
   - `entrypoint`: Path to binary
   - `env`: Environment variables
   - Waits for health check before marking ready

3. **grpc** - gRPC service (future support)

**File Reference:** `/Users/marcxavier/Desktop/work/volant/internal/imagespec/spec.go:165-188`

### Actions (API Surface)

**Concept:**
Images can expose custom HTTP endpoints via the `actions` map. Requests are proxied through volantd → kestrel → workload.

**Example:**
```yaml
actions:
  update:
    method: POST
    path: /api/v1/update
    description: Update application configuration
```

**Request Flow:**
```
Client → POST /api/v1/vms/web-0/actions/nginx/update
  → volantd resolves image manifest
  → Proxies to VM's kestrel agent
  → Kestrel forwards to workload
  → Response flows back
```

**File References:**
- Action definition: `/Users/marcxavier/Desktop/work/volant/internal/imagespec/spec.go:193-199`
- Proxy logic: `/Users/marcxavier/Desktop/work/volant/internal/server/httpapi/httpapi.go:312-350`

---

## Fledge (volantd) - Control Plane

### Configuration

**Environment Variables:**

| Variable | Default | Description |
|----------|---------|-------------|
| `VOLANT_DB_PATH` | `~/.volant/state.db` | SQLite database location |
| `VOLANT_API_LISTEN` | `0.0.0.0:7777` | API listen address |
| `VOLANT_API_ADVERTISE` | Auto-detected | Advertise address for VMs |
| `VOLANT_BRIDGE` | `vbr0` | Bridge interface name |
| `VOLANT_SUBNET` | `192.168.127.0/24` | Subnet CIDR |
| `VOLANT_HOST_IP` | `192.168.127.1` | Host IP on bridge |
| `VOLANT_KERNEL_BZIMAGE` | Auto-detected | Path to bzImage kernel |
| `VOLANT_KERNEL_VMLINUX` | Auto-detected | Path to vmlinux kernel |
| `VOLANT_RUNTIME_DIR` | `~/.volant/run` | Runtime directory (sockets, PIDs) |
| `VOLANT_LOG_DIR` | `~/.volant/logs` | Log directory |
| `VOLANT_DRIFT_ENDPOINT` | None | Drift L4 LB endpoint |
| `VOLANT_DNS_ENABLED` | `true` | Enable DNS server |
| `VOLANT_API_KEY` | None | Optional API authentication |
| `VOLANT_CORS_ORIGINS` | None | CORS allowed origins |

**File Reference:** `/Users/marcxavier/Desktop/work/volant/internal/server/config/config.go`

### HTTP API Endpoints

**VM Lifecycle:**
- `POST /api/v1/vms` - Create VM
- `GET /api/v1/vms` - List VMs
- `GET /api/v1/vms/:name` - Get VM details
- `POST /api/v1/vms/:name/start` - Start VM
- `POST /api/v1/vms/:name/stop` - Stop VM
- `POST /api/v1/vms/:name/restart` - Restart VM
- `DELETE /api/v1/vms/:name` - Delete VM

**Agent Proxying:**
- `GET /api/v1/vms/:name/agent/*path` - Proxy to guest agent
- `POST /api/v1/vms/:name/actions/:image/:action` - Execute image action

**Image Management:**
- `POST /api/v1/images` - Install image
- `GET /api/v1/images` - List images
- `GET /api/v1/images/:name` - Get image manifest
- `DELETE /api/v1/images/:name` - Remove image
- `PATCH /api/v1/images/:name/toggle` - Enable/disable image

**Deployments:**
- `POST /api/v1/deployments` - Create deployment
- `GET /api/v1/deployments` - List deployments
- `PATCH /api/v1/deployments/:name/scale` - Scale replicas

**Events:**
- `GET /api/v1/events/vms` - Server-Sent Events stream

**File Reference:** `/Users/marcxavier/Desktop/work/volant/internal/server/httpapi/httpapi.go`

### Orchestrator Engine

**Key Responsibilities:**

1. **VM Creation:**
   - Validate image manifest
   - Merge manifest defaults with VM overrides
   - Allocate IP/CID from pool
   - Create tap device (bridged/dhcp modes)
   - Prepare cloud-init seed (if configured)
   - Build kernel cmdline with encoded manifest/env
   - Download boot media (initramfs/rootfs)
   - Launch Cloud Hypervisor process
   - Store runtime state in SQLite
   - Publish VM events

2. **VM Lifecycle:**
   - Start: Launch Cloud Hypervisor with stored config
   - Stop: SIGTERM → SIGKILL, cleanup tap devices
   - Restart: Atomic stop + start
   - Destroy: Release IP, delete DB records, cleanup cloud-init, unbind VFIO

3. **Networking:**
   - Auto-detect or configure network mode
   - IPAM for managed subnet
   - MAC derivation from VM name + IP
   - Bridge attachment (Linux) or noop (other platforms)

4. **Deployments:**
   - Replica set management (desired vs ready)
   - Naming: `<deployment>-<n>` (e.g., `web-0`, `web-1`)
   - Reconciliation loop for scaling
   - Atomic updates via DB transactions

**File Reference:** `/Users/marcxavier/Desktop/work/volant/internal/server/orchestrator/orchestrator.go`

### Data Storage

**SQLite Schema:**

| Table | Purpose |
|-------|---------|
| `vms` | Runtime state (ID, name, status, PID, IP, CID, resources) |
| `vm_configs` | Current configuration snapshots |
| `vm_config_history` | Versioned configuration history |
| `vm_groups` | Deployments (desired replicas, base config) |
| `images` | Installed manifests (name, version, enabled, metadata JSON) |
| `ip_allocations` | Simple IPAM for managed subnet |
| `image_artifacts` | Kernel/initramfs/rootfs artifacts per image |

**Transaction Handling:**
All state mutations wrapped in `WithTx()` for atomicity.

**File References:**
- Schema: `/Users/marcxavier/Desktop/work/volant/internal/server/db/sqlite/schema.sql`
- Repository: `/Users/marcxavier/Desktop/work/volant/internal/server/db/sqlite/`

---

## Kestrel - Guest Agent (PID 1)

### Dual Boot Architecture

**Boot Modes:**

1. **Initramfs Mode (Default):**
   - C init detects `/bin/kestrel` in initramfs
   - Immediately execs kestrel (no pivot)
   - Ultra-fast boot (<100ms)
   - Minimal footprint (no rootfs required)

2. **Rootfs Mode (OCI-Compatible):**
   - C init mounts rootfs from block device
   - Supports squashfs, ext4, xfs, btrfs
   - Overlayfs for read-only rootfs + writable upper layer
   - Pivot via `switch_root`, re-exec kestrel
   - Boot time: 2-5 seconds

**Boot Flow:**

```
Cloud Hypervisor starts
  ↓
Linux kernel with embedded initramfs
  ↓
C init (/init)
  ↓
Mount proc, sys, dev
  ↓
Check for /bin/kestrel
  ↓
┌─────────────┬─────────────┐
│  Found      │  Not Found  │
│  (initramfs)│  (rootfs)   │
└─────────────┴─────────────┘
       ↓              ↓
    exec         Mount rootfs
   kestrel      (squashfs/ext4)
                     ↓
                 Overlayfs
                 (if read-only)
                     ↓
                switch_root
                     ↓
                  exec
                 kestrel
```

**File References:**
- C init: `/Users/marcxavier/Desktop/work/volant/build/init.c`
- Go agent: `/Users/marcxavier/Desktop/work/volant/cmd/kestrel/main.go`
- PID 1 bootstrap: `/Users/marcxavier/Desktop/work/volant/internal/agent/app/pid1.go`
- Main application: `/Users/marcxavier/Desktop/work/volant/internal/agent/app/app.go`

### PID 1 Bootstrap

**Linux-Specific Initialization:**

**Stage 1: Initramfs Boot**
- Mount initial filesystems (proc, sys, dev)
- Detect boot mode via `volant.boot` kernel parameter
- If rootfs specified:
  - Mount squashfs/ext4/xfs/btrfs from `/dev/vda`
  - Create overlayfs if read-only (tmpfs upper layer)
  - Pivot via `switch_root`
  - Re-exec as Stage 2

**Stage 2: Post-Pivot or Initramfs**
- Mount essential filesystems (proc, sys, dev, devpts, shm, run, tmp)
- Set up `/dev/console` as controlling terminal
- Start dbus-daemon if available
- Spawn zombie reaper goroutine (reap orphaned processes)
- Handle SIGTERM/SIGINT for graceful shutdown

**File Reference:** `/Users/marcxavier/Desktop/work/volant/internal/agent/app/pid1.go`

### Agent Features

1. **Manifest Resolution:**
   - Decode from kernel cmdline (`volant.manifest` parameter)
   - Fallback: Fetch from host API via `volant.api_addr`
   - Gzip + base64 decoding

2. **Environment Setup:**
   - Read `volant.env` from cmdline
   - Decode base64 JSON
   - Set via `os.Setenv()`
   - Merge with workload manifest env

3. **DNS Configuration:**
   - Read `volant.dns_server` and `volant.dns_search` from cmdline
   - Write `/etc/resolv.conf`

4. **Workload Management:**
   - Start workload process with correct env/args/cwd
   - Monitor for crashes, restart if configured
   - Health check execution (HTTP probes)
   - Mark ready when healthy

5. **HTTP API:**
   - **TCP Listener:** `:8080` (bridged/dhcp modes)
   - **Vsock Listener:** Port 8080 (vsock mode)
   - Routes:
     - `GET /healthz` - Agent health check
     - `GET /v1/*` - Proxy to workload (if HTTP workload type)

6. **Debug Shell:**
   - Optional serial console shell on `/dev/ttyS0`
   - Enabled via `volant.debug_shell=true` kernel parameter

**File Reference:** `/Users/marcxavier/Desktop/work/volant/internal/agent/app/app.go`

---

## Driftd - L4 Load Balancer/NAT

### Architecture

**Components:**
1. **Route Store:** JSON file persistence for port mappings
2. **Dataplane Manager:** eBPF program lifecycle (Linux-only)
3. **Vsock Proxy Manager:** Userspace proxy for vsock routes
4. **Controller:** Coordinates dataplane + storage
5. **HTTP API:** REST interface for route management (port 9090)

### Configuration

**Environment Variables:**

| Variable | Default | Description |
|----------|---------|-------------|
| `DRIFT_HTTP_LISTEN` | `0.0.0.0:9090` | API listen address |
| `DRIFT_METRICS_LISTEN` | `127.0.0.1:9091` | Metrics listen address |
| `DRIFT_BRIDGE` | `vbr0` | Bridge interface name |
| `DRIFT_EXTERNAL_IF` | `auto` | External interface (auto-detect) |
| `DRIFT_STATE_DIR` | `~/.volant/drift` | State directory |
| `DRIFT_ROUTES_PATH` | `<state_dir>/routes.json` | Route storage path |
| `DRIFT_BPF_OBJECT` | `drift_l4` | Base path for BPF objects |
| `DRIFT_API_KEY` | None | Optional API authentication |

**Auto-Detection:**
- External interface auto-detected by reading `/proc/net/route` for default route (0.0.0.0)
- Fallback: Uses bridge interface if detection fails
- Gracefully disables on non-Linux platforms

**File Reference:** `/Users/marcxavier/Desktop/work/volant/internal/drift/config/config.go`

### eBPF TC Dataplane

**Critical Design: Map Sharing**

Ingress and egress programs **must** share `conntrack` and `stats` maps for stateful NAT:

1. Load ingress collection first
2. Extract `conntrack` and `stats` maps from ingress
3. Load egress as `*ebpf.CollectionSpec`
4. **Rewrite egress spec** to use ingress maps via `RewriteMaps()`
5. Load egress collection from modified spec

**Example:**
```go
// Load ingress first
ingressColl, _ := ebpf.LoadCollection(ingressPath)
portmap := ingressColl.Maps["portmap"]
conntrack := ingressColl.Maps["conntrack"]
stats := ingressColl.Maps["stats"]

// Load egress as spec
egressSpec, _ := ebpf.LoadCollectionSpec(egressPath)

// CRITICAL: Rewrite to share maps
egressSpec.RewriteMaps(map[string]*ebpf.Map{
    "conntrack": conntrack,
    "stats":     stats,
})

// Now load egress with shared maps
egressColl, _ := ebpf.NewCollectionWithOptions(egressSpec, ebpf.CollectionOptions{})
```

**File Reference:** `/Users/marcxavier/Desktop/work/volant/internal/drift/dataplane/manager_linux.go:75-96`

### TC Attachment

**Both ingress and egress attached to external interface:**

```go
// Ingress: TC ingress on external interface
ingressLink, _ := link.AttachTCX(link.TCXOptions{
    Program:   ingressProg,
    Interface: extIfaceIndex,  // eth0 (not bridge!)
    Attach:    ebpf.AttachTCXIngress,
})

// Egress: TC egress on external interface
egressLink, _ := link.AttachTCX(link.TCXOptions{
    Program:   egressProg,
    Interface: extIfaceIndex,  // eth0 (forwarded packets bypass bridge TC)
    Attach:    ebpf.AttachTCXEgress,
})
```

**Rationale:**
- Forwarded packets bypass bridge TC hooks
- Must intercept on external interface to catch traffic to/from VMs

**File Reference:** `/Users/marcxavier/Desktop/work/volant/internal/drift/dataplane/manager_linux.go:128-152`

### Ingress Program (DNAT)

**Flow:**
1. Parse Ethernet → IP → TCP/UDP headers
2. Lookup `(proto, dest_port)` in `portmap` BPF map
3. If match:
   - Save original dst IP/port
   - Rewrite packet destination to backend IP/port
   - Update L3/L4 checksums
   - Create conntrack entry for reverse NAT
   - Increment stats

**Byte Order:**
- IP addresses stored in **little-endian (host byte order)** in map
- On x86, memory representation already correct for network byte order
- No conversion needed when writing to packet
- Conntrack key uses network byte order for egress lookup

**File Reference:** `/Users/marcxavier/Desktop/work/volant/internal/drift/bpf/drift_l4_ingress.c`

### Egress Program (SNAT)

**Flow:**
1. Parse return packet (src=VM, dst=Client)
2. Build conntrack key:
   - `src_ip = packet.dst_ip` (client IP)
   - `dst_ip = packet.src_ip` (VM IP)
   - `src_port = packet.dst_port` (client port)
   - `dst_port = packet.src_port` (VM port)
3. Lookup conntrack to get original host IP/port
4. Rewrite source to original host IP/port (reverse NAT)
5. Update checksums
6. Increment stats

**Map Definitions:**

```c
// Conntrack: LRU hash, 65536 entries (SHARED with ingress)
struct conntrack_key {
    __be32 src_ip;   // Client IP (network byte order)
    __be32 dst_ip;   // Backend IP (network byte order)
    __be16 src_port; // Client port (network byte order)
    __be16 dst_port; // Backend port (network byte order)
    __u8 proto;
};

struct conntrack_value {
    __be32 orig_dst_ip;   // Original host IP
    __be16 orig_dst_port; // Original host port
    __u64 last_seen;
};

// Stats: Per-CPU array, 4 entries (SHARED with ingress)
// [0] = ingress packets
// [1] = ingress bytes
// [2] = egress packets
// [3] = egress bytes
```

**File Reference:** `/Users/marcxavier/Desktop/work/volant/internal/drift/bpf/drift_l4_egress.c`

### Controller

**Route Management:**
- `Upsert()`: Validate, apply to dataplane, persist to storage
- `Delete()`: Remove from dataplane, delete from storage
- `Restore()`: Replay persisted routes on startup
- `Stats()`: Aggregate per-CPU counters

**Backend Types:**
1. **Bridge:** TC eBPF dataplane, destination IP + port
2. **Vsock:** Userspace proxy (socat/gvisor-tap-vsock), CID + port

**File Reference:** `/Users/marcxavier/Desktop/work/volant/internal/drift/controller/controller.go`

---

## Networking

### Network Modes

1. **vsock (Default):**
   - No IP allocated
   - Communication via vsock only
   - Isolated from network
   - Lowest attack surface

2. **bridged:**
   - Static IP from managed subnet
   - Tap device attached to bridge
   - Host manages routing
   - MAC derived from VM name + IP

3. **dhcp:**
   - Tap device created, no IP allocation
   - Guest manages DHCP client
   - Flexible for OCI images with DHCP support

**File References:**
- Network manager: `/Users/marcxavier/Desktop/work/volant/internal/server/orchestrator/network/bridge.go`
- Mode resolution: `/Users/marcxavier/Desktop/work/volant/internal/server/orchestrator/orchestrator.go:285-310`

### IPAM

**Simple Subnet Allocation:**
- SQLite table: `ip_allocations` (ip, vm_id)
- Linear scan for next available IP
- Validation: No duplicates, within subnet bounds
- Release on VM deletion

**File Reference:** `/Users/marcxavier/Desktop/work/volant/internal/server/db/sqlite/db.go`

### DNS

**Optional DNS Server:**
- Embedded DNS server on port 53 (UDP/TCP)
- A records: `<vm-name>.volant.local` → VM IP
- Configured via `VOLANT_DNS_ENABLED` (default: true)
- Pushed to VMs via `volant.dns_server` kernel parameter

**File Reference:** `/Users/marcxavier/Desktop/work/volant/internal/server/orchestrator/dns/dns.go`

---

## Build System

### Makefile Targets

**Common Targets:**
- `make build` - Build all binaries (volantd, volar, kestrel, driftd)
- `make install` - Install volantd, volar, and systemd units
- `make install-drift` - Install driftd and systemd unit
- `make kernel` - Build Linux kernel (bzImage + vmlinux)
- `make initramfs` - Build initramfs with kestrel + init.c
- `make test` - Run unit tests
- `make clean` - Remove build artifacts

### Kernel Compilation

**Targets:**
- **bzImage:** Compressed kernel (default for production)
- **vmlinux:** Uncompressed ELF kernel (debug/development)

**Embedded Initramfs:**
Both kernels contain the same embedded initramfs:
- `init.c` compiled to `/init`
- `kestrel` binary at `/bin/kestrel`
- Essential `/bin`, `/dev`, `/proc`, `/sys` directories

**File Reference:** `/Users/marcxavier/Desktop/work/volant/Makefile`

### BPF Object Compilation

**Ingress/Egress Programs:**
```bash
clang -O2 -g -target bpf \
  -D__TARGET_ARCH_x86 \
  -I/usr/include/bpf \
  -Iinternal/drift/bpf \
  -c internal/drift/bpf/drift_l4_ingress.c \
  -o internal/drift/bpf/bin/drift_l4_ingress.bpf.o

llvm-strip -g internal/drift/bpf/bin/drift_l4_ingress.bpf.o
```

**Installation:**
- BPF objects installed to `/usr/local/bin/drift_l4_{ingress,egress}.bpf.o`
- Manager resolves path from `DRIFT_BPF_OBJECT` environment variable

**File Reference:** `/Users/marcxavier/Desktop/work/volant/Makefile:85-92`

---

## Key Design Decisions

### Why Image-First?

**Traditional Orchestration:**
- Control plane logic (autoscaling, health checks, routing)
- Complex state machines
- Centralized decision-making

**Volant Image-First:**
- Runtime logic embedded in manifests
- Control plane is "dumb" orchestrator
- VMs self-report health, expose actions
- Simplifies control plane, increases portability

### Why Dual Boot?

**Initramfs Mode:**
- Ultra-fast boot (<100ms)
- Minimal attack surface (no rootfs)
- Perfect for stateless workloads
- Smaller memory footprint

**Rootfs Mode:**
- OCI image compatibility
- Richer userspace (systemd, package managers)
- Persistent writable layer (overlayfs)
- Familiar for Docker/Podman users

### Why eBPF for L4 NAT?

**Alternatives Considered:**
- iptables/nftables: Userspace overhead, complex rules
- IPVS: Limited to load balancing, no custom logic
- Userspace proxy: Context switches, poor performance

**eBPF TC Advantages:**
- Kernel-space packet processing (zero userspace overhead)
- Programmable: Custom NAT logic, stats collection
- High performance: Line-rate packet processing
- Stateful: Conntrack for bidirectional flows
- Modern: TCX API, map sharing, verifier safety

### Why SQLite?

**Requirements:**
- Single-node control plane (no distributed consensus)
- Simple schema (VMs, images, routes)
- Atomic transactions for state changes
- Embedded (no external database server)

**SQLite Strengths:**
- ACID transactions
- Lightweight, zero configuration
- Excellent performance for read-heavy workloads
- Proven reliability (billions of deployments)

---

## Security Model

### Isolation Layers

1. **Hardware Virtualization (KVM):**
   - Each VM runs in isolated address space
   - No shared memory between VMs
   - Kernel-enforced boundary

2. **Minimal Attack Surface:**
   - No SSH/shell access by default
   - Optional debug shell (disabled in production)
   - vsock-only communication (no network exposure)

3. **Least Privilege:**
   - Kestrel runs as PID 1 but drops capabilities after boot
   - Workload processes run as non-root (if configured)
   - driftd requires only `CAP_BPF`, `CAP_NET_ADMIN`, `CAP_NET_RAW`

4. **Immutable Infrastructure:**
   - Read-only rootfs with overlayfs
   - Ephemeral writable layer (discarded on restart)
   - Declarative configuration (no manual changes)

### API Authentication

**Optional API Key:**
- `VOLANT_API_KEY` environment variable
- Bearer token authentication
- IP filtering via `VOLANT_API_ALLOW_CIDR`
- CORS restrictions

**File Reference:** `/Users/marcxavier/Desktop/work/volant/internal/server/httpapi/middleware.go`

---

## Development Workflow

### Local Development

1. **Build All Components:**
   ```bash
   make build
   ```

2. **Build Kernel + Initramfs:**
   ```bash
   make kernel initramfs
   ```

3. **Install System-Wide:**
   ```bash
   sudo make install
   sudo make install-drift
   ```

4. **Start Services:**
   ```bash
   sudo systemctl start volantd
   sudo systemctl start driftd
   ```

5. **Create VM:**
   ```bash
   volar image install examples/nginx.yaml
   volar vm create nginx-test --image nginx:latest
   ```

### Testing

**Unit Tests:**
```bash
make test
```

**Integration Tests:**
```bash
# TODO: Integration test suite
```

### Debugging

**VM Console Access:**
```bash
# Via Cloud Hypervisor serial console
screen /var/run/cloud-hypervisor/<vm-name>/console
```

**BPF Program Debugging:**
```bash
# List loaded programs
bpftool prog list | grep drift_l4

# Dump map contents
bpftool map dump id <map_id>

# View stats
curl localhost:9090/stats
```

**Agent Logs:**
```bash
# Via VM serial console (if debug_shell enabled)
journalctl -u kestrel
```

---

## File Structure

```
volant/
├── cmd/
│   ├── volantd/           # Control plane daemon
│   ├── volar/             # CLI client
│   ├── kestrel/           # Guest agent
│   └── driftd/            # L4 load balancer
├── internal/
│   ├── server/            # Control plane logic
│   │   ├── httpapi/       # HTTP API handlers
│   │   ├── orchestrator/  # VM lifecycle management
│   │   ├── images/        # Image registry
│   │   ├── db/            # SQLite persistence
│   │   └── config/        # Configuration
│   ├── agent/             # Guest agent logic
│   │   └── app/           # PID 1, workload management
│   ├── drift/             # L4 load balancer
│   │   ├── bpf/           # eBPF programs (C)
│   │   ├── dataplane/     # BPF program management
│   │   └── controller/    # Route management
│   ├── imagespec/         # Manifest schema
│   └── shared/            # Common utilities
├── build/
│   ├── init.c             # Embedded C init
│   ├── linux/             # Kernel config + patches
│   └── systemd/           # Systemd units
├── examples/              # Example image manifests
└── docs/                  # Documentation
```

---

## Common Patterns

### VM Creation Pattern

```go
// 1. Validate manifest
manifest, err := registry.Get(imageName)

// 2. Merge defaults + overrides
vmConfig := mergeConfig(manifest, userOverrides)

// 3. Allocate resources
ip, err := ipam.Allocate()
cid := allocateCID()

// 4. Prepare boot media
initramfs := downloadInitramfs(manifest.Initramfs.URL)
rootfs := downloadRootfs(manifest.RootFS.URL)

// 5. Build kernel cmdline
cmdline := buildCmdline(manifest, vmConfig, ip, cid)

// 6. Launch hypervisor
pid := launchCloudHypervisor(cmdline, initramfs, rootfs)

// 7. Store state
db.SaveVM(vm)
```

### Image Install Pattern

```go
// 1. Parse manifest
manifest, err := parseManifest(yamlBytes)

// 2. Validate schema
err = validate(manifest)

// 3. Store in registry
registry.Register(manifest)

// 4. Persist to database
db.SaveImage(manifest)

// 5. Download artifacts (lazy)
// Artifacts downloaded on first VM creation
```

### Route Forwarding Pattern

```go
// 1. Validate route
err := validateRoute(proto, hostPort, destIP, destPort)

// 2. Apply to dataplane
err = dataplane.ApplyBridge(proto, hostPort, destIP, destPort)

// 3. Persist to storage
err = store.Save(route)

// 4. Stats collection
stats, err := dataplane.Stats()
```

---

## Troubleshooting

### Common Issues

**1. VM fails to boot:**
- Check kernel cmdline: `ps aux | grep cloud-hypervisor`
- Verify kernel/initramfs paths: `ls -la ~/.volant/kernels/`
- Check serial console: `screen /var/run/cloud-hypervisor/<vm>/console`

**2. Network connectivity issues:**
- Verify bridge exists: `ip link show vbr0`
- Check IP allocation: `volar vm get <name>`
- Ping from host: `ping <vm-ip>`
- Check routing: `ip route | grep vbr0`

**3. Port forwarding not working:**
- Verify driftd is running: `systemctl status driftd`
- Check BPF programs: `bpftool prog list | grep drift`
- Check routes: `curl localhost:9090/routes`
- Tcpdump on bridge: `tcpdump -i vbr0 -n port <port>`

**4. eBPF verifier errors:**
- Check dmesg: `dmesg | grep -i bpf`
- Verify kernel version: `uname -r` (requires 5.10+)
- Check capabilities: `getcap /usr/local/bin/driftd`

---

## Future Roadmap

### Planned Features

1. **Image Registry:**
   - OCI-compatible image distribution
   - Remote registry support (pull from external registries)
   - Image signing/verification

2. **Advanced Networking:**
   - Multi-bridge support
   - VXLAN overlays
   - IPv6 support

3. **Storage:**
   - Persistent volume support
   - 9p/virtiofs mounts
   - Snapshot/restore

4. **Observability:**
   - Prometheus metrics export
   - Distributed tracing (OpenTelemetry)
   - Structured logging

5. **Security:**
   - SEV/TDX support (confidential computing)
   - Pod security policies
   - Network policies

6. **Scaling:**
   - Horizontal pod autoscaler
   - Resource quotas
   - Multi-tenant isolation

---

## References

### Key Documentation

- Architecture Overview: `/Users/marcxavier/Desktop/work/volant/docs/5_architecture/1_overview.md`
- Components: `/Users/marcxavier/Desktop/work/volant/docs/5_architecture/2_components.md`
- Networking: `/Users/marcxavier/Desktop/work/volant/docs/5_architecture/4_networking.md`
- Boot and Runtime: `/Users/marcxavier/Desktop/work/volant/docs/5_architecture/5_boot-and-runtime.md`

### External Dependencies

- **Cloud Hypervisor:** VMM for KVM-based microVMs
- **Linux Kernel:** Custom-built with embedded initramfs
- **eBPF/Cilium:** BPF program management library
- **SQLite:** Embedded database for state persistence
- **Go:** Primary implementation language
- **Clang/LLVM:** BPF program compilation

---

**Document Version:** 1.0.0
**Last Verified Against Codebase:** 2025-11-05
**Next Review:** When major architectural changes occur
