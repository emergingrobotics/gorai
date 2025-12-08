#!/bin/bash
# Build the mdBook (lean-back experience)
# Produces HTML, PDF-ready, and optionally ePub output

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PUBLISH_DIR="$(dirname "$SCRIPT_DIR")"
BOOK_DIR="$PUBLISH_DIR/book"
DIST_DIR="$PUBLISH_DIR/dist/book"

echo "=========================================="
echo "Building Gorai Book (mdBook)"
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
"$SCRIPT_DIR/setup-book-links.sh"
echo ""

# Check for mdbook
if ! command -v mdbook &> /dev/null; then
    echo "ERROR: mdbook is not installed"
    echo "Install with: cargo install mdbook"
    exit 1
fi

# Build the book
echo "Building book..."
cd "$BOOK_DIR"

if [ "$SERVE" = true ]; then
    echo "Starting development server..."
    mdbook serve --open
else
    mdbook build

    echo ""
    echo "=========================================="
    echo "Book build complete!"
    echo "=========================================="
    echo ""
    echo "Output: $DIST_DIR"
    echo ""
    echo "Files:"
    ls -la "$DIST_DIR" 2>/dev/null || echo "  (build directory not found)"
    echo ""
    echo "To view locally:"
    echo "  cd $DIST_DIR && python3 -m http.server 8000"
    echo ""
    echo "Or use: $0 --serve"
fi
