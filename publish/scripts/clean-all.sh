#!/bin/bash
# Deep clean all build artifacts and caches
# Usage: ./scripts/clean-all.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$ROOT_DIR"

echo "=== Deep Clean Gorai Publishing ==="
echo ""

# Standard clean
echo "Cleaning build artifacts..."
make clean 2>/dev/null || true

# Additional cleanup
echo "Removing additional caches..."

# Hugo caches
rm -rf website/.hugo_build.lock 2>/dev/null || true
rm -rf website/resources/_gen 2>/dev/null || true

# Any temp files
find . -name "*.aux" -delete 2>/dev/null || true
find . -name "*.log" -delete 2>/dev/null || true
find . -name "*.toc" -delete 2>/dev/null || true
find . -name "*.out" -delete 2>/dev/null || true
find . -name "*~" -delete 2>/dev/null || true
find . -name ".DS_Store" -delete 2>/dev/null || true

echo ""
echo "=== Clean Complete ==="
