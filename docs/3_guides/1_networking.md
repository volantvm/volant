---
title: "Networking guide"
author: "VolantVM"
date: "2025-11-01"
---


Ground truth: internal/server/orchestrator/orchestrator.go (resolveNetworkConfig, needsIPAllocation, needsTapDevice), internal/server/orchestrator/network/{bridge.go,noop.go}, internal/setup/setup.go.

Volant supports three network modes via the image manifest (or per-VM config override):
- bridged (default)
- vsock
- dhcp

The effective mode is resolved by resolveNetworkConfig(manifest, vmConfig). VM‑level config overrides the manifest.

## What each mode means

- bridged:
  - Host creates a tap device and attaches it to the configured Linux bridge (vbr0 by default).
  - Orchestrator allocates an IP from the managed subnet and programs the VM kernel cmdline with ip/gateway/netmask.
  - needsTapDevice = true; needsIPAllocation = true.

- vsock:
  - No guest Ethernet; communication via vsock only (kestrel agent proxy).
  - No tap device and no host‑managed IP.
  - needsTapDevice = false; needsIPAllocation = false.

- dhcp:
  - VM gets IP via DHCP from inside the guest; host only provides a tap/bridge.
  - needsTapDevice = true; needsIPAllocation = false.

## Host networking on Linux

- Bridge manager (internal/server/orchestrator/network/bridge.go) uses vishvananda/netlink to:
  - Ensure the bridge exists and is up
  - Create a tap interface with VNET_HDR and attach it to the bridge
  - Bring it up and hand the tap name to the runtime
- The bridge code is linux‑only (build‑tagged). On non‑Linux hosts, volantd falls back to NoopManager.

## macOS and non‑Linux hosts

- No tap devices are created automatically. volantd logs a warning and uses NoopManager which returns a deterministic tap name (no system changes).
- If you want bridged networking on macOS, create and manage the tap/bridge manually and point your runtime to it.

## Setup helper (Linux)

The CLI command `volar setup` (internal/cli/standard/setup.go) calls internal/setup/ to:
- Create bridge (default vbr0) and assign host IP (default 192.168.127.1/24)
- Enable IP forwarding and add MASQUERADE and FORWARD rules via iptables
- Write systemd units for volantd and driftd with proper environment
- Configure driftd for L4 load balancing with eBPF dataplane

You can run with --dry-run to print commands without applying them.

## Load Balancer Integration (driftd)

Volant includes an integrated L4 load balancer for port forwarding:

- eBPF TC-based dataplane attached to external interface
- Automatically managed by volantd for VM port forwarding
- Stateful NAT with connection tracking
- No manual configuration required for basic VM networking
- Route persistence across restarts

## Kernel cmdline and IP

For bridged mode, the orchestrator computes:
- ip=<guest_ip>::<gateway>:<netmask>:<hostname>:eth0:off
- gateway = hostIP from config
- netmask derived from subnet mask (formatNetmask)

This is passed to the runtime; the guest should configure eth0 accordingly on boot.

## Port Mapping (Docker-Style)

Volant supports Docker-style port mapping to expose VM services to the host. Port exposure works in two phases:

### 1. Image Manifest (Declaration)

Images declare what ports the application listens on inside the VM using the `network.expose` array in `manifest.json`:

```toml
[[network.expose]]
port = 3000
protocol = "tcp"

[[network.expose]]
port = 8080
protocol = "tcp"
host_port = 9000  # Optional: explicit host port mapping
```

This is similar to Docker's `EXPOSE` directive - it's documentation about what ports the app uses, but doesn't actually map them.

### 2. VM Creation (Runtime Mapping)

At VM creation time, you can:

**Auto-assign host ports** (default behavior):
```bash
volar vms create myvm --image myapp
# Manifest ports get auto-assigned: 3000→2234, 8080→2235
```

**Explicitly map host ports** using `--expose`:
```bash
# Docker-style syntax: [HOST_PORT:]CONTAINER_PORT[:PROTOCOL]
volar vms create myvm --image myapp --expose 8080:3000:tcp --expose 9090:8080:tcp
```

**Disable all port exposure** (secure by default):
```bash
volar vms create myvm --image myapp --no-expose
```

### Host Port Auto-Allocation

When ports are not explicitly mapped:
- Host ports are auto-assigned sequentially starting from **2234**
- Volant tracks all allocated ports across VMs to prevent conflicts
- If an explicit port is already in use, VM creation will fail

### Integration with driftd

Port mappings are automatically programmed into driftd's eBPF TC dataplane:
- NAT rules forward `host_ip:host_port` → `vm_ip:vm_port`
- Stateful connection tracking for TCP and UDP
- Routes persist across VM restarts
- No manual configuration required

### Examples

**Web application with auto-assigned ports:**
```toml
# manifest.toml
[[network.expose]]
port = 80
protocol = "tcp"
```
```bash
volar vms create web --image nginx
# Access via http://host_ip:2234 (auto-assigned)
```

**Database with explicit port:**
```bash
volar vms create postgres --image postgres:alpine --expose 5432:5432:tcp
# Access via postgresql://host_ip:5432
```

**Multi-port service:**
```bash
volar vms create app --image myapp \
  --expose 8080:3000:tcp \
  --expose 8081:3001:tcp \
  --expose 9000:9000:udp
```

**Secure deployment (no external ports):**
```bash
volar vms create internal-service --image worker --no-expose
# Only accessible via vsock or internal VM network
```
