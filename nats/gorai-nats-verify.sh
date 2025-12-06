#!/bin/bash
#
# Gorai NATS Verification Script
# Tests connectivity, streams, and KV stores
#

set -e

NATS_URL="${NATS_URL:-nats://localhost:4222}"
ROBOT="${ROBOT:-test}"

echo "Verifying NATS configuration for Gorai"
echo "======================================="
echo "Server: $NATS_URL"
echo "Robot ID: $ROBOT"
echo ""

# -----------------------------------------------------------------------------
# Test 1: Server connectivity
# -----------------------------------------------------------------------------
echo "Test 1: Server connectivity..."
if nats server ping --server "$NATS_URL" > /dev/null 2>&1; then
    echo "  ✓ NATS server is reachable"
else
    echo "  ✗ Cannot reach NATS server at $NATS_URL"
    exit 1
fi

# -----------------------------------------------------------------------------
# Test 2: JetStream enabled
# -----------------------------------------------------------------------------
echo ""
echo "Test 2: JetStream enabled..."
if nats account info --server "$NATS_URL" 2>&1 | grep -q "JetStream"; then
    echo "  ✓ JetStream is enabled"
else
    echo "  ✗ JetStream is not enabled"
    exit 1
fi

# -----------------------------------------------------------------------------
# Test 3: Required streams exist
# -----------------------------------------------------------------------------
echo ""
echo "Test 3: Required streams..."
REQUIRED_STREAMS="GORAI_SENSORS GORAI_COMMANDS GORAI_STATE GORAI_ACTIONS GORAI_LOGS"
for stream in $REQUIRED_STREAMS; do
    if nats stream info "$stream" --server "$NATS_URL" > /dev/null 2>&1; then
        echo "  ✓ Stream $stream exists"
    else
        echo "  ✗ Stream $stream is missing (run gorai-nats-setup.sh)"
        exit 1
    fi
done

# -----------------------------------------------------------------------------
# Test 4: Required KV buckets exist
# -----------------------------------------------------------------------------
echo ""
echo "Test 4: Required KV buckets..."
REQUIRED_KV="GORAI_PARAMS GORAI_CONFIG"
for kv in $REQUIRED_KV; do
    if nats kv info "$kv" --server "$NATS_URL" > /dev/null 2>&1; then
        echo "  ✓ KV bucket $kv exists"
    else
        echo "  ✗ KV bucket $kv is missing (run gorai-nats-setup.sh)"
        exit 1
    fi
done

# -----------------------------------------------------------------------------
# Test 5: Pub/Sub round-trip
# -----------------------------------------------------------------------------
echo ""
echo "Test 5: Pub/Sub round-trip..."
TEST_SUBJECT="gorai.$ROBOT._test.verify"
TEST_MESSAGE="gorai-verify-$(date +%s)"

# Start subscriber in background
nats sub "$TEST_SUBJECT" --server "$NATS_URL" --count 1 > /tmp/nats-verify-sub.txt 2>&1 &
SUB_PID=$!
sleep 0.5

# Publish message
nats pub "$TEST_SUBJECT" "$TEST_MESSAGE" --server "$NATS_URL" > /dev/null 2>&1

# Wait for subscriber
sleep 0.5
kill $SUB_PID 2>/dev/null || true

if grep -q "$TEST_MESSAGE" /tmp/nats-verify-sub.txt 2>/dev/null; then
    echo "  ✓ Pub/Sub round-trip successful"
else
    echo "  ✗ Pub/Sub round-trip failed"
    exit 1
fi
rm -f /tmp/nats-verify-sub.txt

# -----------------------------------------------------------------------------
# Test 6: KV operations
# -----------------------------------------------------------------------------
echo ""
echo "Test 6: KV operations..."
KV_KEY="_test.verify"
KV_VALUE="verify-$(date +%s)"

# Put
nats kv put GORAI_PARAMS "$KV_KEY" "$KV_VALUE" --server "$NATS_URL" > /dev/null 2>&1

# Get
RETRIEVED=$(nats kv get GORAI_PARAMS "$KV_KEY" --server "$NATS_URL" --raw 2>/dev/null)

# Delete
nats kv del GORAI_PARAMS "$KV_KEY" --server "$NATS_URL" --force > /dev/null 2>&1

if [ "$RETRIEVED" = "$KV_VALUE" ]; then
    echo "  ✓ KV put/get/delete successful"
else
    echo "  ✗ KV operations failed"
    exit 1
fi

# -----------------------------------------------------------------------------
# Test 7: JetStream publish
# -----------------------------------------------------------------------------
echo ""
echo "Test 7: JetStream publish..."
JS_SUBJECT="gorai.$ROBOT._test.data"
JS_MESSAGE="jetstream-verify-$(date +%s)"

if nats pub "$JS_SUBJECT" "$JS_MESSAGE" --server "$NATS_URL" > /dev/null 2>&1; then
    echo "  ✓ JetStream publish successful"
else
    echo "  ✗ JetStream publish failed"
    exit 1
fi

# -----------------------------------------------------------------------------
# Summary
# -----------------------------------------------------------------------------
echo ""
echo "======================================="
echo "All verification tests passed!"
echo ""
echo "NATS is correctly configured for Gorai."
echo ""
echo "Quick reference:"
echo "  Subscribe to all: nats sub 'gorai.>'"
echo "  Robot topics:     nats sub 'gorai.$ROBOT.>'"
echo "  System topics:    nats sub 'gorai.$ROBOT._system.>'"
