---
title: "Networking Model"
author: "VolantVM"
date: "2025-11-01"
---


Volant supports three network modes declared in the image manifest (internal/imagespec/spec.go) and overridable per-VM via vmconfig.Config.

## Load Balancer (driftd)

Volant includes an integrated L4 load balancer called driftd that provides port forwarding and NAT capabilities.

- Architecture
  - eBPF TC-based dataplane for high-performance packet processing
  - Single BPF program (drift_l4.bpf.o) with ingress/egress hooks
  - Stateful connection tracking with shared maps
  - Auto-detection of external network interface

- Features
  - Port forwarding: host ports to VM IPs
  - Stateful NAT with bidirectional packet rewriting
  - Connection tracking for consistent routing
  - REST API for route management

- Integration
  - Controlled by volantd via internal/server/driftclient
  - Automatically manages VM port forwarding rules
  - Persistent route storage with automatic restoration
  - Systemd service runs alongside volantd

- Implementation
  - Files: cmd/driftd, internal/drift
  - BPF programs: internal/drift/bpf/drift_l4_ingress.c, internal/drift/bpf/drift_l4_egress.c
  - TC attachment to external interface (not bridge)
  - Route storage: file-based in /var/lib/volant/drift

## Modes

- vsock
  - No IP networking; host↔guest comms via virtio-vsock only
  - Orchestrator: no tap created; no IP allocated
  - Launcher: configures --vsock cid=<cid>

- bridged
  - Host-managed Linux bridge (vbr0) with a per-VM tap device
  - Orchestrator: allocates IP and prepares tap; attaches tap to bridge
  - Launcher: configures --net tap=<tap>,mac=<mac>,ip=<ip>,mask=<netmask>

- dhcp
  - Similar to bridged, but guest obtains IP via DHCP
  - Orchestrator: prepares tap; no host IP allocation
  - Launcher: configures --net tap=<tap>,mac=<mac> (no ip/mask)

## Bridge/Tap Provisioning

- Code: internal/server/orchestrator/network/bridge.go (linux)
  - Creates tuntap device with TUNTAP_VNET_HDR and attaches it to the bridge
  - Naming: vttap-<sanitized-name-or-hash>, constrained to IFNAMSIZ 15 chars
- Non-Linux builds use a noop manager (bridge_stub.go → NewNoop())

## Setup Script

- Code: internal/setup/setup.go
  - Creates bridge vbr0, assigns host CIDR, brings it up
  - Enables IP forwarding and sets NAT masquerade for the subnet
  - Writes systemd units for both volantd and driftd
  - Configures driftd to run alongside volantd for L4 load balancing

## Decision Logic (when to create taps, allocate IPs)

- Tested in internal/server/orchestrator/network_test.go
- needsTapDevice(cfg): true for bridged/dhcp; false for vsock; default true if cfg nil/empty
- needsIPAllocation(cfg): true for bridged; false for dhcp/vsock; default true if cfg nil/empty
