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
    echo ""
    echo "Install mdbook using one of these methods:"
    echo ""
    echo "  # Option 1: Pre-built binary (fastest):"
    echo "  mkdir -p ~/.cargo/bin"
    echo "  curl -sSL https://github.com/rust-lang/mdBook/releases/download/v0.4.40/mdbook-v0.4.40-x86_64-unknown-linux-gnu.tar.gz | tar -xz -C ~/.cargo/bin"
    echo "  export PATH=\"\$HOME/.cargo/bin:\$PATH\""
    echo ""
    echo "  # Option 2: Using cargo (requires Rust):"
    echo "  cargo install mdbook"
    echo ""
    echo "  # Option 3: Using snap:"
    echo "  sudo snap install mdbook"
    echo ""
    exit 1
fi

# Check for mdbook-mermaid
if ! command -v mdbook-mermaid &> /dev/null; then
    echo "ERROR: mdbook-mermaid is not installed"
    echo ""
    echo "Install mdbook-mermaid for diagram support:"
    echo ""
    echo "  cargo install mdbook-mermaid"
    echo ""
    exit 1
fi

# Check for mdbook-toc
if ! command -v mdbook-toc &> /dev/null; then
    echo "ERROR: mdbook-toc is not installed"
    echo ""
    echo "Install mdbook-toc for table of contents support:"
    echo ""
    echo "  cargo install mdbook-toc"
    echo ""
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
