# Service Discovery via DNS

## Overview

Volant includes an embedded DNS server that provides automatic service discovery for VMs and deployments. This eliminates the need for hardcoded IP addresses or external service discovery tools like Consul or etcd.

### Key Features

- **Automatic registration**: VMs are automatically registered in DNS when created
- **Dynamic updates**: DNS records update as VMs start/stop (10-second TTL)
- **Round-robin load balancing**: Deployments return multiple IPs for client-side load balancing
- **Short names**: Works with both `service.volant` and `service` (via search domain)
- **No configuration required**: DNS is enabled by default and auto-configured in guests

## How It Works

1. **volantd starts** an embedded DNS server on `192.168.127.1:53`
2. **VMs boot** with `/etc/resolv.conf` pointing to `192.168.127.1`
3. **Applications query DNS** for `service.volant` or just `service`
4. **DNS server resolves** by querying the SQLite database for VM/deployment IPs
5. **Clients connect** using the resolved IP address(es)

## Configuration

DNS is enabled by default. You can customize it via environment variables:

```bash
# Disable DNS server
VOLANT_DNS_ENABLED=false volantd

# Custom bind address (default: 192.168.127.1:53)
VOLANT_DNS_LISTEN=192.168.127.1:53 volantd

# Custom domain suffix (default: volant)
VOLANT_DNS_DOMAIN=mycompany volantd

# Custom upstream DNS servers (default: 1.1.1.1:53,8.8.8.8:53)
VOLANT_DNS_UPSTREAMS=8.8.8.8:53,1.1.1.1:53 volantd
```

### Default Configuration

```bash
VOLANT_DNS_ENABLED=true                     # Enabled by default
VOLANT_DNS_LISTEN=192.168.127.1:53          # Host IP on bridge
VOLANT_DNS_DOMAIN=volant                    # Domain suffix
VOLANT_DNS_UPSTREAMS=1.1.1.1:53,8.8.8.8:53  # Fallback for external queries
```

## Resolution Rules

### VM Names → Single IP

When you create a VM, it's automatically registered in DNS:

```bash
volar vms create postgres --image postgres-db
# VM gets IP: 192.168.127.10
```

DNS resolution:
```bash
$ dig @192.168.127.1 +short postgres.volant
192.168.127.10
```

### Deployment Names → Multiple IPs (Round-Robin)

Deployments with multiple replicas return all IPs:

```bash
volar deployments create web --image nginx --replicas 3
# Creates: web-0 (192.168.127.11), web-1 (192.168.127.12), web-2 (192.168.127.13)
```

DNS resolution returns multiple A records:
```bash
$ dig @192.168.127.1 +short web.volant
192.168.127.11
192.168.127.12
192.168.127.13
```

Most HTTP clients (curl, browsers, application libraries) automatically round-robin across these IPs.

### Short Names (Search Domain)

VMs are configured with `search volant` in `/etc/resolv.conf`, so short names work:

```bash
# From inside a VM:
$ ping postgres
# Resolves to postgres.volant automatically

$ curl http://api
# Resolves to http://api.volant automatically
```

## Usage Examples

### In Environment Variables

Reference services by name when creating VMs:

```bash
volar vms create api --image myapp \
  --env DATABASE_URL=postgres://postgres.volant:5432/mydb \
  --env CACHE_URL=redis://cache.volant:6379 \
  --env QUEUE_URL=amqp://rabbitmq.volant:5672
```

The hostnames automatically resolve to the correct IPs.

### In Application Code

#### Python

```python
import psycopg2

# Uses DNS to resolve postgres.volant
conn = psycopg2.connect("postgres://admin:pass@postgres.volant:5432/mydb")
```

#### Node.js

```javascript
const { Client } = require('pg');

const client = new Client({
  host: 'postgres.volant',  // Resolves via DNS
  port: 5432,
  database: 'mydb'
});

await client.connect();
```

#### Go

```go
import "database/sql"
import _ "github.com/lib/pq"

// DNS resolves postgres.volant to IP
db, err := sql.Open("postgres", "postgres://admin:pass@postgres.volant:5432/mydb")
```

#### Curl

```bash
# From inside a VM
curl http://api.volant/v1/users
curl http://api/v1/users  # Short name works too
```

### Testing Resolution

#### From Host

```bash
# Query Volant's DNS server directly
dig @192.168.127.1 postgres.volant

# Use nslookup
nslookup postgres.volant 192.168.127.1

# Test with curl
curl http://192.168.127.1:8080  # Direct IP
curl http://api.volant:8080      # Via DNS (requires DNS in /etc/resolv.conf)
```

#### From Inside VM

```bash
# DNS is auto-configured, so standard tools work
nslookup postgres.volant
dig postgres.volant
ping postgres
curl http://api.volant
```

## Multi-Service Stack Example

Create a complete multi-service application:

```bash
# 1. Create database
volar vms create postgres --image postgres-db \
  --env POSTGRES_PASSWORD=secret

# 2. Create cache
volar vms create redis --image redis-cache

# 3. Create API that references database and cache by name
volar vms create api --image myapp \
  --env DATABASE_URL=postgres://postgres.volant:5432/mydb \
  --env REDIS_URL=redis://redis.volant:6379 \
  --env CACHE_TTL=3600

# 4. Create frontend that references API by name
volar vms create web --image myweb \
  --env API_URL=http://api.volant:8080

# All services can communicate using DNS names!
```

## Load Balancing with Deployments

Create a deployment with multiple replicas for load balancing:

```bash
# Create 3 API replicas
volar deployments create api --image myapp --replicas 3
# Creates: api-0, api-1, api-2

# Query DNS
dig @192.168.127.1 api.volant
# Returns 3 IPs (one for each replica)
```

Clients automatically round-robin:

```bash
# Each request may go to a different replica
for i in {1..6}; do
  curl http://api.volant/v1/health
done

# Requests distributed across api-0, api-1, api-2
```

## Dynamic Updates

DNS records update automatically as VMs start/stop:

```bash
# Create VM
volar vms create myvm --image myapp
dig @192.168.127.1 +short myvm.volant  # Returns IP

# Stop VM
volar vms stop myvm
dig @192.168.127.1 +short myvm.volant  # Returns NXDOMAIN (not found)

# Start VM again
volar vms start myvm
dig @192.168.127.1 +short myvm.volant  # Returns IP again
```

### TTL (Time To Live)

DNS responses have a **10-second TTL** to enable fast updates:

```bash
$ dig @192.168.127.1 myvm.volant

;; ANSWER SECTION:
myvm.volant.  10  IN  A  192.168.127.10
             ^^
             TTL = 10 seconds
```

This means:
- Clients cache DNS responses for only 10 seconds
- When VMs are created/destroyed, new DNS queries reflect changes within 10s
- No stale DNS entries for long-running services

## Guest DNS Configuration

VMs are automatically configured with DNS during boot. The guest agent:

1. Reads `volant.dns_server` and `volant.dns_search` from kernel cmdline
2. Creates `/etc/resolv.conf`:

```
nameserver 192.168.127.1
search volant
```

No manual configuration needed!

### Verify DNS Configuration

Check inside a VM:

```bash
$ volar exec myvm -- cat /etc/resolv.conf
nameserver 192.168.127.1
search volant

$ volar exec myvm -- nslookup postgres.volant
Server:    192.168.127.1
Address:   192.168.127.1#53

Name:      postgres.volant
Address:   192.168.127.10
```

## Architecture

### DNS Server

- **Location**: Runs inside `volantd` process
- **Port**: 53 (UDP)
- **Bind Address**: 192.168.127.1 (host IP on bridge)
- **Backend**: SQLite database queries
- **Protocol**: Standard DNS (RFC 1035)

### Query Flow

```
┌─────────────┐          ┌─────────────┐          ┌──────────┐
│   VM Guest  │          │ DNS Server  │          │ SQLite   │
│             │          │ (volantd)   │          │ Database │
└──────┬──────┘          └──────┬──────┘          └─────┬────┘
       │                        │                       │
       │ Query postgres.volant  │                       │
       ├───────────────────────>│                       │
       │                        │                       │
       │                        │ SELECT * FROM vms    │
       │                        │ WHERE name='postgres'│
       │                        ├──────────────────────>│
       │                        │                       │
       │                        │ Returns VM record     │
       │                        │ (IP: 192.168.127.10) │
       │                        │<──────────────────────┤
       │                        │                       │
       │ DNS Response:          │                       │
       │ A 192.168.127.10       │                       │
       │<───────────────────────┤                       │
       │                        │                       │
```

### Database Schema

DNS resolution queries these tables:

```sql
-- VMs table
SELECT ip_address FROM vms
WHERE name = 'postgres' AND status = 'running';

-- VM Groups (deployments) table
SELECT v.ip_address FROM vms v
JOIN vm_groups g ON v.group_id = g.id
WHERE g.name = 'web' AND v.status = 'running';
```

## Best Practices

### 1. Use DNS Names, Not IPs

```bash
# Bad: Hardcoded IP
--env DATABASE_URL=postgres://192.168.127.10:5432/mydb

# Good: DNS name
--env DATABASE_URL=postgres://postgres.volant:5432/mydb
```

### 2. Use Consistent Naming

```bash
# Good naming convention
volar vms create postgres-primary --image postgres
volar vms create postgres-replica --image postgres
volar vms create redis-cache --image redis

# Applications can reference predictable names
DATABASE_URL=postgres://postgres-primary.volant:5432/mydb
CACHE_URL=redis://redis-cache.volant:6379
```

### 3. Use Deployments for Load Balancing

```bash
# Create replicated service
volar deployments create api --image myapp --replicas 3

# Clients get automatic round-robin
API_URL=http://api.volant:8080
```

### 4. Test DNS Before Deploying

```bash
# Verify DNS works from host
dig @192.168.127.1 +short myservice.volant

# Verify DNS works from guest
volar exec myvm -- nslookup myservice.volant
```

## Troubleshooting

### DNS Server Not Starting

Check if volantd started successfully:

```bash
# Check volantd logs
journalctl -u volantd -f

# Verify DNS is enabled
ps aux | grep volantd
# Should not see VOLANT_DNS_ENABLED=false

# Check if port 53 is already in use
sudo lsof -i :53
```

### DNS Queries Not Resolving

```bash
# Test from host
dig @192.168.127.1 +short myvm.volant

# If NXDOMAIN (not found):
# 1. Verify VM exists and is running
volar vms list
volar vms get myvm

# 2. Check database
sqlite3 ~/.volant/state.db "SELECT name, ip_address, status FROM vms;"
```

### Guest Cannot Resolve Names

Check `/etc/resolv.conf` inside VM:

```bash
volar exec myvm -- cat /etc/resolv.conf
# Should contain:
# nameserver 192.168.127.1
# search volant

# If missing, check agent logs
volar logs myvm
```

### Wrong IP Returned

```bash
# Check database state
volar vms get myvm --output json | jq '.ip_address'

# Force DNS cache clear (10s TTL)
sleep 11
dig @192.168.127.1 +short myvm.volant
```

### Deployment Not Round-Robining

```bash
# Verify all replicas are running
volar deployments get web --output json | jq '.vms[] | {name, ip, status}'

# Test DNS returns multiple IPs
dig @192.168.127.1 web.volant
# Should see multiple A records

# Test client round-robin
for i in {1..10}; do
  curl -s http://web.volant/healthz | grep hostname
done
# Should see requests distributed across replicas
```

## Limitations

### No External DNS Forwarding (Yet)

Currently, Volant DNS only resolves `*.volant` queries. External queries (e.g., `google.com`) are not forwarded to upstream servers.

**Workaround**: VMs can query external DNS directly:
```bash
# Use custom DNS for external queries
dig @1.1.1.1 google.com
```

**Future**: Upstream forwarding planned for Phase 2.

### No SRV Records

Only A records (IPv4) are currently supported. SRV records for service discovery (with port information) are not yet implemented.

### No DNSSEC

DNSSEC validation is not currently supported.

## Advanced Configuration

### Custom Domain

Use a custom domain suffix:

```bash
VOLANT_DNS_DOMAIN=mycompany volantd

# Now services resolve as:
postgres.mycompany  → 192.168.127.10
api.mycompany       → 192.168.127.11
```

Update application configs accordingly:
```bash
volar vms create api --image myapp \
  --env DATABASE_URL=postgres://postgres.mycompany:5432/mydb
```

### Disable DNS

If you prefer to use an external DNS server:

```bash
VOLANT_DNS_ENABLED=false volantd
```

Then configure VMs manually:
```bash
volar vms create myvm --image myapp \
  --kernel-args "volant.dns_server=8.8.8.8 volant.dns_search=example.com"
```

## See Also

- [Environment Variables](./environment-variables.md) - Configure apps with env vars
- [Networking Architecture](../architecture/networking.md) - Bridge, vsock, and L4 routing
- [Database Schema](../reference/database-schema.md) - SQLite schema for DNS resolution
- [DNS RFC 1035](https://www.ietf.org/rfc/rfc1035.txt) - DNS protocol specification
