#!/bin/bash
# Build the MkDocs website (lean-forward experience)
# Produces interactive, searchable documentation site

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PUBLISH_DIR="$(dirname "$SCRIPT_DIR")"
WEBSITE_DIR="$PUBLISH_DIR/website"
DIST_DIR="$PUBLISH_DIR/dist/website"

echo "=========================================="
echo "Building Gorai Website (MkDocs)"
echo "=========================================="
echo ""

# Parse arguments
CLEAN=false
SERVE=false
while [[ $# -gt 0 ]]; do
    case $1 in
        --clean)
            CLEAN=true
            shift
            ;;
        --serve)
            SERVE=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--clean] [--serve]"
            exit 1
            ;;
    esac
done

# Clean if requested
if [ "$CLEAN" = true ]; then
    echo "Cleaning previous build..."
    rm -rf "$DIST_DIR"
fi

# Setup symlinks first
echo "Setting up symlinks..."
"$SCRIPT_DIR/setup-website-links.sh"
echo ""

# Check for mkdocs
if ! command -v mkdocs &> /dev/null; then
    echo "ERROR: mkdocs is not installed"
    echo "Install with: pip install mkdocs-material"
    exit 1
fi

# Build the website
echo "Building website..."
cd "$WEBSITE_DIR"

if [ "$SERVE" = true ]; then
    echo "Starting development server..."
    mkdocs serve --dev-addr 0.0.0.0:8001
else
    mkdocs build --site-dir "$DIST_DIR"

    echo ""
    echo "=========================================="
    echo "Website build complete!"
    echo "=========================================="
    echo ""
    echo "Output: $DIST_DIR"
    echo ""
    echo "Files:"
    ls -la "$DIST_DIR" 2>/dev/null | head -20 || echo "  (build directory not found)"
    echo ""
    echo "To view locally:"
    echo "  cd $DIST_DIR && python3 -m http.server 8001"
    echo ""
    echo "Or use: $0 --serve"
fi
