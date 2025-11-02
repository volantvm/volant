---
title: "Quick Start: Initramfs Strategy"
author: "VolantVM"
date: "2025-11-01"
---


This path builds ultra-fast, minimal appliances using initramfs. Initramfs images boot in under 100ms and have minimal memory footprints, making them ideal for high-density or performance-critical workloads.

## Install a Prebuilt Image

```bash
volar images install --manifest \
  https://raw.githubusercontent.com/volantvm/initramfs-plugin-example/main/manifest/caddy.json

volar vms create web --plugin caddy --cpu 1 --memory 512
```

Result: a microVM that boots in ~100ms and serves HTTP on its assigned IP.

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
mkdir my-initramfs-app
cd my-initramfs-app
```

### fledge.toml (Build Configuration)

```toml
version = "1"
strategy = "initramfs"

[agent]
source_strategy = "release"
version = "latest"

[source]
# BusyBox defaults are applied automatically
# Override if you need a different build:
# busybox_url = "https://busybox.net/downloads/binaries/1.35.0-x86_64-linux-musl/busybox"
# busybox_sha256 = "6e123e7f3202a8c1e9b1f94d8941580a25135382b99e8d3e34fb858bba311348"

[mappings]
"./myapp" = "/usr/bin/myapp"
```

### manifest.toml (Runtime Defaults)

```toml
schema_version = "v1"
name = "myapp"
version = "0.1.0"
runtime = "myapp"

# Resources section is OPTIONAL - omit to use defaults (cpu_cores=2, memory_mb=2048)
[resources]
cpu_cores = 1
memory_mb = 512

[workload]
type = "exec"  # Can be "exec", "http", or "grpc"
entrypoint = ["/usr/bin/myapp"]

[env]
PORT = "8080"
LOG_LEVEL = "info"

[[network.expose]]
port = 8080
protocol = "tcp"
```

### Build the Image

```bash
sudo fledge build
```

This will:
1. Read both fledge.toml and manifest.toml
2. Download and configure BusyBox
3. Install the Kestrel agent
4. Apply your file mappings
5. Generate manifest.json and plugin.cpio.gz in the dist/ directory

### Install and Run

```bash
# Install the image
volar images install dist/manifest.json

# Create a VM with defaults from manifest.toml
volar vms create demo --plugin myapp

# Or create a VM with overrides
volar vms create prod --plugin myapp \
  --cpu 2 \
  --memory 1024 \
  --env LOG_LEVEL=debug

# List your VMs
volar vms list

# Check the VM's IP and access your app
volar vms show demo
curl http://<vm-ip>:8080
```

## Advanced: Build from Dockerfile

Fledge can execute Dockerfiles using its embedded BuildKit and merge the result into the initramfs:

### Option A: Config-Driven Workflow

```toml
# fledge.toml
version = "1"
strategy = "initramfs"

[agent]
source_strategy = "release"
version = "latest"

[source]
dockerfile = "./Dockerfile"
context = "."
target = "final"          # optional multi-stage target

[source.build_args]
APP_VERSION = "1.0.0"

[mappings]
"./extra-config" = "/etc/myapp/config.toml"
```

```toml
# manifest.toml
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
entrypoint = ["/usr/bin/myapp"]
```

Then build:

```bash
sudo fledge build
```

### Option B: Direct CLI Workflow

Skip the config files and build directly from a Dockerfile:

```bash
sudo fledge build ./Dockerfile \
  --context . \
  --build-arg APP_VERSION=1.0.0 \
  --output-initramfs \
  --output myapp-initramfs

# Outputs: myapp-initramfs.cpio.gz + myapp-initramfs.manifest.json
```

Note: When using direct CLI builds, you'll need to manually create a manifest.toml or edit the generated manifest.json to set runtime defaults.

## Configuration Override Examples

The three-tier configuration system lets you customize VMs without rebuilding images:

```bash
# Development: verbose logging, minimal resources
volar vms create dev --plugin myapp \
  --cpu 1 \
  --memory 256 \
  --env LOG_LEVEL=trace \
  --env DEBUG=true

# Production: more resources, custom port mapping
volar vms create prod --plugin myapp \
  --cpu 4 \
  --memory 2048 \
  --env LOG_LEVEL=warn \
  --port 443:8080

# Staging: different environment config
volar vms create staging --plugin myapp \
  --env DATABASE_URL=staging.db.example.com \
  --env CACHE_URL=staging.cache.example.com
```

The override hierarchy:
- CLI flags (highest priority) override manifest defaults
- Manifest defaults (from manifest.toml) override system defaults
- System defaults (cpu=2, memory=2048) are the fallback

## Next Steps

- See [Initramfs Development Guide](../4_image-development/2_initramfs.md) for deeper coverage
- Learn about [Configuration Hierarchy](../1_introduction/1_introduction.md#configuration-hierarchy)
- Explore [fledge.toml Reference](../6_reference/2_fledge-toml.md)
- Explore [manifest.toml Reference](../6_reference/3_manifest-toml.md)
