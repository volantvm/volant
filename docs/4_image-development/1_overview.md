---
title: "Image Development Overview"
author: "VolantVM"
date: "2025-11-01"
---


This section is the canonical guide for building Volant images using Fledge. It explains when to choose initramfs vs. OCI Rootfs, how the configuration split works, and how fledge builds images that can be installed and instantiated as VMs.

## Terminology: Images, Not Plugins

Volant uses **images** as templates for creating VMs. An image defines:
- Boot media (initramfs or rootfs)
- Default runtime configuration (CPU, memory, environment)
- Workload entrypoint and networking
- Optional cloud-init, devices, and actions

Think of images like Docker images or AWS AMIs - reusable templates that can be instantiated many times with different configurations.

## References

- Fledge config: fledge/internal/config/schema.go, config.go
- Fledge builders: fledge/internal/builder/*.go
- Volant manifest: volant/internal/pluginspec/spec.go

## Choosing a Boot Strategy

**Initramfs** (fast boot, minimal, static payloads):
- Best for small stateless services or apps that can run from RAM
- Artifact: .cpio.gz
- Boot times: 50-150ms
- Memory: 10-20 MB + workload
- Use when: Performance is critical, workload is static

**OCI Rootfs** (run existing Docker/OCI images):
- Best for mature images with many dependencies
- Artifact: .img (ext4/xfs/btrfs/squashfs)
- Boot times: 2-5 seconds
- Memory: ~50 MB + workload
- Use when: Compatibility is needed, existing container images

See: [Quick Start: Initramfs](../2_getting-started/2_quick-start-initramfs.md) and [Quick Start: OCI Rootfs](../2_getting-started/3_quick-start-rootfs.md)

## Understanding the Configuration Split

Fledge uses **two configuration files** with distinct purposes:

### fledge.toml (Build-Time Configuration)

Defines **how to build** the image:
- `strategy`: "initramfs" or "oci_rootfs"
- `source`: Base image, Dockerfile, or BusyBox configuration
- `agent`: How to source the Kestrel agent
- `init`: Init mode (for initramfs)
- `filesystem`: Filesystem type and sizing (for oci_rootfs)
- `mappings`: Host files to include in the image

**This file is NOT included in the final image.** It's purely for the build process.

### manifest.toml (Runtime Defaults)

Defines **default runtime behavior** for VMs created from this image:
- `resources`: Default CPU cores and memory
- `workload`: Entrypoint, args, environment variables
- `network`: Mode, port exposures
- `actions`: Custom API endpoints
- `cloud_init`: VM initialization
- `devices`: PCI passthrough configuration

**This file IS merged into the final manifest.json** and defines defaults that can be overridden at VM creation time.

### The Build Workflow

```
Project Directory:
├── fledge.toml      (how to build)
├── manifest.toml    (runtime defaults)
└── Dockerfile       (optional)

↓ fledge build ↓

dist/:
├── manifest.json    (generated: merged result)
└── plugin.cpio.gz   (or rootfs.img)

↓ volar images install dist/manifest.json ↓

Image Registry:
- Image stored with manifest.json

↓ volar vms create myvm --plugin myimage ↓

Running VM:
- Uses manifest defaults
- Can override with --cpu, --memory, --env, --port flags
```

## Creating Images: Step by Step

### 1. Create Project Directory

```bash
mkdir my-image
cd my-image
```

### 2. Create fledge.toml

For initramfs:
```toml
version = "1"
strategy = "initramfs"

[agent]
source_strategy = "release"
version = "latest"

[mappings]
"./myapp" = "/usr/bin/myapp"
```

For OCI rootfs:
```toml
version = "1"
strategy = "oci_rootfs"

[agent]
source_strategy = "release"
version = "latest"

[source]
image = "docker://nginx:alpine"

[filesystem]
type = "squashfs"
size_buffer_mb = 100
```

### 3. Create manifest.toml

```toml
schema_version = "1.0"
name = "myapp"
version = "1.0.0"
runtime = "myapp"

[resources]
cpu_cores = 2
memory_mb = 1024

[workload]
entrypoint = "/usr/bin/myapp"
args = []

[workload.env]
PORT = "8080"
LOG_LEVEL = "info"

[[network.expose]]
port = 8080
protocol = "tcp"
```

### 4. Build

```bash
sudo fledge build
```

This reads both files and generates:
- `dist/manifest.json` (merged configuration)
- `dist/plugin.cpio.gz` or `dist/rootfs.img` (boot media)

### 5. Install and Run

```bash
# Install image to registry
volar images install dist/manifest.json

# Create VM with defaults
volar vms create demo --plugin myapp

# Or create with overrides
volar vms create prod --plugin myapp \
  --cpu 4 \
  --memory 2048 \
  --env LOG_LEVEL=debug

# List VMs
volar vms list
```

## Direct Build Mode (No Config Files)

You can build directly from a Dockerfile without fledge.toml:

```bash
sudo fledge build ./Dockerfile \
  --context . \
  --target runtime-stage \
  --build-arg APP_VERSION=1.0.0 \
  --output myapp

# For initramfs output:
sudo fledge build ./Dockerfile \
  --context . \
  --output-initramfs \
  --output myapp-initramfs
```

This generates manifest.json with sensible defaults, but you'll need to edit it or create a manifest.toml to customize runtime configuration.

## The Generated Manifest

Fledge merges fledge.toml + manifest.toml to produce manifest.json:

**From fledge.toml:**
- Boot media URLs and checksums (initramfs or rootfs)
- Image lineage (for OCI sources)
- Agent configuration details

**From manifest.toml:**
- All runtime defaults (resources, workload, network, etc.)

**You never edit manifest.json directly** - it's always generated by `fledge build`.

## Init Modes (Initramfs Only)

Initramfs supports three modes controlled by the `[init]` section of fledge.toml:

**Default mode** (no `[init]` section):
- C init mounts /proc, /sys, /dev, /tmp, /run
- Execs /bin/kestrel as PID 1
- Requires `[agent]` section
- Best for most use cases

**Custom mode** (`init.path` set):
- Your binary/script becomes PID 1
- Mapped to /init
- No kestrel agent
- You handle basic mounts

**None mode** (`init.none = true`):
- Your binary is PID 1
- Must mount /proc, /sys, /dev yourself
- Complete control, maximum responsibility

See details: [Initramfs Development Guide](2_initramfs.md)

## File Mappings

Use `[mappings]` in fledge.toml to place files inside the artifact following FHS conventions:

```toml
[mappings]
"./myapp" = "/usr/bin/myapp"              # Executable: 0755
"./lib/libfoo.so" = "/usr/lib/libfoo.so"  # Library: 0755
"./config.toml" = "/etc/myapp/config.toml" # Config: 0644
```

Fledge automatically sets permissions based on destination path:
- Executables under /usr/bin, /usr/sbin, /bin, /sbin → 0755
- Libraries under /lib, /usr/lib → 0755
- Others → 0644

## Configuration Override Hierarchy

When creating VMs, configuration is merged:

```
VM Creation Flags  >  manifest.toml defaults  >  System defaults
```

Example:

**manifest.toml:**
```toml
[resources]
cpu_cores = 2
memory_mb = 1024

[workload.env]
LOG_LEVEL = "info"
```

**VM creation:**
```bash
volar vms create prod --plugin myapp \
  --cpu 4 \
  --env LOG_LEVEL=debug \
  --env DATABASE_URL=prod.db.internal
```

**Result:**
- CPU: 4 (overridden)
- Memory: 1024 (from manifest)
- LOG_LEVEL: "debug" (overridden)
- DATABASE_URL: "prod.db.internal" (added)

This allows you to build images once and deploy them in different configurations without rebuilding.

## Common Patterns

### Development vs Production from One Image

```bash
# Build once
sudo fledge build
volar images install dist/manifest.json

# Development: minimal resources
volar vms create dev --plugin myapp \
  --cpu 1 \
  --memory 512 \
  --env LOG_LEVEL=trace

# Production: more resources
volar vms create prod --plugin myapp \
  --cpu 8 \
  --memory 8192 \
  --env LOG_LEVEL=error
```

### Multiple Instances from One Image

```bash
# Create multiple VMs with different ports
for i in {1..5}; do
  volar vms create web-$i --plugin myapp --port $((8080+i)):8080
done
```

### Blue-Green Deployment

```bash
# Build new version
sudo fledge build  # myapp:2.0.0
volar images install dist/manifest.json

# Create new instances
volar vms create app-v2-1 --plugin myapp:2.0.0
volar vms create app-v2-2 --plugin myapp:2.0.0

# Test, then remove old instances
volar vms delete app-v1-1 app-v1-2
```

## Next Steps

- [Initramfs Development Guide](2_initramfs.md) - Deep dive into initramfs images
- [OCI Rootfs Development Guide](3_oci-rootfs.md) - Deep dive into OCI-based images
- [fledge.toml Reference](../6_reference/2_fledge-toml.md) - Complete build config reference
- [manifest.toml Reference](../6_reference/3_manifest-toml.md) - Complete runtime config reference
- [Image Manifest Schema](../6_reference/1_manifest-schema.md) - manifest.json format
