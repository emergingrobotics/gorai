#!/bin/bash
#
# Gorai NATS Setup Script
# Creates required streams, consumers, and KV buckets
#

set -e

NATS_URL="${NATS_URL:-nats://localhost:4222}"

echo "Configuring NATS for Gorai at $NATS_URL"
echo "=========================================="

# -----------------------------------------------------------------------------
# Stream: GORAI_SENSORS
# Purpose: Sensor data with limited retention
# -----------------------------------------------------------------------------
echo ""
echo "Creating GORAI_SENSORS stream..."

nats stream add GORAI_SENSORS \
    --server "$NATS_URL" \
    --subjects "gorai.*.*.data,gorai.*.*.data.*" \
    --retention limits \
    --max-msgs 100000 \
    --max-bytes 1GB \
    --max-age 1h \
    --max-msg-size 10MB \
    --storage file \
    --replicas 1 \
    --discard old \
    --dupe-window 2m \
    --no-deny-delete \
    --no-deny-purge \
    2>/dev/null || echo "Stream GORAI_SENSORS already exists or updated"

# -----------------------------------------------------------------------------
# Stream: GORAI_COMMANDS
# Purpose: Control commands with acknowledgment
# -----------------------------------------------------------------------------
echo ""
echo "Creating GORAI_COMMANDS stream..."

nats stream add GORAI_COMMANDS \
    --server "$NATS_URL" \
    --subjects "gorai.*.*.command,gorai.*.*.command.*" \
    --retention limits \
    --max-msgs 10000 \
    --max-bytes 100MB \
    --max-age 24h \
    --storage file \
    --replicas 1 \
    --discard old \
    2>/dev/null || echo "Stream GORAI_COMMANDS already exists or updated"

# -----------------------------------------------------------------------------
# Stream: GORAI_STATE
# Purpose: State updates with last-value retention
# -----------------------------------------------------------------------------
echo ""
echo "Creating GORAI_STATE stream..."

nats stream add GORAI_STATE \
    --server "$NATS_URL" \
    --subjects "gorai.*.*.state,gorai.*._system.*" \
    --retention limits \
    --max-msgs-per-subject 1 \
    --max-bytes 100MB \
    --storage file \
    --replicas 1 \
    --discard old \
    2>/dev/null || echo "Stream GORAI_STATE already exists or updated"

# -----------------------------------------------------------------------------
# Stream: GORAI_ACTIONS
# Purpose: Long-running action messages
# -----------------------------------------------------------------------------
echo ""
echo "Creating GORAI_ACTIONS stream..."

nats stream add GORAI_ACTIONS \
    --server "$NATS_URL" \
    --subjects "gorai.*.*.*.goal,gorai.*.*.*.feedback,gorai.*.*.*.result,gorai.*.*.*.cancel,gorai.*.*.*.status" \
    --retention limits \
    --max-msgs 50000 \
    --max-bytes 500MB \
    --max-age 1h \
    --storage file \
    --replicas 1 \
    --discard old \
    2>/dev/null || echo "Stream GORAI_ACTIONS already exists or updated"

# -----------------------------------------------------------------------------
# Stream: GORAI_LOGS
# Purpose: System logs with longer retention
# -----------------------------------------------------------------------------
echo ""
echo "Creating GORAI_LOGS stream..."

nats stream add GORAI_LOGS \
    --server "$NATS_URL" \
    --subjects "gorai.*._system.logs,gorai.*._system.diagnostics" \
    --retention limits \
    --max-msgs 1000000 \
    --max-bytes 5GB \
    --max-age 7d \
    --storage file \
    --replicas 1 \
    --discard old \
    2>/dev/null || echo "Stream GORAI_LOGS already exists or updated"

# -----------------------------------------------------------------------------
# KV Bucket: GORAI_PARAMS
# Purpose: Robot parameters (camera settings, control gains, etc.)
# -----------------------------------------------------------------------------
echo ""
echo "Creating GORAI_PARAMS KV bucket..."

nats kv add GORAI_PARAMS \
    --server "$NATS_URL" \
    --history 5 \
    --ttl 0 \
    --max-bytes 100MB \
    --storage file \
    --replicas 1 \
    2>/dev/null || echo "KV bucket GORAI_PARAMS already exists"

# -----------------------------------------------------------------------------
# KV Bucket: GORAI_CONFIG
# Purpose: Robot configuration storage
# -----------------------------------------------------------------------------
echo ""
echo "Creating GORAI_CONFIG KV bucket..."

nats kv add GORAI_CONFIG \
    --server "$NATS_URL" \
    --history 10 \
    --ttl 0 \
    --max-bytes 50MB \
    --storage file \
    --replicas 1 \
    2>/dev/null || echo "KV bucket GORAI_CONFIG already exists"

# -----------------------------------------------------------------------------
# Object Store: GORAI_MODELS
# Purpose: ML model storage
# -----------------------------------------------------------------------------
echo ""
echo "Creating GORAI_MODELS object store..."

nats object add GORAI_MODELS \
    --server "$NATS_URL" \
    --max-bytes 10GB \
    --storage file \
    --replicas 1 \
    2>/dev/null || echo "Object store GORAI_MODELS already exists"

# -----------------------------------------------------------------------------
# Object Store: GORAI_MAPS
# Purpose: SLAM map storage
# -----------------------------------------------------------------------------
echo ""
echo "Creating GORAI_MAPS object store..."

nats object add GORAI_MAPS \
    --server "$NATS_URL" \
    --max-bytes 5GB \
    --storage file \
    --replicas 1 \
    2>/dev/null || echo "Object store GORAI_MAPS already exists"

echo ""
echo "=========================================="
echo "Gorai NATS setup complete!"
echo ""
echo "Summary:"
nats stream ls --server "$NATS_URL"
echo ""
nats kv ls --server "$NATS_URL"
echo ""
nats object ls --server "$NATS_URL"
