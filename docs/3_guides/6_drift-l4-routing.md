# Drift L4 Routing

Drift is an optional eBPF-based L4 packet router that provides high-performance port forwarding for Volant microVMs.

## Overview

By default, Volant uses userspace vsock proxy processes to forward network traffic to VMs. While functional, this approach incurs context switching overhead for every packet. Drift replaces these userspace proxies with eBPF programs that rewrite packets directly in the kernel, dramatically reducing latency and CPU usage.

### Performance Comparison

**Without Drift (vsock proxy):**
- Userspace process per port mapping
- Context switches for every packet
- Latency: ~100μs per packet
- CPU overhead: moderate

**With Drift (eBPF):**
- Kernel-space packet rewriting
- Zero context switches
- Latency: ~10μs per packet (10x faster)
- CPU overhead: minimal

**Expected Performance Gains:**
- **Throughput**: 2-10x higher requests/second
- **Latency**: 10x lower packet forwarding time
- **CPU usage**: 50% lower overhead
- **Scalability**: Support for thousands of concurrent connections

## When to Use Drift

Drift provides significant benefits for workloads that are:

1. **Network-intensive**: High packet rates (web servers, APIs, proxies)
2. **Latency-sensitive**: Real-time applications, gaming servers, trading systems
3. **High-scale**: Many concurrent connections or port mappings

Drift may not be necessary for:
- Development/testing environments with light traffic
- Workloads dominated by computation rather than I/O
- Systems where vsock proxy performance is sufficient

## Architecture

Drift consists of three components:

### 1. eBPF Dataplane
- **Ingress program**: Rewrites destination IP/port for incoming packets
- **Egress program**: Rewrites source IP/port for return packets
- **Connection tracking**: Maintains state for bidirectional flows
- **Metrics**: Tracks packets/bytes processed

### 2. Controller
- Manages route lifecycle (create/update/delete)
- Synchronizes routes between persistent storage and eBPF maps
- Validates routes and prevents conflicts (duplicate ports)
- Provides health checks and metrics API

### 3. HTTP API
- RESTful interface for route management
- Integrates with Volant orchestrator
- Exposes metrics endpoint for monitoring

## Configuration

### Starting Drift

```bash
# Start the Drift daemon
driftd --interface vbr0 --port 9090
```

Options:
- `--interface`: Bridge interface to attach eBPF programs (default: vbr0)
- `--port`: HTTP API port (default: 9090)
- `--external-interface`: External interface for TC attachment (optional)
- `--db-path`: SQLite database path for persistent storage

### Enabling Drift in Volant

Set the `VOLANT_DRIFT_ENDPOINT` environment variable when starting Volant:

```bash
VOLANT_DRIFT_ENDPOINT=http://localhost:9090 volantd
```

Volant will automatically register routes with Drift when creating VMs with port mappings.

### Example: Full Stack with Drift

```bash
# Terminal 1: Start Drift
driftd --interface vbr0

# Terminal 2: Start Volant with Drift enabled
VOLANT_DRIFT_ENDPOINT=http://localhost:9090 volantd

# Terminal 3: Create VMs
volar vms create web --image nginx --port 8080:80
volar vms create api --image myapp --port 8081:8080
```

## Automatic Fallback

Drift is designed to be **optional and fault-tolerant**. If Drift becomes unavailable, Volant automatically falls back to vsock proxy:

1. **Startup**: If Drift is not running when Volant starts, vsock proxy is used
2. **Mid-flight**: If Drift crashes or stops responding, new VMs use vsock proxy
3. **Existing VMs**: Continue running (though connections may be disrupted)

This design ensures high availability - your VMs continue to work even if Drift fails.

## Monitoring

### Metrics Endpoint

Drift exposes a `/metrics` endpoint with real-time statistics:

```bash
curl http://localhost:9090/metrics
```

Response:
```json
{
  "ingress_packets": 125847,
  "ingress_bytes": 87654321,
  "egress_packets": 125840,
  "egress_bytes": 95123456
}
```

### Health Check

Check if Drift is healthy:

```bash
curl http://localhost:9090/healthz
```

Response: `ok` (HTTP 200) if healthy

### Route Inspection

List all active routes:

```bash
curl http://localhost:9090/routes
```

Response:
```json
[
  {
    "host_port": 8080,
    "protocol": "tcp",
    "backend": {
      "type": "bridge",
      "ip": "192.168.127.3",
      "port": 80
    }
  }
]
```

## Performance Benchmarking

Use the provided benchmark script to measure performance gains:

```bash
./scripts/bench-drift.sh
```

This script:
1. Starts Drift and creates a test VM
2. Benchmarks Drift routing performance
3. Switches to vsock proxy
4. Benchmarks vsock proxy performance
5. Compares results and calculates speedup

Expected output:
```
=== Results ===

Drift L4 (eBPF):
  Requests/sec: 12543.21
  Latency (ms): 7.97

Vsock Proxy (userspace):
  Requests/sec: 4982.14
  Latency (ms): 20.07

Performance Improvement: 2.52x
✓ SUCCESS: Drift achieves 2x+ speedup
```

## Troubleshooting

### Drift fails to start

**Symptom**: `driftd` exits with "failed to load eBPF objects"

**Causes**:
- eBPF not supported on kernel (requires Linux 5.10+)
- Missing kernel BTF information
- Insufficient permissions (requires CAP_BPF or root)

**Solution**:
```bash
# Check kernel version
uname -r

# Run with sudo if needed
sudo driftd --interface vbr0

# On macOS: Drift is not supported (Linux-only)
```

### Routes not being created

**Symptom**: VMs start but don't use Drift routing

**Causes**:
- Drift endpoint not configured
- Drift daemon not running
- Network interface mismatch

**Solution**:
```bash
# Verify Drift is running
curl http://localhost:9090/healthz

# Check Volant configuration
env | grep DRIFT

# Verify routes exist
curl http://localhost:9090/routes
```

### Performance not improved

**Symptom**: Benchmarks show minimal speedup

**Causes**:
- CPU bottleneck elsewhere (not networking)
- Test workload too light
- Bridge configuration issues

**Solution**:
```bash
# Run comprehensive benchmark
./scripts/bench-drift.sh

# Check Drift is actually being used
curl http://localhost:9090/metrics
# Should show non-zero packet counts

# Verify eBPF programs attached
sudo bpftool prog list | grep drift
```

## Technical Details

### Packet Flow with Drift

1. **Ingress Path**:
   ```
   External Client → NIC → TC Ingress Hook → eBPF Program
     ↓
   Packet rewrite: dst_ip:dst_port → VM_IP:VM_PORT
     ↓
   Bridge → VM
   ```

2. **Egress Path**:
   ```
   VM → Bridge → TC Egress Hook → eBPF Program
     ↓
   Packet rewrite: src_ip:src_port → Host_IP:Host_PORT
     ↓
   NIC → External Client
   ```

### Connection Tracking

Drift maintains a connection tracking table (LRU hash map) with entries:
- Client IP:Port
- VM IP:Port
- Original destination (for reverse NAT)
- Last seen timestamp

This enables bidirectional packet rewriting for stateful protocols like TCP.

### Checksum Recalculation

After rewriting IP addresses and ports, Drift recalculates checksums:
- IP header checksum (L3)
- TCP/UDP checksum (L4)
- Uses BPF helper functions for efficient computation

## Limitations

1. **Linux-only**: eBPF is not available on macOS/Windows
2. **IPv4 only**: IPv6 not currently supported
3. **TCP/UDP only**: Other protocols fall back to vsock
4. **Kernel requirement**: Linux 5.10+ with BTF support
5. **Root/CAP_BPF**: Requires elevated privileges to load eBPF programs

## Best Practices

1. **Always enable Drift in production** for maximum performance
2. **Monitor metrics** regularly to detect issues
3. **Test failover** to ensure vsock proxy fallback works
4. **Benchmark your workload** to measure actual gains
5. **Keep kernel updated** for latest eBPF features and fixes

## Related Documentation

- [Networking Guide](1_networking.md) - General Volant networking concepts
- [Architecture: Networking](../5_architecture/5_networking.md) - Deep dive into network stack
- [Troubleshooting](5_troubleshooting.md) - General troubleshooting guide

## Further Reading

- [eBPF Documentation](https://ebpf.io/what-is-ebpf/)
- [TC (Traffic Control) in Linux](https://tldp.org/HOWTO/Traffic-Control-HOWTO/)
- [Cilium eBPF Library](https://github.com/cilium/ebpf)
