#!/bin/bash
# Build both book and website
#
# NOTE: This script is for LOCAL builds outside the container.
# For container-based builds, use: make all (from publish/ directory)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PUBLISH_DIR="$(dirname "$SCRIPT_DIR")"
DIST_DIR="$PUBLISH_DIR/dist"

echo "=========================================="
echo "Gorai Documentation Build"
echo "=========================================="
echo ""
echo "Building book (Pandoc) and website (Hugo)"
echo ""

# Parse arguments
CLEAN=false
BOOK_ONLY=false
WEBSITE_ONLY=false
while [[ $# -gt 0 ]]; do
    case $1 in
        --clean)
            CLEAN=true
            shift
            ;;
        --book-only)
            BOOK_ONLY=true
            shift
            ;;
        --website-only)
            WEBSITE_ONLY=true
            shift
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --clean         Remove previous builds first"
            echo "  --book-only     Only build the book"
            echo "  --website-only  Only build the website"
            echo "  --help          Show this help"
            echo ""
            echo "For container-based builds (recommended):"
            echo "  make -C $PUBLISH_DIR all"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Clean if requested
if [ "$CLEAN" = true ]; then
    echo "Cleaning all previous builds..."
    rm -rf "$DIST_DIR"
    mkdir -p "$DIST_DIR"
    echo ""
fi

# Build book
if [ "$WEBSITE_ONLY" = false ]; then
    echo "------------------------------------------"
    echo "Building Book (Pandoc)"
    echo "------------------------------------------"
    "$SCRIPT_DIR/build-book.sh" || {
        echo "ERROR: Book build failed"
        exit 1
    }
    echo ""
fi

# Build website
if [ "$BOOK_ONLY" = false ]; then
    echo "------------------------------------------"
    echo "Building Website (Hugo)"
    echo "------------------------------------------"
    "$SCRIPT_DIR/build-website.sh" || {
        echo "ERROR: Website build failed"
        exit 1
    }
    echo ""
fi

echo "=========================================="
echo "Build Complete!"
echo "=========================================="
echo ""
echo "Distribution directory: $DIST_DIR"
echo ""
if [ "$WEBSITE_ONLY" = false ]; then
    echo "Book:     $DIST_DIR/book/"
fi
if [ "$BOOK_ONLY" = false ]; then
    echo "Website:  $DIST_DIR/website/"
fi
echo ""
echo "To serve locally:"
if [ "$WEBSITE_ONLY" = false ]; then
    echo "  Book:    cd $DIST_DIR/book && python3 -m http.server 8000"
fi
if [ "$BOOK_ONLY" = false ]; then
    echo "  Website: cd $DIST_DIR/website && python3 -m http.server 1313"
fi
