---
title: "Introduction"
author: "VolantVM"
date: "2025-11-01"
---


## Meet Volant

Volant is a modular microVM orchestration engine — a platform that makes virtualization programmable. It ships a compact control plane, CLI, and in-guest agent that share a common manifest format. Each workload runs inside its own microVM with deterministic networking, real resource isolation, and clean lifecycle control.

Runtime behavior lives in signed image manifests and their artifacts — kernels, initramfs bundles, or rootfs disks — keeping the core minimal while letting image authors define their environment precisely.

Modern infrastructure has drifted into abstraction overload. Layers on layers have made simple things complex. Volant moves the other way — treating microVMs as first-class runtimes, combining the clarity and security of hardware virtualization with the ergonomics of containers. No sidecars. No service meshes. No hidden daemons. Just a single binary and a clear contract between control plane and guest.

Volant exists to make infrastructure predictable again.

## Understanding Core Concepts

Before diving into installation and usage, it's important to understand how Volant structures workloads and configuration.

### Images vs VMs

An **image** is a template or blueprint that defines what software to run, default resource allocations, environment configuration, and boot media. Think of an image like a Docker image or AWS AMI — it's a reusable template.

A **VM** (Virtual Machine) is a running instance created from an image. Each VM:
- Is created from exactly one image
- Can override the image's defaults at creation time
- Has its own state, IP address, and lifecycle
- Runs independently from other VMs

Think of a VM like a Docker container or EC2 instance — it's a running instance.

### Configuration Hierarchy

Volant uses a three-tier configuration system that provides flexibility while maintaining clear boundaries:

```
VM Creation Flags (Highest)  >  Image Manifest (Middle)  >  System Defaults (Lowest)
```

**System Defaults** (hardcoded fallbacks):
- cpu_cores: 2
- memory_mb: 2048

**Image Manifest** (runtime defaults defined in manifest.toml):
- CPU, memory, environment variables
- Port exposures, networking mode
- Workload command and args
- Devices and PCI passthrough rules

**VM Creation Flags** (instance-specific overrides):
- `--cpu 4` overrides manifest CPU
- `--memory 2048` overrides manifest memory
- `--env KEY=value` merges with manifest environment (CLI wins on conflicts)
- `--port 8080:80` completely replaces manifest ports if provided

This hierarchy allows you to:
- Build images once with sensible defaults
- Deploy many instances with different configurations
- Override any setting without rebuilding the image

### Build-Time vs Runtime Configuration

Volant maintains a clean separation between build-time and runtime configuration:

**fledge.toml** (Build-Time):
- Defines **how to build an image**
- Builder configuration (initramfs or oci_rootfs)
- Source configuration (base image or Dockerfile)
- Agent configuration
- File mappings

**manifest.toml** (Runtime Defaults):
- Defines **what the image provides by default**
- Default resources (CPU, memory)
- Default environment variables
- Networking configuration
- Workload entrypoint and args

**CLI Flags** (Instance Overrides):
- Customize individual VMs without changing the image
- Override resources, environment, ports
- Perfect for dev/staging/prod differences

The workflow:
1. Build: `fledge build` reads both fledge.toml and manifest.toml
2. Output: Generates manifest.json (merged result) and boot media
3. Install: `volar images install dist/manifest.json`
4. Create VMs: `volar vms create app --plugin myimage` (uses defaults)
5. Or override: `volar vms create prod --plugin myimage --cpu 4 --memory 4096`

## Two Paths to MicroVM Execution

Volant unifies two worlds — **compatibility** and **performance** — under one control plane:

### For Universal Compatibility: The OCI Rootfs Path

Run any unmodified Docker/OCI image as a secure, high-performance microVM.

Thousands of existing containerized applications can run unmodified inside Cloud Hypervisor microVMs with full hardware isolation and deterministic networking — no code changes required.

This is the **pragmatic path** — for when you need compatibility and migration ease.

Boot times: 2-5 seconds
Memory footprint: ~50 MB + workload
Use cases: Existing containerized apps, migration scenarios

### For Maximum Performance: The Initramfs Path

Create hyper-optimized, appliance-style microVMs that boot in milliseconds.

Package your static binary and dependencies into a minimal initramfs — no extra files, no unused packages, no wasted bytes.

- Boot times under 100 ms
- Memory footprints under 20 MB
- Attack surface measured in kilobytes
- Perfect for serverless-style or high-density workloads

This is the **performance path** — for when you need speed, efficiency, and total control.

Both paths share the same platform, API, and tooling.

## A Search for Sanity

Something is off with how we run software today. We don't hate containers — they're indispensable — but somewhere along the way, we lost the plot.

A simple web server shouldn't weigh 197 MB. It shouldn't drag in an entire userland, package manager, and libraries it never touches. Yet that's become our definition of "lightweight."

A decade of comfort left us with runtimes that are bloated, opaque, and fragile — systems so heavy with tooling that debugging the tooling takes longer than building the software itself.

Namespace "isolation" still shares a kernel. Service meshes add layers to fix layers. And deploying a basic service now requires consensus algorithms.

Volant restores sanity — stripping away unnecessary complexity and returning to a model that is secure by hardware design, transparent by construction, and predictable by default.

## What Makes Volant Different

### Developer-First Design
Infrastructure isn't inherently complex — the experience built around it is. Volant fixes that with a human-first design and frictionless workflows.

### True Hardware Isolation
Every workload runs in its own Cloud Hypervisor microVM — not a container, not a namespace — a real VM with its own kernel, isolated from the host at the CPU level. Security by design, not by sandbox trickery.

### Static, Predictable Networking
Each microVM gets a static IP from a deterministic pool. No overlays, no discovery layers — simple, reliable, and debuggable.

### Kernel and Boot Path
Volant ships a custom-built kernel with an embedded initramfs containing the kestrel agent and a lightweight C init. The embedded init detects the boot path from kernel command-line parameters and either stays in initramfs for appliance workloads or pivots to rootfs for OCI-based workloads. Each release includes both **bzImage** (compressed) and **vmlinux** (uncompressed ELF) formats of the same kernel. bzImage is used by default; vmlinux is available for power users who need the uncompressed format. All artifacts ship with SHA256 checksums and build provenance attestation.

### Kestrel: The Intelligent Supervisor
Every microVM runs **kestrel**, Volant's in-guest PID 1 that handles mounts, pivots, supervision, and manifest-driven orchestration. It's the heartbeat of every VM.

## The Core Components

### volantd — The Control Plane
A single Go binary that manages state (SQLite), allocates IPs, orchestrates microVMs, hosts the image registry, and exposes REST + MCP APIs. No dependencies. No consensus systems. Just one daemon.

### volar — The CLI
A scriptable tool that creates, lists, stops, and manages microVMs and images. Designed for both automation and direct use.

### kestrel — The In-Guest Agent
Handles two-stage boot, mounts essential filesystems, supervises workloads, performs health checks, and exposes an optional HTTP proxy.

### fledge — The Image Builder
Builds rootfs- or initramfs-based images from declarative configs (fledge.toml + manifest.toml). Reproducible, CI/CD-friendly, and minimal by default.

## The Image System

Volant is **image-first.** The core engine is generic; runtime-specific logic lives in manifests that define resources, entrypoints, and artifacts. Images are built with fledge, installed into the image registry, and can be instantiated as many VMs as needed with different configurations.

The workflow:
1. Author: Create fledge.toml (build config) and manifest.toml (runtime defaults)
2. Build: Run `fledge build` to generate manifest.json and boot media
3. Install: `volar images install dist/manifest.json` adds image to registry
4. Create VMs: `volar vms create name --plugin imagename` with optional overrides

## Real-World Use Cases

1. **Isolated Browser Automation** — Hardware-level sandboxing for headless browsers
2. **AI/ML Inference** — Secure, snapshot-ready GPU workloads
3. **Multi-Tenant Dev Environments** — True isolation without Docker-in-Docker
4. **Secure CI/CD Runners** — Disposable build VMs
5. **Edge Nodes** — Lightweight, deterministic execution at the edge
6. **Protocol Bridges** — Dedicated VMs for networking and gateway tasks

## Who Should Use Volant

Engineers who demand control, performance, and security — from platform teams to AI infra engineers, image authors, and anyone tired of container bloat.

## What Volant Is Not

- **Not Kubernetes** — Focused on single-node or small-cluster orchestration
- **Not a Container Runtime** — Runs VMs, not namespaces
- **Not a Cloud Platform** — You own the data and deployment
- **Not Magic** — Prioritizes simplicity and determinism over extreme density

## Core Principles

1. **Hardware Isolation as a Primitive** — Every workload gets its own kernel
2. **Simplicity Over Cleverness** — Static IPs, SQLite, and direct config over abstractions
3. **Image-First Architecture** — Extensible without bloat
4. **Developer Experience Matters** — Good defaults, clear logs, predictable tools
5. **Security Without Compromise** — Reproducible builds, verified artifacts, minimal surface

## Performance Characteristics

| Mode | Boot Time | Memory | Disk | Use Case |
|------|------------|--------|-------|-----------|
| **Rootfs (OCI)** | 2–5 s | ~50 MB + workload | Variable | Compatibility |
| **Initramfs** | 50–150 ms | 10–20 MB + workload | 5–50 MB | Performance |

Both paths support snapshot/restore and deterministic resource allocation.

## Technology Stack

**Hypervisor:** Cloud Hypervisor (Rust)
**Control Plane:** Go 1.22+
**Database:** SQLite
**Agent:** Go (static binary)
**C Shim:** Minimal init < 10 KB
**Networking:** Linux bridge + static IPAM
**Build Tools:** fledge, skopeo, umoci, busybox

No external dependencies beyond Linux + KVM.

## Get Started

1. **[Installation](../2_getting-started/1_installation.md)** — Install in under 60 seconds
2. **[Quick Start: Initramfs](../2_getting-started/2_quick-start-initramfs.md)** — Build and deploy an appliance
3. **[Quick Start: OCI Rootfs](../2_getting-started/3_quick-start-rootfs.md)** — Run your first OCI image

**For Image Authors:**
- **[Image Development Overview](../4_image-development/1_overview.md)**
- **[Initramfs Guide](../4_image-development/2_initramfs.md)**
- **[OCI Rootfs Guide](../4_image-development/3_oci-rootfs.md)**

**For Deep Divers:**
- **[Architecture Overview](../5_architecture/1_overview.md)**
- **[Components and Responsibilities](../5_architecture/2_components.md)**

## Community and Support

Volant is open source under **Business Source License 1.1**, converting to **Apache 2.0** on Oct 4, 2029.

- **GitHub:** [github.com/volantvm/volant](https://github.com/volantvm/volant)
- **Docs:** [docs.volantvm.com](https://docs.volantvm.com)
- **Issues:** [github.com/volantvm/volant/issues](https://github.com/volantvm/volant/issues)
- **Discussions:** [github.com/volantvm/volant/discussions](https://github.com/volantvm/volant/discussions)

We ship fast. Volant already supports dual-kernel boot, OCI & initramfs paths, static IP management, image registry, REST/MCP APIs, deployments, and event streams.

**Coming Soon (1–3 months):** GPU passthrough, snapshots, and multi-host clustering.
**Coming Later (3–6 months):** integrated PaaS, snapshot-warmed serverless, advanced networking.

## The Bottom Line

Volant is **microVM orchestration done right** — simple, secure, and production-ready. Two paths. One platform. Real isolation. Predictable performance. Image-first design.

**Build the runtime you need, without rebuilding the control plane.**

---

*Volant — The Intelligent Execution Cloud*
