---
title: "Initramfs Image Development"
author: "VolantVM"
date: "2025-11-01"
---


Build minimal, fast-booting images using Fledge's initramfs strategy. Initramfs images boot in under 150ms and have minimal memory footprints, making them ideal for performance-critical or high-density workloads.

Ground truth:
- fledge/internal/config/schema.go (InitConfig, AgentConfig, SourceConfig)
- fledge/internal/builder/initramfs.go (build pipeline)
- fledge/internal/builder/embed/init.c (C init behavior)

## When to Use

- Small static binaries, stateless services
- Cold-start sensitive workloads
- Full control over PID 1 when needed
- Performance-critical applications
- High-density deployments

## Configuration Files

Initramfs images require two configuration files:

### fledge.toml (Build Configuration)

Defines how to build the initramfs artifact.

### manifest.toml (Runtime Defaults)

Defines default resources, environment, and workload configuration for VMs created from this image.

## Init Modes

Fledge supports 3 modes via `[init]` in fledge.toml:

### Mode 1: Default (Kestrel Agent)

The C init mounts /proc, /sys, /dev, /tmp, /run then execs `/bin/kestrel`. Requires `[agent]`.

**fledge.toml:**
```toml
version = "1"
strategy = "initramfs"

[agent]
source_strategy = "release"
version = "latest"

[source]
# BusyBox defaults are applied automatically; override if needed:
# busybox_url = "https://busybox.net/downloads/binaries/1.35.0-x86_64-linux-musl/busybox"
# busybox_sha256 = "6e123e7f3202a8c1e9b1f94d8941580a25135382b99e8d3e34fb858bba311348"

[mappings]
"./myapp" = "/usr/bin/myapp"
```

**manifest.toml:**
```toml
schema_version = "1.0"
name = "myapp"
version = "1.0.0"
runtime = "myapp"

[resources]
cpu_cores = 1
memory_mb = 512

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

Kestrel is placed at `/bin/kestrel` and becomes PID 1 via C init.

### Mode 2: Custom Init

Your binary/script is PID 1 (mapped to /init). No kestrel.

**fledge.toml:**
```toml
version = "1"
strategy = "initramfs"

[init]
path = "./my-init"

[source]
# BusyBox defaults will be injected if omitted

[mappings]
"./my-init" = "/init"
"./myapp" = "/usr/bin/myapp"
```

**manifest.toml:**
```toml
schema_version = "1.0"
name = "myapp"
version = "1.0.0"
runtime = "myapp"

[resources]
cpu_cores = 1
memory_mb = 512

[workload]
entrypoint = "/init"
args = []
```

Fledge copies `./my-init` to `/init` (0755). No agent allowed in this mode.

### Mode 3: None (Direct PID 1)

Your binary is PID 1 and must mount filesystems yourself.

**fledge.toml:**
```toml
version = "1"
strategy = "initramfs"

[init]
none = true

[source]
# BusyBox defaults will be injected if omitted

[mappings]
"./my-supervisor" = "/init"
```

**manifest.toml:**
```toml
schema_version = "1.0"
name = "my-supervisor"
version = "1.0.0"
runtime = "my-supervisor"

[resources]
cpu_cores = 1
memory_mb = 512

[workload]
entrypoint = "/init"
args = []
```

Your binary must mount `/proc`, `/sys`, `/dev` and handle PID 1 responsibilities.

## Build

```bash
# Install fledge
curl -LO https://github.com/volantvm/fledge/releases/latest/download/fledge-linux-amd64
chmod +x fledge-linux-amd64 && sudo mv fledge-linux-amd64 /usr/local/bin/fledge

# Build image
sudo fledge build
# → outputs dist/plugin.cpio.gz and dist/manifest.json
```

## Install and Run

```bash
# Install image to registry
volar images install dist/manifest.json

# Create VM with defaults from manifest.toml
volar vms create demo --plugin myapp

# Or create with overrides
volar vms create prod --plugin myapp \
  --cpu 2 \
  --memory 1024 \
  --env LOG_LEVEL=debug \
  --env WORKERS=4

# List VMs
volar vms list

# Check VM details
volar vms show demo
```

## Build from Dockerfile

Fledge can execute Dockerfiles inside its embedded BuildKit and merge the resulting filesystem into the initramfs before adding your init payload.

### Config-Driven Workflow

**fledge.toml:**
```toml
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

**manifest.toml:**
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
args = ["--config", "/etc/myapp/config.toml"]

[workload.env]
APP_VERSION = "1.0.0"
LOG_LEVEL = "info"
```

Fledge runs the Dockerfile via the embedded BuildKit solver, overlays the result, applies mappings, and injects the agent according to the chosen init mode.

### Direct CLI Workflow

Skip the config files entirely by pointing `fledge build` at a Dockerfile:

```bash
sudo fledge build ./Dockerfile \
  --context . \
  --build-arg APP_VERSION=1.0.0 \
  --output-initramfs \
  --output myapp-initramfs
# → outputs myapp-initramfs.cpio.gz + myapp-initramfs.manifest.json
```

Note: You'll need to manually create a manifest.toml or edit the generated manifest.json to customize runtime defaults.

## Configuration Override Examples

The three-tier hierarchy allows customization without rebuilding:

```bash
# Development: debug mode
volar vms create dev --plugin myapp \
  --cpu 1 \
  --memory 256 \
  --env LOG_LEVEL=trace \
  --env DEBUG=true

# Production: optimized resources
volar vms create prod --plugin myapp \
  --cpu 4 \
  --memory 2048 \
  --env LOG_LEVEL=error \
  --env WORKERS=16

# Staging: custom database
volar vms create staging --plugin myapp \
  --env DATABASE_URL=staging.db.internal \
  --env CACHE_ENABLED=false
```

Override hierarchy:
- VM creation flags (highest priority)
- manifest.toml defaults
- System defaults (lowest priority)

## Complete Example

A complete initramfs image project:

**fledge.toml:**
```toml
version = "1"
strategy = "initramfs"

[agent]
source_strategy = "release"
version = "latest"

[source]
busybox_url = "https://busybox.net/downloads/binaries/1.35.0-x86_64-linux-musl/busybox"
busybox_sha256 = "6e123e7f3202a8c1e9b1f94d8941580a25135382b99e8d3e34fb858bba311348"

[mappings]
"./bin/web-server" = "/usr/bin/web-server"
"./config/app.toml" = "/etc/web-server/config.toml"
"./lib/libssl.so.1.1" = "/usr/lib/libssl.so.1.1"
```

**manifest.toml:**
```toml
schema_version = "1.0"
name = "web-server"
version = "1.2.3"
runtime = "web-server"

[resources]
cpu_cores = 2
memory_mb = 512

[workload]
entrypoint = "/usr/bin/web-server"
args = ["--config", "/etc/web-server/config.toml"]

[workload.env]
PORT = "8080"
LOG_LEVEL = "info"
LOG_FORMAT = "json"
WORKERS = "4"
MAX_CONNECTIONS = "1000"

[network]
mode = "bridged"

[[network.expose]]
port = 8080
protocol = "tcp"

[[network.expose]]
port = 9090
protocol = "tcp"  # metrics

[actions]

[actions.reload]
description = "Reload configuration without restart"
method = "POST"
path = "/admin/reload"
timeout_ms = 5000

[actions.metrics]
description = "Get application metrics"
method = "GET"
path = "/metrics"
timeout_ms = 3000

[labels]
environment = "production"
service = "web-server"
team = "platform"
```

Build and deploy:

```bash
# Build
sudo fledge build

# Install
volar images install dist/manifest.json

# Create multiple instances
volar vms create web-1 --plugin web-server
volar vms create web-2 --plugin web-server --port 8081:8080

# Production instance with more resources
volar vms create web-prod --plugin web-server \
  --cpu 8 \
  --memory 4096 \
  --env WORKERS=16 \
  --env MAX_CONNECTIONS=10000
```

## Additional References

- BusyBox defaults and agent validation: fledge/internal/config/config.go
- Build pipeline: fledge/internal/builder/initramfs.go
- Init wrapper behavior: fledge/internal/builder/embed/init.c
- Image manifest schema: docs/6_reference/1_manifest-schema.md
- fledge.toml reference: docs/6_reference/2_fledge-toml.md
- manifest.toml reference: docs/6_reference/3_manifest-toml.md
