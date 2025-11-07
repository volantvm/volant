# volant-compose

Convert Docker Compose files to Volant stack definitions.

## Quick Start

```bash
# Convert docker-compose.yml to volant.yaml
volant-compose convert -f docker-compose.yml -o volant.yaml

# Validate before converting
volant-compose validate -f docker-compose.yml

# Get help
volant-compose --help
```

## Installation

```bash
# Build from source
make build-compose

# Binary will be in bin/volant-compose
./bin/volant-compose --version
```

## What is volant-compose?

`volant-compose` bridges the gap between Docker Compose and Volant, converting container-centric configurations to VM-centric stack definitions. It handles the architectural differences between Docker's shared-kernel model and Volant's isolated microVM approach.

## Features

✅ **Supported:**
- Services (image, ports, environment, volumes, depends_on)
- Named volumes
- Port mappings
- Environment variables (array and map formats)
- Command and entrypoint overrides

❌ **Not Supported:**
- Build contexts (use Fledge)
- Bind mounts (use named volumes)
- deploy.replicas (Phase 2)
- Multiple custom networks (simplified to single subnet)

## Documentation

See [docs/volant-compose.md](../../docs/volant-compose.md) for complete documentation including:
- Installation guide
- Usage examples
- Migration guide
- Troubleshooting
- API reference

## Examples

### WordPress + MySQL

```bash
cd testdata/wordpress
volant-compose convert -f docker-compose.yml -o volant.yaml
```

### Node.js + MongoDB

```bash
cd testdata/nodejs-mongo
volant-compose convert -f docker-compose.yml -o volant.yaml
```

## Architecture

volant-compose consists of three main packages:

- **pkg/compose**: Docker Compose YAML parser
- **pkg/volant**: Volant stack YAML schema
- **pkg/converter**: Impedance mismatch handler

See the [Docker Compose Migration Guide](../../docs/3_guides/6_docker-compose-migration.md) for detailed conversion rules and examples.

## Testing

```bash
# Unit tests
go test ./pkg/compose/...
go test ./pkg/volant/...
go test ./pkg/converter/...

# Integration tests
go test github.com/volantvm/volant/pkg/converter -run Integration
```

## License

Same as Volant project.
