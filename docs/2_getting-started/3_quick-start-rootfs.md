---
title: "Quick Start: OCI Rootfs Strategy"
author: "VolantVM"
date: "2025-11-01"
---


Run unmodified Docker/OCI images as microVMs with hardware isolation. This path provides compatibility with existing container images while adding true VM-level security and deterministic networking.

## Install a Prebuilt Image

```bash
volar images install --manifest \
  https://raw.githubusercontent.com/volantvm/oci-plugin-example/main/manifest/nginx.json

volar vms create web --plugin nginx --cpu 2 --memory 1024
```

Result: a microVM that boots in seconds with an OCI-based root filesystem and full hardware isolation.

## Build Your Own Image with Fledge

### Install Fledge

```bash
curl -LO https://github.com/volantvm/fledge/releases/latest/download/fledge-linux-amd64
chmod +x fledge-linux-amd64 && sudo mv fledge-linux-amd64 /usr/local/bin/fledge
```

### Understanding the Configuration Split

Fledge uses two configuration files:

- **fledge.toml** - Defines how to build the image (build-time configuration)
- **manifest.toml** - Defines runtime defaults for VMs created from this image

This separation allows you to:
- Build images once with sensible defaults
- Create many VMs with different configurations
- Override settings without rebuilding

### Create Your Project

Create a directory for your image:

```bash
mkdir my-oci-app
cd my-oci-app
```

### fledge.toml (Build Configuration)

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
compression_level = 15      # 1-22, default 15 (balanced)
overlay_size = "1G"         # tmpfs size for runtime writes
```

You can swap `source.image` for a Dockerfile build:

```toml
[source]
dockerfile = "./Dockerfile"
context = "."
target = "runtime-stage"  # Optional multi-stage target

[source.build_args]
APP_VERSION = "1.0.0"
```

### manifest.toml (Runtime Defaults)

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
LOG_LEVEL = "info"

[[network.expose]]
port = 80
protocol = "tcp"

[[network.expose]]
port = 443
protocol = "tcp"
```

### Build the Image

```bash
sudo fledge build
```

This will:
1. Read both fledge.toml and manifest.toml
2. Fetch the OCI image or build from Dockerfile
3. Unpack layers and install the Kestrel agent
4. Create squashfs filesystem image (compressed, read-only with overlay)
5. Generate manifest.json and rootfs.squashfs in the dist/ directory

### Install and Run

```bash
# Install the image
volar images install dist/manifest.json

# Create a VM with defaults from manifest.toml
volar vms create demo --plugin nginx

# Or create a VM with overrides
volar vms create prod --plugin nginx \
  --cpu 4 \
  --memory 2048 \
  --env LOG_LEVEL=warn \
  --port 8080:80 \
  --port 8443:443

# List your VMs
volar vms list

# Check the VM's IP and access your app
volar vms show demo
curl http://<vm-ip>:80
```

## Build Directly from Dockerfile

Skip the config files and build directly from a Dockerfile:

```bash
sudo fledge build ./Dockerfile \
  --context . \
  --target runtime-stage \
  --build-arg FOO=bar \
  --output my-rootfs

# Outputs: my-rootfs.img + my-rootfs.manifest.json
```

Note: When using direct CLI builds, you'll need to manually create a manifest.toml or edit the generated manifest.json to set runtime defaults.

Then install and run:

```bash
volar images install my-rootfs.manifest.json
volar vms create demo --plugin my-rootfs
```

## Configuration Override Examples

The three-tier configuration system lets you customize VMs without rebuilding images:

```bash
# Development: minimal resources, debug logging
volar vms create dev --plugin nginx \
  --cpu 1 \
  --memory 512 \
  --env LOG_LEVEL=debug \
  --env NGINX_PORT=8080

# Production: more resources, different ports
volar vms create prod --plugin nginx \
  --cpu 8 \
  --memory 8192 \
  --env LOG_LEVEL=error \
  --port 443:80

# Staging: custom environment variables
volar vms create staging --plugin nginx \
  --env UPSTREAM_HOST=staging-backend.internal \
  --env CACHE_ENABLED=true
```

The override hierarchy:
- CLI flags (highest priority) override manifest defaults
- Manifest defaults (from manifest.toml) override system defaults
- System defaults (cpu=2, memory=2048) are the fallback

## Filesystem Options

**Squashfs (Default & Recommended)**

Squashfs is the default filesystem type for OCI rootfs images. It provides:
- Compressed storage (significantly smaller disk usage)
- Read-only base with tmpfs overlay for writes
- Faster deployment (no disk preallocation needed)

```toml
[filesystem]
type = "squashfs"           # Default
compression_level = 15      # 1-22, default 15 (balanced compression)
overlay_size = "1G"         # tmpfs size for runtime writes (default "1G")
```

**Legacy Options (ext4/xfs/btrfs)**

For advanced use cases requiring writable disks, ext4/xfs/btrfs are available but require manual disk management:

```toml
[filesystem]
type = "ext4"               # or "xfs", "btrfs"
size_buffer_mb = 100        # Extra space on top of image size
preallocate = false         # Sparse vs preallocated file
```

Note: These legacy options are only feasible if you manually manage boot disks. For most users pulling pre-built images, squashfs is the practical choice.

## Next Steps

- See [OCI Rootfs Development Guide](../4_image-development/3_oci-rootfs.md) for deeper coverage
- Learn about [Configuration Hierarchy](../1_introduction/1_introduction.md#configuration-hierarchy)
- Explore [fledge.toml Reference](../6_reference/2_fledge-toml.md)
- Explore [manifest.toml Reference](../6_reference/3_manifest-toml.md)
- Learn about [Networking](../3_guides/1_networking.md)
- Set up [Cloud-init](../3_guides/2_cloud-init.md)
