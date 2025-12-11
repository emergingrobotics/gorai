#!/bin/bash
# Build the Hugo website
#
# NOTE: This script is for LOCAL builds outside the container.
# For container-based builds, use: make website (from publish/ directory)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PUBLISH_DIR="$(dirname "$SCRIPT_DIR")"
WEBSITE_DIR="$PUBLISH_DIR/website"
DIST_DIR="$PUBLISH_DIR/dist/website"

echo "=========================================="
echo "Building Gorai Website (Hugo)"
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
    rm -rf "$WEBSITE_DIR/public"
    rm -rf "$WEBSITE_DIR/resources/_gen"
fi

# Check for hugo
if ! command -v hugo &> /dev/null; then
    echo "ERROR: hugo is not installed"
    echo ""
    echo "Install Hugo Extended:"
    echo ""
    echo "  # Option 1: Download from GitHub releases"
    echo "  # https://github.com/gohugoio/hugo/releases"
    echo ""
    echo "  # Option 2: Using snap"
    echo "  sudo snap install hugo"
    echo ""
    echo "Or use the container-based build:"
    echo "  cd $PUBLISH_DIR && make website"
    echo ""
    exit 1
fi

# Build the website
cd "$WEBSITE_DIR"

# Initialize/update Hugo modules if go.mod exists
if [ -f "go.mod" ]; then
    echo "Updating Hugo modules..."
    hugo mod get -u 2>/dev/null || true
fi

if [ "$SERVE" = true ]; then
    echo "Starting development server..."
    hugo server \
        --bind 0.0.0.0 \
        --port 1313 \
        --buildDrafts \
        --buildFuture \
        --disableFastRender
else
    echo "Building website..."
    hugo --minify --destination "$DIST_DIR"

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
    echo "  cd $DIST_DIR && python3 -m http.server 1313"
    echo ""
    echo "Or use: $0 --serve"
fi
