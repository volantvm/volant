#!/bin/bash
set -e

echo "=== Drift Performance Benchmark ==="
echo ""

# Check if required tools are installed
if ! command -v ab &> /dev/null; then
    echo "Error: Apache Bench (ab) not found. Install with: brew install httpd (macOS) or apt-get install apache2-utils (Linux)"
    exit 1
fi

# Configuration
REQUESTS=10000
CONCURRENCY=100
DRIFT_PORT=9090
TEST_VM_NAME="bench-nginx"
TEST_VM_PORT=8080

echo "Configuration:"
echo "  Requests: $REQUESTS"
echo "  Concurrency: $CONCURRENCY"
echo "  Test VM: $TEST_VM_NAME"
echo "  Host Port: $TEST_VM_PORT"
echo ""

# 1. Start Drift daemon
echo "[1/6] Starting Drift daemon..."
if pgrep -x "driftd" > /dev/null; then
    echo "  Drift daemon already running"
else
    driftd --interface vbr0 --port $DRIFT_PORT &
    DRIFT_PID=$!
    echo "  Drift PID: $DRIFT_PID"
    sleep 2
fi

# 2. Start Volantd with Drift enabled
echo "[2/6] Starting Volantd with Drift..."
if pgrep -x "volantd" > /dev/null; then
    echo "  Volantd already running"
else
    VOLANT_DRIFT_ENDPOINT=http://localhost:$DRIFT_PORT volantd &
    VOLANTD_PID=$!
    echo "  Volantd PID: $VOLANTD_PID"
    sleep 2
fi

# 3. Create test VM with Drift routing
echo "[3/6] Creating test VM..."
volar vms create $TEST_VM_NAME --image nginx --port $TEST_VM_PORT:80 || true
sleep 3

# 4. Benchmark Drift routing
echo "[4/6] Benchmarking Drift L4 routing..."
ab -n $REQUESTS -c $CONCURRENCY http://localhost:$TEST_VM_PORT/ > /tmp/drift-results.txt 2>&1 || true

# 5. Stop Drift and restart VM to use vsock proxy
echo "[5/6] Stopping Drift, switching to vsock proxy..."
if [ -n "$DRIFT_PID" ]; then
    kill $DRIFT_PID 2>/dev/null || true
fi
pkill -9 driftd 2>/dev/null || true
sleep 2

# Restart VM to pick up vsock proxy
volar vms stop $TEST_VM_NAME || true
sleep 1
volar vms start $TEST_VM_NAME || true
sleep 3

# 6. Benchmark vsock proxy
echo "[6/6] Benchmarking vsock proxy..."
ab -n $REQUESTS -c $CONCURRENCY http://localhost:$TEST_VM_PORT/ > /tmp/vsock-results.txt 2>&1 || true

# Parse and compare results
echo ""
echo "=== Results ==="
echo ""

if [ -f /tmp/drift-results.txt ]; then
    DRIFT_RPS=$(grep "Requests per second" /tmp/drift-results.txt | awk '{print $4}' || echo "0")
    DRIFT_LATENCY=$(grep "Time per request" /tmp/drift-results.txt | head -1 | awk '{print $4}' || echo "0")
    echo "Drift L4 (eBPF):"
    echo "  Requests/sec: $DRIFT_RPS"
    echo "  Latency (ms): $DRIFT_LATENCY"
else
    echo "Drift benchmark failed"
    DRIFT_RPS=0
    DRIFT_LATENCY=0
fi

echo ""

if [ -f /tmp/vsock-results.txt ]; then
    VSOCK_RPS=$(grep "Requests per second" /tmp/vsock-results.txt | awk '{print $4}' || echo "0")
    VSOCK_LATENCY=$(grep "Time per request" /tmp/vsock-results.txt | head -1 | awk '{print $4}' || echo "0")
    echo "Vsock Proxy (userspace):"
    echo "  Requests/sec: $VSOCK_RPS"
    echo "  Latency (ms): $VSOCK_LATENCY"
else
    echo "Vsock benchmark failed"
    VSOCK_RPS=0
    VSOCK_LATENCY=0
fi

echo ""

# Calculate speedup
if [ "$DRIFT_RPS" != "0" ] && [ "$VSOCK_RPS" != "0" ]; then
    SPEEDUP=$(echo "scale=2; $DRIFT_RPS / $VSOCK_RPS" | bc)
    echo "Performance Improvement: ${SPEEDUP}x"

    if (( $(echo "$SPEEDUP >= 2.0" | bc -l) )); then
        echo "✓ SUCCESS: Drift achieves 2x+ speedup"
    else
        echo "⚠ WARNING: Speedup below 2x target"
    fi
fi

# Cleanup
echo ""
echo "Cleaning up..."
volar vms destroy $TEST_VM_NAME 2>/dev/null || true
if [ -n "$VOLANTD_PID" ]; then
    kill $VOLANTD_PID 2>/dev/null || true
fi
pkill -9 volantd 2>/dev/null || true

echo ""
echo "=== Benchmark Complete ==="
echo "Full results saved to:"
echo "  - /tmp/drift-results.txt"
echo "  - /tmp/vsock-results.txt"
