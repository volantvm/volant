# Docker Compose Migration Guide

## Overview

`volant-compose` is a CLI tool that converts Docker Compose files (`docker-compose.yml`) to Volant stack definitions (`volant.yaml`). It bridges the gap between Docker's container-centric model and Volant's VM-centric architecture, enabling migration from Docker Compose to Volant's microVM orchestration platform.

## Why volant-compose?

Docker Compose is widely used but has security limitations due to shared kernel architecture. Volant provides VM-level isolation with container-like speed (200-500ms boot times), but requires a different configuration format. This tool automates the conversion process while handling the architectural differences between containers and microVMs.

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/volantvm/volant.git
cd volant

# Build volant-compose
make build-compose

# The binary will be in bin/volant-compose
./bin/volant-compose --version
```

### Manual Build

```bash
go build -o volant-compose ./cmd/volant-compose
```

## Quick Start

### Basic Conversion

```bash
# Convert docker-compose.yml to volant.yaml
volant-compose convert -f docker-compose.yml -o volant.yaml

# With custom stack name
volant-compose convert -f docker-compose.yml -o volant.yaml --name my-stack

# Verbose output
volant-compose convert -f docker-compose.yml -v
```

### Validation

Check if your docker-compose.yml can be converted before actually converting:

```bash
volant-compose validate -f docker-compose.yml
```

## Supported Features

| Feature | Docker Compose | Volant | Status |
|---------|---------------|---------|--------|
| **Services** | container-centric | VM-centric | ✅ Supported |
| **Image** | `image: nginx:latest` | `image: nginx:latest` | ✅ Direct mapping |
| **Ports** | `"8080:80"` | `"8080:80"` | ✅ Direct mapping |
| **Environment** | Array or map | Map | ✅ Converted |
| **Volumes** | Named volumes | Named volumes | ✅ Supported |
| **Dependencies** | `depends_on` | `depends_on` | ✅ Direct mapping |
| **Command** | `command` | `args` | ✅ Converted |
| **Entrypoint** | `entrypoint` | `entrypoint` | ✅ Direct mapping |
| **Networks** | Custom networks | Single subnet | ⚠️ Simplified |

## Unsupported Features

The following Docker Compose features are **not supported** and will cause conversion errors:

### Build Contexts
```yaml
# ❌ NOT SUPPORTED
services:
  app:
    build:
      context: ./app
      dockerfile: Dockerfile
```

**Solution:** Use Fledge to build images separately:
```bash
fledge build -f app/Dockerfile --name myapp --version 1.0.0
volant image import myapp:1.0.0
```

### Bind Mounts
```yaml
# ❌ NOT SUPPORTED
services:
  web:
    volumes:
      - ./html:/usr/share/nginx/html
```

**Solution:** Use named volumes:
```yaml
# ✅ SUPPORTED
services:
  web:
    volumes:
      - html-data:/usr/share/nginx/html

volumes:
  html-data:
```

### Replicas
```yaml
# ❌ NOT SUPPORTED (Phase 2)
services:
  api:
    deploy:
      replicas: 3
```

**Solution:** Phase 2 feature (horizontal scaling)

### Other Unsupported Features
- Health checks (Track D - Stack Orchestrator)
- Secrets (Track A - Environment Variables)
- Configs
- Docker Swarm deployment settings
- Resource limits (memory/cpu limits in deploy section)

## Docker Compose → Volant Mapping

### Container Name → VM Name
```yaml
# Docker Compose
services:
  web:
    container_name: my-web

# Volant (service name becomes VM identifier)
services:
  web:
    image: nginx:latest
```

### Environment Variables
```yaml
# Docker Compose (array format)
services:
  app:
    environment:
      - DATABASE_URL=postgres://localhost/db
      - API_KEY=secret

# Volant (map format)
services:
  app:
    environment:
      DATABASE_URL: postgres://localhost/db
      API_KEY: secret
```

### Service Discovery
```yaml
# Docker Compose
services:
  web:
    environment:
      DATABASE_URL: postgres://db:5432/mydb

# Volant (use .svc.volant DNS suffix)
services:
  web:
    environment:
      DATABASE_URL: postgres://db.svc.volant:5432/mydb
```

### Memory and CPU
```yaml
# Volant adds VM resource allocations
services:
  web:
    image: nginx:latest
    memory: 512  # MB (default: 512)
    vcpus: 1     # CPU count (default: 1)
```

## Examples

### Example 1: WordPress + MySQL

**Input: docker-compose.yml**
```yaml
version: "3.8"

services:
  wordpress:
    image: wordpress:latest
    ports:
      - "8080:80"
    environment:
      WORDPRESS_DB_HOST: db
      WORDPRESS_DB_USER: wordpress
      WORDPRESS_DB_PASSWORD: secret
    depends_on:
      - db
    volumes:
      - wordpress-data:/var/www/html

  db:
    image: mysql:8.0
    environment:
      MYSQL_DATABASE: wordpress
      MYSQL_USER: wordpress
      MYSQL_PASSWORD: secret
      MYSQL_ROOT_PASSWORD: rootsecret
    volumes:
      - db-data:/var/lib/mysql

volumes:
  wordpress-data:
  db-data:
```

**Output: volant.yaml**
```yaml
version: "1.0"
name: wordpress-stack
services:
  wordpress:
    image: wordpress:latest
    memory: 512
    vcpus: 1
    ports:
      - 8080:80
    environment:
      WORDPRESS_DB_HOST: db
      WORDPRESS_DB_USER: wordpress
      WORDPRESS_DB_PASSWORD: secret
    volumes:
      - wordpress-data:/var/www/html
    depends_on:
      - db
  db:
    image: mysql:8.0
    memory: 512
    vcpus: 1
    environment:
      MYSQL_DATABASE: wordpress
      MYSQL_USER: wordpress
      MYSQL_PASSWORD: secret
      MYSQL_ROOT_PASSWORD: rootsecret
    volumes:
      - db-data:/var/lib/mysql
networks:
  default:
    subnet: 192.168.127.0/24
    gateway: 192.168.127.1
volumes:
  wordpress-data:
    size: 10G
  db-data:
    size: 10G
```

### Example 2: Node.js + MongoDB

**Input: docker-compose.yml**
```yaml
version: "3.8"

services:
  app:
    image: node:18-alpine
    ports:
      - "3000:3000"
    environment:
      NODE_ENV: production
      MONGO_URL: mongodb://mongo:27017/myapp
    depends_on:
      - mongo
    command: ["node", "server.js"]

  mongo:
    image: mongo:7
    environment:
      - MONGO_INITDB_ROOT_USERNAME=admin
      - MONGO_INITDB_ROOT_PASSWORD=secret
    volumes:
      - mongo-data:/data/db

volumes:
  mongo-data:
```

**Conversion:**
```bash
volant-compose convert -f docker-compose.yml -o volant.yaml --name nodejs-stack
```

## Migration Guide

### Step 1: Audit Your docker-compose.yml

Check for unsupported features:
```bash
volant-compose validate -f docker-compose.yml
```

If you see errors:
- Remove `build:` directives (use Fledge separately)
- Replace bind mounts with named volumes
- Remove `deploy.replicas` settings

### Step 2: Build Images with Fledge

If your docker-compose.yml uses `build:` directives, build images with Fledge first:

```bash
# For each service with build: directive
cd service-directory
fledge build -f Dockerfile --name service-name --version 1.0.0

# Import to Volant
volant image import service-name:1.0.0
```

### Step 3: Convert Compose File

```bash
volant-compose convert -f docker-compose.yml -o volant.yaml --name my-stack
```

### Step 4: Review Generated volant.yaml

Check:
- ✅ Service names are correct
- ✅ Environment variables are preserved
- ✅ Volumes are named volumes (not bind mounts)
- ✅ Dependencies are correct
- ✅ Port mappings are correct

### Step 5: Deploy (Phase 2)

```bash
# Once Phase 2 (Track D - Stack Orchestrator) is complete:
volant stack deploy -f volant.yaml
```

## Troubleshooting

### Error: 'build' directive not supported

**Problem:**
```
service 'app': 'build' directive not supported
   → Use Fledge to build images: fledge build -f Dockerfile
```

**Solution:**
1. Build the image with Fledge:
   ```bash
   fledge build -f app/Dockerfile --name myapp --version 1.0.0
   ```
2. Update docker-compose.yml to use the built image:
   ```yaml
   services:
     app:
       image: myapp:1.0.0  # Remove build: directive
   ```
3. Convert again

### Error: bind mount not supported

**Problem:**
```
service 'web': bind mount './html:/data' not supported
   → Use named volumes instead: volumes: [data:/data]
```

**Solution:**
Replace bind mount with named volume:
```yaml
# Before
services:
  web:
    volumes:
      - ./html:/usr/share/nginx/html

# After
services:
  web:
    volumes:
      - html-data:/usr/share/nginx/html

volumes:
  html-data:
```

### Error: deploy.replicas not supported

**Problem:**
```
service 'api': deploy.replicas=3 not supported yet
   → Phase 2 feature (horizontal scaling)
```

**Solution:**
Remove replicas for now. This will be supported in Phase 2.

## CLI Reference

### Global Flags

```
-h, --help      Show help
-v, --version   Show version
```

### convert Command

Convert a docker-compose.yml file to volant.yaml

**Usage:**
```bash
volant-compose convert [flags]
```

**Flags:**
```
-f, --file string      Input docker-compose.yml file (default: docker-compose.yml)
-o, --output string    Output volant.yaml file (default: volant.yaml)
-n, --name string      Stack name (default: derived from directory name)
-v, --verbose          Verbose output
-h, --help            Help for convert
```

**Examples:**
```bash
# Basic conversion
volant-compose convert

# Custom files
volant-compose convert -f compose.yml -o stack.yaml

# Custom stack name
volant-compose convert --name production-stack

# Verbose output
volant-compose convert -v
```

### validate Command

Validate that a docker-compose.yml file can be converted

**Usage:**
```bash
volant-compose validate [flags]
```

**Flags:**
```
-f, --file string   Input docker-compose.yml file (default: docker-compose.yml)
-v, --verbose       Verbose output
-h, --help         Help for validate
```

**Examples:**
```bash
# Validate default file
volant-compose validate

# Validate specific file
volant-compose validate -f my-compose.yml

# Verbose validation
volant-compose validate -v
```

## Architecture Notes

### Impedance Mismatches

Docker Compose and Volant have different architectural philosophies:

| Aspect | Docker Compose | Volant |
|--------|---------------|---------|
| **Isolation** | Container (shared kernel) | MicroVM (isolated kernel) |
| **Boot Time** | <1s | 200-500ms |
| **Resource Model** | Soft limits | Hard allocation |
| **Networking** | Bridge + custom networks | Static IP pool (192.168.127.0/24) |
| **Service Discovery** | Docker DNS | DNS (.svc.volant) |

### Design Decisions

1. **Named Volumes Only:** Bind mounts require host filesystem access, which violates VM isolation principles.

2. **Single Subnet:** Volant Phase 1 uses a single subnet (192.168.127.0/24). Custom networks are simplified.

3. **Static Memory/CPU:** Each service gets VM resource allocations (default: 512MB, 1 vCPU).

4. **No Build Support:** Image building is delegated to Fledge to maintain separation of concerns.

## FAQ

**Q: Can I use my existing docker-compose.yml without changes?**

A: Most docker-compose.yml files work with minimal changes. The main issues are build contexts and bind mounts, which need to be addressed.

**Q: What happens to my custom networks?**

A: Volant Phase 1 uses a single default subnet (192.168.127.0/24). Custom network definitions are ignored but services can still communicate.

**Q: How do I set memory and CPU for services?**

A: The converter applies defaults (512MB, 1 vCPU). You can manually edit the generated volant.yaml to adjust these values.

**Q: Can I convert back from volant.yaml to docker-compose.yml?**

A: Not currently. The conversion is one-way due to VM-specific features in Volant.

**Q: What about Docker Compose v2 or v1?**

A: Only v3.x is supported. If you have v1 or v2, upgrade to v3 first using `docker-compose config`.

## Contributing

Contributions are welcome! See the main Volant repository for contribution guidelines.

## License

Same as Volant project.

## Support

- GitHub Issues: https://github.com/volantvm/volant/issues
- Documentation: https://github.com/volantvm/volant/tree/main/docs

## Related Documentation

- [Getting Started Guide](../2_getting-started/1_installation.md) - Installation and setup
- [Networking Guide](./1_networking.md) - Network configuration
- [Troubleshooting](./5_troubleshooting.md) - Common issues and solutions
