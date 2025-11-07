---
title: "Environment Variables"
author: "VolantVM"
date: "2025-01-07"
---

Ground truth: internal/imagespec/spec.go (Workload.Env), internal/server/orchestrator/orchestrator.go (env encoding), internal/agent/app/app.go (setEnvironmentFromCmdline, workload env).

Volant supports runtime environment variable configuration following the 12-Factor App methodology. Variables can be defined as image defaults and overridden per-VM at creation time.

## Configuration Sources

Environment variables come from two sources with clear precedence:

1. **VM creation flags** (`--env`) - highest priority
2. **Image manifest** (`[env]` in manifest.toml) - defaults

VM-level overrides replace manifest defaults entirely for a given key.

## Image Defaults

Define defaults in manifest.toml that apply to all VMs from this image:

```toml
[env]
LOG_LEVEL = "info"
DATABASE_URL = "postgres://localhost:5432/mydb"
PORT = "8080"
```

These are embedded in the manifest during build and become part of the image artifact.

## Runtime Overrides

Override at VM creation using `--env`:

```bash
volar vms create api --image myapp \
  --env LOG_LEVEL=debug \
  --env DATABASE_URL=postgres://database.volant:5432/mydb
```

Multiple `--env` flags are supported. Each flag follows the format `KEY=VALUE`.

## Implementation

The orchestrator (orchestrator.go:442-453):
- Merges manifest env + VM env (VM wins)
- Marshals to JSON and base64-encodes
- Passes via kernel cmdline as `volant.env=<base64>`

The guest agent (agent/app.go:505-547):
- Reads `volant.env` from `/proc/cmdline`
- Decodes base64 → JSON → map[string]string
- Sets each via `os.Setenv()`
- Passes to workload process via `cmd.Env`

This happens early in boot (line 94-97) before the workload starts.

## Service Discovery Integration

Combine with DNS for dynamic service references:

```bash
volar vms create api --image myapp \
  --env DATABASE_URL=postgres://postgres.volant:5432/mydb \
  --env CACHE_URL=redis://cache.volant:6379
```

Hostnames like `postgres.volant` resolve automatically via the embedded DNS server (see service-discovery.md).

## Usage in Workloads

Environment variables are available to the workload process:

```python
# Python
import os
db_url = os.environ['DATABASE_URL']
```

```javascript
// Node.js
const dbUrl = process.env.DATABASE_URL;
```

```go
// Go
import "os"
dbURL := os.Getenv("DATABASE_URL")
```

```bash
# Shell
echo $DATABASE_URL
```

## Limitations

- **Size**: Limited by kernel cmdline buffer (~2KB after compression)
- **Security**: Values visible in `/proc/cmdline` (plaintext)
- **Updates**: Requires VM restart (no runtime updates)

For production secrets, consider using a secrets manager and referencing vault paths via env vars.

## Examples

**Development vs Production:**
```bash
# Dev
volar vms create app-dev --image myapp \
  --env ENVIRONMENT=development \
  --env LOG_LEVEL=debug

# Prod
volar vms create app-prod --image myapp \
  --env ENVIRONMENT=production \
  --env LOG_LEVEL=error
```

**Feature Flags:**
```bash
volar vms create api --image myapp \
  --env FEATURE_NEW_API=true \
  --env RATE_LIMIT=1000
```

**Multi-Service Stack:**
```bash
# Database
volar vms create postgres --image postgres-db

# API with service references
volar vms create api --image myapp \
  --env DATABASE_URL=postgres://postgres.volant:5432/mydb \
  --env REDIS_URL=redis://cache.volant:6379
```
