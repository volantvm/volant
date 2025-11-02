---
title: "kestrel (agent) reference"
author: "VolantVM"
date: "2025-11-01"
---


Kestrel is the in-guest agent and default PID1 for initramfs default mode.

Key behavior (see fledge/internal/builder/embed/init.c and agent code):
- When used as PID1, it prepares the guest environment and exposes a control socket over vsock.
- It can proxy HTTP requests from volantd to workloads running in the guest, enabling fully isolated vsock-only deployments.

The agent receives runtime arguments via the kernel cmdline, encoded by the orchestrator, including:
- runtime (imagespec.RuntimeKey)
- api host/port (imagespec.APIHostKey/APIPortKey)
- encoded manifest (imagespec.CmdlineKey)

For most users, there are no CLI flags to pass directly to kestrel; configuration is provided through the manifest and per-VM config.
