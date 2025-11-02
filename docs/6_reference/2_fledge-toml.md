---
title: "fledge.toml Reference"
author: "VolantVM"
date: "2025-11-01"
---


The fledge.toml file defines **build-time configuration** for Fledge, the Volant image builder. This file is separate from manifest.toml (which defines runtime defaults) and contains only build-related settings.

## Understanding the Configuration Split

Fledge uses two configuration files with distinct purposes:

**fledge.toml** (this file):
- Defines **how to build** the image
- Strategy selection (initramfs or oci_rootfs)
- Source configuration (base image, Dockerfile)
- Agent sourcing
- File mappings
- Build-time only - not included in final image

**manifest.toml**:
- Defines **runtime defaults** for VMs
- CPU, memory, environment variables
- Networking, port exposures
- Workload entrypoint and args
- See [manifest.toml Reference](3_manifest-toml.md)

Both files are read by `fledge build`, which merges them with build metadata to produce manifest.json and boot media.

## References

- fledge/internal/config/schema.go
- fledge/internal/config/config.go (Validation rules)

## Editor Integration

For Taplo-compatible editors (VS Code/JetBrains TOML images), add this comment at the very top of your fledge.toml to enable autocomplete and validation:

```
# schema = "https://raw.githubusercontent.com/volantvm/volant/main/docs/schemas/fledge-toml-v1.json"
```

## Top-level

- version: string (required) — must equal "1"
- strategy: string (required) — "initramfs" or "oci_rootfs"
- agent: AgentConfig (optional; only allowed for initramfs default mode)
- init: InitConfig (optional; initramfs only)
- source: SourceConfig (required; fields depend on strategy)
- filesystem: FilesystemConfig (optional; required for oci_rootfs)
- mappings: map[string]string (optional; host→guest file placements)

## InitConfig (initramfs only)

- path: string (optional) — sets custom PID1; mutually exclusive with none=true
- none: bool (optional) — makes your binary PID1; mutually exclusive with path

Init mode is derived as:
- default: init unset or empty (requires [agent])
- custom: init.path set (forbids [agent])
- none: init.none=true (forbids [agent])

## AgentConfig (initramfs default mode only)

- source_strategy: string (required) — "release" | "local" | "http"
- version: string (required for release)
- path: string (required for local)
- url: string (required for http)
- checksum: string (optional for http)

## SourceConfig

- For oci_rootfs:
  - image: string — reference to an existing image (mutually exclusive with dockerfile)
  - dockerfile: string — path to a Dockerfile to build locally (mutually exclusive with image)
  - context: string (optional) — build context directory; defaults to the Dockerfile's directory
  - target: string (optional) — multi-stage target
  - build_args: map[string]string (optional) — forwarded as build arguments

- For initramfs:
  - dockerfile/context/target/build_args — optional Dockerfile overlay before init payload is added
  - busybox_url: string (optional) — override default BusyBox URL
  - busybox_sha256: string (optional) — override default BusyBox checksum

## FilesystemConfig (oci_rootfs only)

- type: string (required) — one of: ext4, xfs, btrfs
- size_buffer_mb: int (required) — additional free space to add; must be >= 0
- preallocate: bool (optional) — preallocate the image file

When absent, defaults are applied (DefaultFilesystemConfig):
- type: ext4
- size_buffer_mb: 100
- preallocate: false

## mappings

A map of host source path → absolute destination path inside the image. Validation:
- destination must be absolute (starts with /)
- destination cannot contain ".."

Placement rules follow FHS semantics (see fledge/internal/builder/mapping.go):
- Executables under /usr/bin, /usr/sbin, /bin, /sbin → 0755
- Libraries under /lib, /usr/lib → 0755
- Others keep mode or default to 0644

## Validation Summary

- version must be "1"
- strategy must be initramfs or oci_rootfs
- initramfs:
  - default mode requires [agent]
  - custom/none forbid [agent]
  - BusyBox URL/checksum default automatically if omitted
- oci_rootfs:
  - exactly one of source.image or source.dockerfile must be set
  - [filesystem] required; type in {squashfs (default), ext4, xfs, btrfs}; see below for type-specific options
- mappings: destination absolute and no ".."

See fledge/internal/config/config.go for full validation logic.

## What Does NOT Belong in fledge.toml

The following configuration belongs in **manifest.toml**, not fledge.toml:

- CPU and memory defaults (`resources`)
- Environment variables (`workload.env`)
- Workload entrypoint and args (`workload`)
- Port exposures (`network.expose`)
- Network mode (`network.mode`)
- Actions, cloud-init, devices
- Any runtime configuration

**Rule of thumb**: If it affects how VMs run (not how images build), it belongs in manifest.toml.

## See Also

- [manifest.toml Reference](3_manifest-toml.md) - Runtime defaults configuration
- [Build vs Runtime Configuration](../1_introduction/1_introduction.md#build-time-vs-runtime-configuration)
- [Quick Start: Initramfs](../2_getting-started/2_quick-start-initramfs.md)
- [Quick Start: OCI Rootfs](../2_getting-started/3_quick-start-rootfs.md)
