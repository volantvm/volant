---
title: "OCI Rootfs Image Development"
author: "VolantVM"
date: "2025-11-01"
---


Run existing Docker/OCI images as microVMs with a read/write root filesystem. This path provides compatibility with existing container images while adding true hardware isolation and deterministic networking.

Ground truth:
- fledge/internal/config/schema.go (FilesystemConfig, SourceConfig)
- fledge/internal/builder/oci_rootfs.go (build pipeline)
- fledge/internal/builder/mapping.go (FHS-aware placement and permissions)
- volant/internal/imagespec/spec.go (manifest)
- volant/internal/server/orchestrator/orchestrator.go (boot media resolution)

## When to Use

- You already have a mature container image
- You need package managers, dynamic linking, or larger dependency trees
- Compatibility with existing Docker/OCI workflows
- Migration from containers to microVMs

## Configuration Files

OCI rootfs images require two configuration files:

### fledge.toml (Build Configuration)

Defines how to build the rootfs image.

### manifest.toml (Runtime Defaults)

Defines default resources, environment, and workload configuration for VMs created from this image.

## Minimal Configuration

**fledge.toml:**
```toml
version = "1"
strategy = "oci_rootfs"

[agent]
source_strategy = "release"
version = "latest"

[source]
image = "docker://nginx:alpine"

[filesystem]
type = "squashfs"           # Default: compressed read-only with overlay
compression_level = 15      # 1-22, default 15 (balanced compression)
overlay_size = "1G"         # tmpfs size for runtime writes
```

**manifest.toml:**
```toml
schema_version = "v1"
name = "nginx"
version = "0.1.0"
runtime = "nginx"

# Resources section is OPTIONAL - omit to use defaults (cpu_cores=2, memory_mb=2048)
[resources]
cpu_cores = 2
memory_mb = 1024

[workload]
type = "exec"  # Can be "exec", "http", or "grpc"
entrypoint = ["/docker-entrypoint.sh", "nginx", "-g", "daemon off;"]

[env]
NGINX_PORT = "80"

[[network.expose]]
port = 80
protocol = "tcp"
```

## Build

```bash
sudo fledge build
# → outputs dist/nginx.img and dist/manifest.json
```

## What Fledge Does

1. Fetch image layers or build locally via embedded BuildKit
2. Unpack layers with umoci into an intermediate rootfs
3. Optionally extract OCI config to /etc/fsify-entrypoint for introspection
4. Install kestrel agent to /bin/kestrel (when agent configured)
5. Apply file mappings and permissions following FHS
6. Create filesystem image:
   - Squashfs (default): mksquashfs with compression
   - Legacy (ext4/xfs/btrfs): mkfs, mount via loop, copy rootfs, optionally shrink
8. Generate manifest.json with boot media URLs and checksums

## Install and Run

```bash
# Install image to registry
volar images install dist/manifest.json

# Create VM with defaults
volar vms create web --image nginx

# Or create with overrides
volar vms create prod --image nginx \
  --cpu 4 \
  --memory 2048 \
  --env NGINX_PORT=8080 \
  --port 8080:80

# List VMs
volar vms list

# Check VM details
volar vms show web
```

## Using a Dockerfile

You can swap `source.image` for a Dockerfile build by providing `source.dockerfile`:

**fledge.toml:**
```toml
version = "1"
strategy = "oci_rootfs"

[agent]
source_strategy = "release"
version = "latest"

[source]
dockerfile = "./Dockerfile"
context = "."
target = "runtime-stage"  # optional multi-stage target

[source.build_args]
APP_VERSION = "1.0.0"
NODE_ENV = "production"

[filesystem]
type = "squashfs"
compression_level = 15
overlay_size = "1G"
```

**manifest.toml:**
```toml
schema_version = "v1"
name = "myapp"
version = "1.0.0"
runtime = "myapp"

# Resources section is OPTIONAL - omit to use defaults (cpu_cores=2, memory_mb=2048)
[resources]
cpu_cores = 2
memory_mb = 1024

[workload]
type = "exec"  # Can be "exec", "http", or "grpc"
entrypoint = ["/usr/local/bin/node", "/app/server.js"]

[env]
NODE_ENV = "production"
PORT = "3000"
LOG_LEVEL = "info"

[[network.expose]]
port = 3000
protocol = "tcp"
```

## Direct Dockerfile Build (No Config Files)

Use the embedded BuildKit workflow without writing a config file:

```bash
sudo fledge build ./Dockerfile \
  --context . \
  --target runtime-stage \
  --build-arg FOO=bar \
  --output custom-rootfs
# → outputs custom-rootfs.img + custom-rootfs.manifest.json
```

`--output` overrides the artifact prefix. Without it, Fledge derives a name from the context directory.

Note: You'll need to manually create a manifest.toml or edit the generated manifest.json to customize runtime defaults.

## Filesystem Options

**Squashfs (Default & Recommended)**

Squashfs is the default filesystem type for OCI rootfs images:

```toml
[filesystem]
type = "squashfs"           # Default
compression_level = 15      # 1-22, default 15 (balanced compression)
overlay_size = "1G"         # tmpfs size for runtime writes
```

Benefits:
- Compressed storage (50-70% smaller than uncompressed)
- Read-only base with tmpfs overlay for writes
- No disk preallocation needed (faster deployment)
- Ideal for pre-built images and production use

**Legacy Options (ext4/xfs/btrfs)**

For advanced use cases requiring writable persistent disks:

```toml
[filesystem]
type = "ext4"               # ext4|xfs|btrfs
size_buffer_mb = 100        # extra free space beyond image size
preallocate = false         # optionally preallocate file
```

Note: These options require manual disk management and are only feasible for users building their own boot disks.

Validation (config.Validate):
- Either `source.image` **or** `source.dockerfile` must be set (mutually exclusive)
- `[filesystem]` required
- For squashfs: compression_level 0-22, overlay_size required
- For ext4/xfs/btrfs: non-negative size_buffer_mb

### Size and Performance Notes

- `size_buffer_mb` controls free space to accommodate runtime writes
- ext4 is the default trade-off; xfs and btrfs are supported if your runtime expects them
- For large images, consider `preallocate = true` to reduce fragmentation on host filesystems

## File Mappings and Permissions

Use `[mappings]` in fledge.toml to place additional files:

```toml
[mappings]
"./myconfig.toml" = "/etc/myapp/config.toml"
"./scripts/entrypoint.sh" = "/usr/local/bin/entrypoint.sh"
"./lib/libcustom.so" = "/usr/lib/libcustom.so"
```

Fledge automatically sets permissions based on destination path:
- Executables under /usr/bin, /usr/sbin, /bin, /sbin → 0755
- Libraries under /lib, /usr/lib → 0755
- Others default to 0644 unless already executable

See fledge/internal/builder/mapping.go for rules.

## Manifest Wiring

The generated manifest.json includes the rootfs section:

```json
{
  "$schema": "./../schemas/image-manifest-v1.json",
  "schema_version": "1.0",
  "name": "nginx",
  "version": "0.1.0",
  "runtime": "nginx",
  "enabled": true,
  "rootfs": {
    "url": "file:///path/to/rootfs.img",
    "checksum": "sha256:...",
    "format": "ext4"
  },
  "resources": {
    "cpu_cores": 2,
    "memory_mb": 1024
  },
  "workload": {
    "entrypoint": ["/docker-entrypoint.sh", "nginx", "-g", "daemon off;"],
    "env": {
      "NGINX_PORT": "80"
    }
  }
}
```

When RootFS is set, the orchestrator ensures kernel args include a default root device/fstype if not provided by the runtime:
- root device: vda
- fstype: ext4

You can override these via runtime args in the manifest if your disk image requires different values.

## Configuration Override Examples

The three-tier hierarchy allows customization without rebuilding:

```bash
# Development: minimal resources
volar vms create dev --image nginx \
  --cpu 1 \
  --memory 512 \
  --env LOG_LEVEL=debug

# Production: more resources, different ports
volar vms create prod --image nginx \
  --cpu 8 \
  --memory 8192 \
  --env LOG_LEVEL=error \
  --port 443:80

# Staging: custom configuration
volar vms create staging --image nginx \
  --env UPSTREAM_HOST=staging-backend.internal \
  --env CACHE_ENABLED=true \
  --memory 2048
```

Override hierarchy:
- VM creation flags (highest priority)
- manifest.toml defaults
- System defaults (lowest priority)

## Complete Example

A complete OCI rootfs image project:

**Dockerfile:**
```dockerfile
FROM node:18-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .

FROM node:18-alpine AS runtime
WORKDIR /app
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/src ./src
COPY --from=builder /app/package.json ./

EXPOSE 3000
CMD ["node", "src/server.js"]
```

**fledge.toml:**
```toml
version = "1"
strategy = "oci_rootfs"

[agent]
source_strategy = "release"
version = "latest"

[source]
dockerfile = "./Dockerfile"
context = "."
target = "runtime"

[source.build_args]
NODE_ENV = "production"

[filesystem]
type = "squashfs"
compression_level = 15
overlay_size = "1G"

[mappings]
"./config/production.toml" = "/app/config/production.toml"
```

**manifest.toml:**
```toml
schema_version = "v1"
name = "node-api"
version = "2.1.0"
runtime = "node-api"

# Resources section is OPTIONAL - omit to use defaults (cpu_cores=2, memory_mb=2048)
[resources]
cpu_cores = 2
memory_mb = 1024

[workload]
type = "exec"  # Can be "exec", "http", or "grpc"
entrypoint = ["/usr/local/bin/node", "src/server.js"]

[env]
NODE_ENV = "production"
PORT = "3000"
LOG_LEVEL = "info"
DATABASE_URL = "postgres://db.internal/api"
REDIS_URL = "redis://cache.internal:6379"

[network]
mode = "bridged"

[[network.expose]]
port = 3000
protocol = "tcp"

[[network.expose]]
port = 9090
protocol = "tcp"  # metrics

[actions]

[actions.health]
description = "Health check endpoint"
method = "GET"
path = "/health"
timeout_ms = 5000

[actions.reload]
description = "Reload configuration"
method = "POST"
path = "/admin/reload"
timeout_ms = 10000

[cloud_init]
datasource = "nocloud"

[cloud_init.user_data]
content = '''
#cloud-config
packages:
  - curl
  - jq
write_files:
  - path: /etc/systemd/resolved.conf.d/dns.conf
    content: |
      [Resolve]
      DNS=8.8.8.8
'''

[labels]
environment = "production"
service = "node-api"
team = "backend"
```

Build and deploy:

```bash
# Build
sudo fledge build

# Install
volar images install dist/manifest.json

# Development instance
volar vms create api-dev --image node-api \
  --cpu 1 \
  --memory 512 \
  --env NODE_ENV=development \
  --env LOG_LEVEL=trace

# Production instances
volar vms create api-prod-1 --image node-api \
  --cpu 4 \
  --memory 4096 \
  --env LOG_LEVEL=warn

volar vms create api-prod-2 --image node-api \
  --cpu 4 \
  --memory 4096 \
  --env LOG_LEVEL=warn
```

## Overrides and Additional Disks

Per-VM overrides in volar config:
- Setting Config.RootFS.URL clears any Initramfs
- Additional disks from the manifest are attached as secondary volumes

## Additional References

- Fledge config schema: fledge/internal/config/schema.go
- OCI rootfs builder: fledge/internal/builder/oci_rootfs.go
- File mapping rules: fledge/internal/builder/mapping.go
- Image manifest schema: docs/6_reference/1_manifest-schema.md
- fledge.toml reference: docs/6_reference/2_fledge-toml.md
- manifest.toml reference: docs/6_reference/3_manifest-toml.md
