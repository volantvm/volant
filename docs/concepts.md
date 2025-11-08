# Volant Core Concepts

## Stacks

A **stack** is a multi-VM application that works as a cohesive unit. Think of it as your application's architecture blueprint, defining how different services interact and depend on each other.

### What is a Stack?

A stack consists of:
- **Services**: Individual VMs running specific components (web server, database, cache, etc.)
- **Networks**: Virtual networks connecting the services
- **Volumes**: Persistent storage shared between services
- **Configuration**: Environment variables and runtime settings

### Stack Lifecycle

1. **Define**: Create a `volant.yaml` or `docker-compose.yml` file
2. **Deploy**: Use `volant stack up` to create and start all services
3. **Manage**: Start, stop, scale, and monitor the stack
4. **Update**: Modify configuration and redeploy
5. **Teardown**: Use `volant stack down` to stop and remove all resources

### Example Stack Definition

```yaml
# volant.yaml
name: my-app
services:
  web:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./html:/usr/share/nginx/html
    environment:
      - API_URL=http://api:3000

  api:
    build: ./api
    ports:
      - "3000:3000"
    environment:
      - DB_HOST=postgres
      - DB_NAME=myapp

  postgres:
    image: postgres:15
    volumes:
      - db-data:/var/lib/postgresql/data
    environment:
      - POSTGRES_DB=myapp
      - POSTGRES_PASSWORD=secret

volumes:
  db-data:
```

### Stack vs Individual VMs

| Feature | Stack | Individual VM |
|---------|-------|---------------|
| **Use Case** | Multi-service applications | Single-purpose services |
| **Management** | Managed as a unit | Managed individually |
| **Networking** | Automatic service discovery | Manual configuration |
| **Scaling** | Scale entire services | Scale single instances |
| **Configuration** | Declarative YAML | Imperative commands |

## Virtual Machines (VMs)

Volant VMs are lightweight, hardware-isolated virtual machines powered by Cloud Hypervisor. Each VM runs in its own secure sandbox with dedicated resources.

### VM Characteristics

- **Fast Boot**: Sub-second startup times
- **Secure**: Hardware-level isolation via KVM
- **Efficient**: Minimal overhead, near-native performance
- **Rootless**: Can run without root privileges (with proper setup)

### VM States

- **Created**: VM defined but not started
- **Running**: VM actively executing
- **Paused**: VM suspended but keeping memory state
- **Stopped**: VM shut down cleanly
- **Failed**: VM encountered an error

## Images

Images are the templates from which VMs are created. Volant supports multiple image formats:

### Image Types

1. **OCI Images**: Standard Docker/container images
2. **Volant Images**: Optimized VM images with Volant agent
3. **Custom Images**: Built from Dockerfiles or Volantfiles

### Image Layers

```
┌─────────────────────┐
│   Application Code  │  <- Your app
├─────────────────────┤
│   Volant Agent      │  <- Management layer
├─────────────────────┤
│   Root Filesystem   │  <- OS files
├─────────────────────┤
│   Kernel (shared)   │  <- Host kernel
└─────────────────────┘
```

## Volumes

Volumes provide persistent storage that survives VM restarts and can be shared between VMs in a stack.

### Volume Types

- **Named Volumes**: Managed by Volant, stored in `/var/lib/volant/volumes`
- **Bind Mounts**: Direct mapping to host directories
- **Tmpfs**: In-memory temporary storage

### Volume Lifecycle

```bash
# Create a volume
volant volume create my-data

# Use in a stack
services:
  app:
    volumes:
      - my-data:/data

# Backup a volume
volant volume export my-data > backup.tar

# Restore a volume
volant volume import my-data < backup.tar
```

## Networks

Volant provides virtual networking between VMs with automatic service discovery.

### Network Features

- **Isolation**: Each stack gets its own network namespace
- **Service Discovery**: Services can reach each other by name
- **Port Mapping**: Expose services to the host
- **Load Balancing**: Distribute traffic across scaled services

### Network Modes

1. **Bridge Mode** (default): VMs on virtual bridge with NAT
2. **Host Mode**: VM shares host network namespace
3. **None**: No networking

## Volantfile

The Volantfile is a unified configuration format that defines both build-time and runtime settings for your application.

### Volantfile Structure

```toml
[image]
name = "my-app"
version = "1.0"

[build]
strategy = "dockerfile"
dockerfile = "Dockerfile"
context = "."

[runtime]
cpu_cores = 2
memory_mb = 512
entrypoint = ["/app/server"]

[runtime.env]
PORT = "8080"
LOG_LEVEL = "info"

[runtime.expose]
ports = [8080]
```

### Benefits of Volantfile

- **Single Source of Truth**: One file for all configuration
- **Build + Runtime**: Defines both how to build and how to run
- **Portable**: Works across different environments
- **Version Controlled**: Track changes with your code

## Architecture Overview

```
┌──────────────────────────────────────┐
│          Volant CLI                  │  <- User interface
├──────────────────────────────────────┤
│          Volantd (Daemon)            │  <- Orchestration layer
├──────────────────────────────────────┤
│     Cloud Hypervisor | Firecracker   │  <- Hypervisor layer
├──────────────────────────────────────┤
│            KVM (Kernel)              │  <- Virtualization
└──────────────────────────────────────┘
```

## Best Practices

### Stack Design

1. **Single Responsibility**: Each service should do one thing well
2. **Stateless Services**: Keep state in databases/volumes
3. **Health Checks**: Define health checks for each service
4. **Resource Limits**: Set appropriate CPU/memory limits
5. **Environment Variables**: Use env vars for configuration

### Security

1. **Least Privilege**: Run services with minimal permissions
2. **Network Segmentation**: Use separate networks for different tiers
3. **Secrets Management**: Never hardcode secrets in images
4. **Regular Updates**: Keep base images updated

### Performance

1. **Right-size Resources**: Don't over-provision CPU/memory
2. **Use Caching**: Cache layers during builds
3. **Optimize Images**: Remove unnecessary files
4. **Monitor Metrics**: Track resource usage

## Comparison with Other Tools

### Volant vs Docker

| Feature | Volant | Docker |
|---------|--------|--------|
| **Isolation** | Hardware (VM) | OS-level (container) |
| **Security** | Strong isolation | Shared kernel |
| **Performance** | Near-native | Native |
| **Compatibility** | Linux guests | Any Linux process |
| **Resource Overhead** | ~50MB per VM | ~10MB per container |

### Volant vs Traditional VMs

| Feature | Volant | Traditional VMs |
|---------|--------|-----------------|
| **Boot Time** | <1 second | 30-60 seconds |
| **Memory Overhead** | ~50MB | ~500MB |
| **Disk Usage** | Shared layers | Full OS copy |
| **Management** | Simple CLI | Complex tools |

## Glossary

- **Stack**: A multi-VM application managed as a unit
- **Service**: A component of a stack (e.g., web server, database)
- **Image**: Template for creating VMs
- **Volume**: Persistent storage
- **Network**: Virtual network connecting VMs
- **Volantfile**: Unified configuration file
- **Volantd**: The Volant daemon that manages VMs
- **Agent**: Process inside VM for management
- **Drift**: Configuration synchronization system