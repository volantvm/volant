---
title: "Service Discovery (DNS)"
author: "VolantVM"
date: "2025-01-07"
---

Ground truth: internal/server/dns/server.go (DNS server), internal/server/config/config.go (DNS config), cmd/volantd/main.go (DNS startup), internal/agent/app/app.go (configureDNSFromCmdline).

Volant includes an embedded DNS server for automatic service discovery. VMs and deployments are registered in DNS when created, eliminating the need for hardcoded IPs or external service discovery tools.

## How It Works

1. volantd starts DNS server on `192.168.127.1:53` (configurable)
2. Orchestrator injects DNS parameters into VM kernel cmdline
3. Guest agent creates `/etc/resolv.conf` pointing to `192.168.127.1`
4. Applications query DNS for `service.volant` or short names
5. DNS server queries SQLite database for VM/deployment IPs
6. Responses include only running VMs with 10-second TTL

## Resolution Rules

**VM names** → single IP:
```bash
volar vms create postgres --image postgres-db
# postgres.volant → 192.168.127.10
```

**Deployment names** → multiple IPs (round-robin):
```bash
volar deployments create web --image nginx --replicas 3
# web.volant → [192.168.127.11, 192.168.127.12, 192.168.127.13]
```

**Short names** work via search domain:
```bash
# Inside VM: postgres → postgres.volant
# Due to "search volant" in /etc/resolv.conf
```

## Configuration

DNS is enabled by default. Configure via environment variables:

```bash
VOLANT_DNS_ENABLED=true                     # Default: enabled
VOLANT_DNS_LISTEN=192.168.127.1:53          # Default: host IP
VOLANT_DNS_DOMAIN=volant                    # Default: .volant suffix
VOLANT_DNS_UPSTREAMS=1.1.1.1:53,8.8.8.8:53  # Reserved for future
```

To disable:
```bash
VOLANT_DNS_ENABLED=false volantd
```

## Implementation

**DNS Server** (dns/server.go):
- Listens on UDP port 53
- Handles A record queries for `*.volant`
- Calls `resolveService()` which queries:
  - `VMs().GetByName(name)` for single VMs
  - `VMGroups().GetByName(name)` then `VMs().ListByGroupID()` for deployments
- Returns only VMs with `status == "running"`
- TTL: 10 seconds for dynamic updates

**Guest Configuration** (agent/app.go:549-595):
- `configureDNSFromCmdline()` runs early in boot
- Reads `volant.dns_server` and `volant.dns_search` from `/proc/cmdline`
- Creates `/etc/resolv.conf`:
  ```
  nameserver 192.168.127.1
  search volant
  ```

**Orchestrator** (orchestrator.go:439-440):
- Injects DNS parameters into kernel cmdline:
  ```go
  cmdlineArgs["volant.dns_server"] = e.hostIP.String()
  cmdlineArgs["volant.dns_search"] = "volant"
  ```

## Usage Examples

**Multi-service stack:**
```bash
# Create services
volar vms create postgres --image postgres-db
volar vms create redis --image redis-cache

# Create API that references them by name
volar vms create api --image myapp \
  --env DATABASE_URL=postgres://postgres.volant:5432/mydb \
  --env CACHE_URL=redis://redis.volant:6379
```

**Test resolution:**
```bash
# From host
dig @192.168.127.1 +short postgres.volant
# Returns: 192.168.127.10

# From inside VM
volar exec api -- nslookup postgres.volant
# Resolves via 192.168.127.1
```

**Load balancing:**
```bash
# Create deployment with 3 replicas
volar deployments create api --image myapp --replicas 3

# DNS returns 3 A records
dig @192.168.127.1 api.volant
# Clients automatically round-robin across IPs
```

## Dynamic Updates

DNS records update automatically as VMs start/stop:

```bash
volar vms create myvm --image myapp
dig @192.168.127.1 +short myvm.volant  # Returns IP

volar vms stop myvm
dig @192.168.127.1 +short myvm.volant  # Returns NXDOMAIN

volar vms start myvm
dig @192.168.127.1 +short myvm.volant  # Returns IP again
```

The 10-second TTL ensures clients pick up changes quickly.

## Application Code

Applications use standard DNS resolution:

```python
# Python
import psycopg2
conn = psycopg2.connect("postgres://admin:pass@postgres.volant:5432/mydb")
```

```javascript
// Node.js
const { Client } = require('pg');
const client = new Client({ host: 'postgres.volant', port: 5432 });
```

```go
// Go
import "database/sql"
db, _ := sql.Open("postgres", "postgres://user:pass@postgres.volant:5432/mydb")
```

## Limitations

- **IPv4 only**: No AAAA records (IPv6) yet
- **No upstream forwarding**: External queries (e.g., `google.com`) not forwarded
- **No SRV records**: Port discovery not supported
- **No DNSSEC**: Validation not implemented

Workaround for external queries: Applications can query `1.1.1.1` or `8.8.8.8` directly.

## Troubleshooting

**DNS server not starting:**
```bash
# Check if port 53 is available
sudo lsof -i :53

# Verify volantd started with DNS enabled
ps aux | grep volantd | grep -v DNS_ENABLED=false
```

**Resolution not working:**
```bash
# Verify VM exists and is running
volar vms list
volar vms get myvm

# Check database
sqlite3 ~/.volant/state.db "SELECT name, ip_address, status FROM vms;"

# Test from host
dig @192.168.127.1 +short myvm.volant
```

**Guest cannot resolve:**
```bash
# Check /etc/resolv.conf
volar exec myvm -- cat /etc/resolv.conf
# Should contain: nameserver 192.168.127.1

# Check agent logs
volar logs myvm | grep -i dns
```
