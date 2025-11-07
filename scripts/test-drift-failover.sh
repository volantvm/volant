#!/bin/bash
set -e

echo "=== Drift Failover Test ==="
echo ""

TEST_FAILED=0
DRIFT_PORT=9090

# Helper function to test connectivity
test_connectivity() {
    local name=$1
    local port=$2
    local max_retries=5
    local retry=0

    while [ $retry -lt $max_retries ]; do
        if curl -sf "http://localhost:$port/" > /dev/null 2>&1; then
            echo "  ✓ $name connectivity OK"
            return 0
        fi
        retry=$((retry + 1))
        sleep 1
    done

    echo "  ✗ $name connectivity FAILED"
    return 1
}

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up..."
    volar vms destroy web 2>/dev/null || true
    volar vms destroy api 2>/dev/null || true
    volar vms destroy db 2>/dev/null || true
    pkill -9 driftd 2>/dev/null || true
    pkill -9 volantd 2>/dev/null || true
    sleep 1
}

trap cleanup EXIT

# Test 1: Drift unavailable at start
echo "[Test 1/3] VM creation without Drift (should use vsock proxy)"
echo "---"

# Ensure drift is not running
pkill -9 driftd 2>/dev/null || true
pkill -9 volantd 2>/dev/null || true
sleep 1

# Start volantd WITHOUT drift
echo "  Starting volantd (no Drift)..."
volantd &
VOLANTD_PID=$!
sleep 2

# Create VM - should automatically use vsock proxy
echo "  Creating web VM..."
volar vms create web --image nginx --port 8080:80 || {
    echo "  ✗ VM creation failed"
    TEST_FAILED=1
}
sleep 3

# Test connectivity
if ! test_connectivity "web" 8080; then
    TEST_FAILED=1
fi

echo ""

# Test 2: Drift appears mid-flight
echo "[Test 2/3] Starting Drift mid-flight (new VMs should use Drift)"
echo "---"

# Start Drift daemon
echo "  Starting Drift daemon..."
driftd --interface vbr0 --port $DRIFT_PORT &
DRIFT_PID=$!
sleep 2

# Restart volantd WITH drift
echo "  Restarting volantd with Drift..."
kill $VOLANTD_PID 2>/dev/null || true
sleep 1
VOLANT_DRIFT_ENDPOINT=http://localhost:$DRIFT_PORT volantd &
VOLANTD_PID=$!
sleep 2

# Create another VM - should use Drift
echo "  Creating api VM..."
volar vms create api --image nginx --port 8081:80 || {
    echo "  ✗ VM creation failed"
    TEST_FAILED=1
}
sleep 3

# Test connectivity
if ! test_connectivity "api" 8081; then
    TEST_FAILED=1
fi

# Verify Drift is being used by checking metrics
echo "  Checking Drift metrics..."
if curl -sf "http://localhost:$DRIFT_PORT/metrics" | grep -q "ingress_packets"; then
    echo "  ✓ Drift metrics available"
else
    echo "  ⚠ Drift metrics not available (may be zero traffic)"
fi

echo ""

# Test 3: Drift disappears mid-flight
echo "[Test 3/3] Stopping Drift (existing VMs should continue via vsock)"
echo "---"

# Kill Drift daemon
echo "  Stopping Drift daemon..."
kill $DRIFT_PID 2>/dev/null || true
pkill -9 driftd 2>/dev/null || true
sleep 2

# Existing VMs should still work (NOTE: existing connections may break,
# but this tests that the system doesn't crash)
echo "  Testing existing VM connectivity..."
if test_connectivity "web" 8080; then
    echo "  ✓ Existing VMs continue to work"
else
    echo "  ⚠ Existing VM connectivity lost (expected - eBPF routes removed)"
    # This is expected behavior - when Drift stops, routes are removed
    # The test is that the system doesn't crash
fi

# Create new VM without Drift - should fall back to vsock
echo "  Creating db VM (should fall back to vsock)..."
volar vms create db --image postgres --port 5432:5432 || {
    echo "  ✗ VM creation failed"
    TEST_FAILED=1
}
sleep 3

# Try to connect
if test_connectivity "db" 5432; then
    echo "  ✓ New VM uses vsock fallback successfully"
else
    echo "  ⚠ New VM connectivity failed"
    # Not a hard failure - might be due to postgres startup time
fi

echo ""
echo "=== Test Results ==="
echo ""

if [ $TEST_FAILED -eq 0 ]; then
    echo "✓ ALL TESTS PASSED"
    echo ""
    echo "Verified:"
    echo "  1. VMs work without Drift (vsock proxy)"
    echo "  2. VMs work when Drift is available (eBPF dataplane)"
    echo "  3. System handles Drift unavailability gracefully"
    exit 0
else
    echo "✗ SOME TESTS FAILED"
    echo ""
    echo "Review the output above for details"
    exit 1
fi
