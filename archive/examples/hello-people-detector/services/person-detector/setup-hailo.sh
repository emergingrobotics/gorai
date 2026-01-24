#!/bin/bash
#
# Setup script to copy Hailo runtime files from host into container build context.
# Run this on the Raspberry Pi before building the container.
#
# Usage: ./setup-hailo.sh
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HAILO_RUNTIME_DIR="$SCRIPT_DIR/hailo_runtime"

echo "=== Hailo Runtime Setup ==="
echo ""

# Check prerequisites
echo "Checking prerequisites..."

if ! command -v hailortcli &> /dev/null; then
    echo "ERROR: hailortcli not found. Install with: sudo apt install hailo-all"
    exit 1
fi

if [ ! -e /dev/hailo0 ]; then
    echo "ERROR: /dev/hailo0 not found. Is Hailo NPU connected?"
    exit 1
fi

if [ ! -d /usr/lib/python3/dist-packages/hailo_platform ]; then
    echo "ERROR: Python hailo_platform not found. Install with: sudo apt install python3-hailort"
    exit 1
fi

if [ ! -f /usr/lib/libhailort.so ]; then
    echo "ERROR: libhailort.so not found. Install with: sudo apt install hailort"
    exit 1
fi

echo "  hailortcli: $(hailortcli --version | head -1)"
echo "  /dev/hailo0: OK"
echo "  hailo_platform: OK"
echo "  libhailort.so: OK"
echo ""

# Get HailoRT version
HAILORT_VERSION=$(hailortcli --version | grep -oP '\d+\.\d+\.\d+' | head -1)
echo "HailoRT version: $HAILORT_VERSION"
echo ""

# Create directories
echo "Creating hailo_runtime directory..."
rm -rf "$HAILO_RUNTIME_DIR"
mkdir -p "$HAILO_RUNTIME_DIR/lib"
mkdir -p "$HAILO_RUNTIME_DIR/python"

# Copy Python bindings
echo "Copying Python bindings..."
cp -r /usr/lib/python3/dist-packages/hailo_platform "$HAILO_RUNTIME_DIR/python/"
if [ -d /usr/lib/python3/dist-packages/hailort-*.egg-info ]; then
    cp -r /usr/lib/python3/dist-packages/hailort-*.egg-info "$HAILO_RUNTIME_DIR/python/" 2>/dev/null || true
fi

# Copy native library
echo "Copying native library..."
LIBHAILORT=$(readlink -f /usr/lib/libhailort.so)
LIBHAILORT_NAME=$(basename "$LIBHAILORT")
cp "$LIBHAILORT" "$HAILO_RUNTIME_DIR/lib/"
ln -sf "$LIBHAILORT_NAME" "$HAILO_RUNTIME_DIR/lib/libhailort.so"

# Verify
echo ""
echo "=== Verification ==="
echo "Library files:"
ls -la "$HAILO_RUNTIME_DIR/lib/"
echo ""
echo "Python package:"
ls -la "$HAILO_RUNTIME_DIR/python/hailo_platform/" | head -10
echo ""

# Check for native extension
NATIVE_EXT="$HAILO_RUNTIME_DIR/python/hailo_platform/pyhailort/_pyhailort.cpython-311-aarch64-linux-gnu.so"
if [ -f "$NATIVE_EXT" ]; then
    echo "Native extension: OK"
else
    echo "WARNING: Native extension not found. Container may not work."
    echo "Expected: $NATIVE_EXT"
fi

echo ""
echo "=== Model Setup ==="

# Check/copy model
MODEL_DIR="/opt/gorai/models"
MODEL_FILE="$MODEL_DIR/yolov8s_h8.hef"

if [ -f "$MODEL_FILE" ]; then
    echo "Model already exists: $MODEL_FILE"
else
    echo "Model not found at $MODEL_FILE"

    # Check for source model
    SRC_MODEL="/usr/share/hailo-models/yolov8s_h8.hef"
    if [ -f "$SRC_MODEL" ]; then
        echo "Copying from $SRC_MODEL..."
        sudo mkdir -p "$MODEL_DIR"
        sudo cp "$SRC_MODEL" "$MODEL_FILE"
        echo "Model copied to: $MODEL_FILE"
    else
        echo "WARNING: Source model not found at $SRC_MODEL"
        echo "Install with: sudo apt install hailo-tappas-core"
        echo "Or download from Hailo Model Zoo"
    fi
fi

echo ""
echo "=== Setup Complete ==="
echo ""
echo "Next steps:"
echo "  1. Build container: make build"
echo "  2. Run robot: make run"
echo "  3. Open dashboard: http://localhost:8080/models"
echo ""
