# Environment Variables

## Overview

Environment variables in Volant allow you to configure your applications at runtime without rebuilding images. This follows the [12-Factor App](https://12factor.net/config) methodology for configuration management.

Environment variables can be set at two levels:

1. **Image defaults** (in `manifest.toml`)
2. **VM overrides** (at VM creation time)

VM overrides take precedence over image defaults, allowing you to use the same image across different environments (development, staging, production) with different configurations.

## Setting Defaults in Images

### In manifest.toml

Define default environment variables that will be available to all VMs created from this image:

```toml
[env]
LOG_LEVEL = "info"
DATABASE_URL = "postgres://localhost:5432/mydb"
API_KEY = "default-key"
PORT = "8080"
```

These values are embedded in the image manifest during the build process and become the defaults for any VM created from this image.

### Build and Install

```bash
# Build the image with fledge
sudo fledge build -c fledge.toml -m manifest.toml -o myapp.img

# Install the image
volar images install --manifest myapp.manifest.json
```

## Overriding at VM Creation

Override environment variables when creating a VM using the `--env` flag:

```bash
volar vms create myvm --image myapp \
  --env LOG_LEVEL=debug \
  --env DATABASE_URL=postgres://database.volant:5432/mydb \
  --env API_KEY=production-secret
```

### Multiple Environment Variables

You can specify multiple `--env` flags:

```bash
volar vms create api --image myapp \
  --env ENVIRONMENT=production \
  --env LOG_LEVEL=error \
  --env DATABASE_URL=postgres://prod-db.volant:5432/mydb \
  --env REDIS_URL=redis://cache.volant:6379 \
  --env API_KEY=prod-key-12345
```

## Override Precedence

The override hierarchy (highest to lowest priority):

1. **VM creation flags** (`--env` at creation time)
2. **Image manifest defaults** (`[env]` in manifest.toml)

Example:

```toml
# manifest.toml
[env]
LOG_LEVEL = "info"
DATABASE_URL = "postgres://localhost:5432/mydb"
```

```bash
# Create VM with overrides
volar vms create myvm --image myapp --env LOG_LEVEL=debug

# Result inside VM:
# LOG_LEVEL=debug        (overridden)
# DATABASE_URL=postgres://localhost:5432/mydb  (from manifest)
```

## Accessing in Workload

Environment variables are automatically available to your workload process:

### Shell Script

```bash
#!/bin/bash
echo "Log level: $LOG_LEVEL"
echo "Database: $DATABASE_URL"
```

### Python

```python
import os

log_level = os.environ.get('LOG_LEVEL', 'info')
database_url = os.environ['DATABASE_URL']

print(f"Connecting to {database_url}")
```

### Node.js

```javascript
const logLevel = process.env.LOG_LEVEL || 'info';
const databaseUrl = process.env.DATABASE_URL;

console.log(`Database: ${databaseUrl}`);
```

### Go

```go
import "os"

logLevel := os.Getenv("LOG_LEVEL")
databaseURL := os.Getenv("DATABASE_URL")

fmt.Printf("Database: %s\n", databaseURL)
```

## Use Cases

### 12-Factor Apps

Separate configuration from code, making it easy to deploy the same image across multiple environments:

```bash
# Development
volar vms create app-dev --image myapp \
  --env ENVIRONMENT=development \
  --env LOG_LEVEL=debug \
  --env DATABASE_URL=postgres://dev-db.volant:5432/mydb \
  --env DEBUG=true

# Staging
volar vms create app-staging --image myapp \
  --env ENVIRONMENT=staging \
  --env LOG_LEVEL=info \
  --env DATABASE_URL=postgres://staging-db.volant:5432/mydb

# Production
volar vms create app-prod --image myapp \
  --env ENVIRONMENT=production \
  --env LOG_LEVEL=error \
  --env DATABASE_URL=postgres://prod-db.volant:5432/mydb
```

### Service Discovery Integration

Combine environment variables with DNS service discovery:

```bash
# Create database VM
volar vms create postgres --image postgres-db

# Create API that references database by name
volar vms create api --image myapp \
  --env DATABASE_URL=postgres://postgres.volant:5432/mydb \
  --env CACHE_URL=redis://cache.volant:6379 \
  --env QUEUE_URL=amqp://rabbitmq.volant:5672
```

The hostnames (`postgres.volant`, `cache.volant`, etc.) automatically resolve to the correct IPs via Volant's built-in DNS server.

### Feature Flags

Control application features without rebuilding:

```bash
volar vms create api --image myapp \
  --env FEATURE_NEW_API=true \
  --env FEATURE_BETA_UI=false \
  --env RATE_LIMIT=1000
```

### Secret Management

Pass secrets securely at runtime (not baked into images):

```bash
volar vms create api --image myapp \
  --env API_KEY=$SECRET_API_KEY \
  --env JWT_SECRET=$SECRET_JWT_TOKEN \
  --env STRIPE_KEY=$SECRET_STRIPE_KEY
```

**Note:** For production workloads, consider using a proper secrets management solution (HashiCorp Vault, AWS Secrets Manager, etc.) and reference secrets via environment variables.

## Technical Implementation

### How It Works

1. **Build Time:** Environment variables from `manifest.toml` are embedded in the image manifest JSON.

2. **VM Creation:** The orchestrator:
   - Reads env vars from the image manifest
   - Merges with env vars from `--env` flags (overrides take precedence)
   - Encodes the final env map as base64 JSON
   - Passes it to the VM via kernel cmdline parameter `volant.env`

3. **Guest Boot:** The guest agent:
   - Reads `volant.env` from `/proc/cmdline`
   - Decodes the base64 JSON
   - Sets environment variables using `os.Setenv()`
   - Passes env vars to the workload process

### Kernel Cmdline Format

```
volant.env=<base64-encoded-json>
```

Example:
```
volant.env=eyJMT0dfTEVWRUwiOiJpbmZvIiwiREFUQUJBU0VfVVJMIjoicG9zdGdyZXM6Ly9sb2NhbGhvc3Q6NTQzMi9teWRiIn0=
```

Decodes to:
```json
{
  "LOG_LEVEL": "info",
  "DATABASE_URL": "postgres://localhost:5432/mydb"
}
```

## Best Practices

### 1. Use Descriptive Variable Names

```bash
# Good
DATABASE_URL=postgres://db.volant:5432/mydb
REDIS_CACHE_URL=redis://cache.volant:6379

# Avoid
DB=postgres://db.volant:5432/mydb
CACHE=redis://cache.volant:6379
```

### 2. Provide Sensible Defaults

Define defaults in `manifest.toml` for non-sensitive values:

```toml
[env]
LOG_LEVEL = "info"
PORT = "8080"
TIMEOUT = "30s"
```

### 3. Never Hardcode Secrets

```bash
# NEVER do this in manifest.toml:
[env]
API_KEY = "hardcoded-secret-key"  # ❌ Bad!

# Instead, provide at runtime:
volar vms create api --image myapp --env API_KEY=$SECRET_KEY  # ✅ Good
```

### 4. Use Service Discovery

Instead of hardcoded IPs, use DNS names:

```bash
# Bad: Hardcoded IP
--env DATABASE_URL=postgres://192.168.127.10:5432/mydb

# Good: DNS name
--env DATABASE_URL=postgres://database.volant:5432/mydb
```

### 5. Document Required Variables

In your image README, document required environment variables:

```markdown
## Required Environment Variables

- `DATABASE_URL`: PostgreSQL connection string (e.g., `postgres://db.volant:5432/mydb`)
- `API_KEY`: API authentication key
- `LOG_LEVEL`: Logging level (debug|info|warn|error) - default: info
```

## Troubleshooting

### Environment Variable Not Set

Check if the variable is defined in the manifest or passed via `--env`:

```bash
# View image manifest
volar images get myapp --output json | jq '.workload.env'

# Check inside VM
volar exec myvm -- env | grep MY_VAR
```

### Variable Has Wrong Value

Verify override precedence:

```bash
# Check image defaults
volar images get myapp --output json | jq '.workload.env.LOG_LEVEL'

# Check VM config
volar vms get myvm --output json | jq '.metadata.env.LOG_LEVEL'
```

### Workload Not Seeing Variables

Ensure the guest agent is running and has set the environment:

```bash
# Check agent logs
volar logs myvm

# Verify env in guest
volar exec myvm -- env | sort
```

## See Also

- [Service Discovery](./service-discovery.md) - DNS-based service discovery
- [Manifest Specification](../reference/manifest-spec.md) - Complete manifest format
- [12-Factor App Config](https://12factor.net/config) - Configuration best practices
