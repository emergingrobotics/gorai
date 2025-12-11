#!/bin/bash
# Build the Gorai book using Pandoc
#
# NOTE: This script is for LOCAL builds outside the container.
# For container-based builds, use: make book (from publish/ directory)
#
# The current build system uses Pandoc to generate PDF and ePub from
# markdown chapters in book/chapters/

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PUBLISH_DIR="$(dirname "$SCRIPT_DIR")"
BOOK_DIR="$PUBLISH_DIR/book"
DIST_DIR="$PUBLISH_DIR/dist/book"

echo "=========================================="
echo "Building Gorai Book (Pandoc)"
echo "=========================================="
echo ""

# Parse arguments
CLEAN=false
SERVE=false
FORMAT="all"  # pdf, epub, html, or all
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
        --pdf)
            FORMAT="pdf"
            shift
            ;;
        --epub)
            FORMAT="epub"
            shift
            ;;
        --html)
            FORMAT="html"
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--clean] [--serve] [--pdf|--epub|--html]"
            exit 1
            ;;
    esac
done

# Clean if requested
if [ "$CLEAN" = true ]; then
    echo "Cleaning previous build..."
    rm -rf "$DIST_DIR"
fi

# Check for pandoc
if ! command -v pandoc &> /dev/null; then
    echo "ERROR: pandoc is not installed"
    echo ""
    echo "Install pandoc:"
    echo "  sudo apt install pandoc"
    echo ""
    echo "Or use the container-based build:"
    echo "  cd $PUBLISH_DIR && make book"
    echo ""
    exit 1
fi

# Collect chapters
cd "$BOOK_DIR"
CHAPTERS=$(find chapters -name '*.md' | sort)

if [ -z "$CHAPTERS" ]; then
    echo "ERROR: No chapters found in $BOOK_DIR/chapters/"
    exit 1
fi

mkdir -p "$DIST_DIR"

echo "Found chapters:"
echo "$CHAPTERS" | while read -r ch; do echo "  $ch"; done
echo ""

# Build functions
build_pdf() {
    echo "Building PDF..."
    if ! command -v xelatex &> /dev/null; then
        echo "WARNING: xelatex not found, skipping PDF"
        echo "Install with: sudo apt install texlive-xetex"
        return 1
    fi
    pandoc \
        --metadata-file=metadata.yaml \
        --template=templates/pdf-template.tex \
        --pdf-engine=xelatex \
        --resource-path=.:images:../images:../images/logos \
        --toc \
        --number-sections \
        --highlight-style=tango \
        --top-level-division=chapter \
        -o "$DIST_DIR/gorai-book.pdf" \
        $CHAPTERS
    echo "  Created: $DIST_DIR/gorai-book.pdf"
}

build_epub() {
    echo "Building ePub..."
    pandoc \
        --metadata-file=metadata.yaml \
        --css=templates/epub.css \
        --toc \
        --number-sections \
        --top-level-division=chapter \
        -o "$DIST_DIR/gorai-book.epub" \
        $CHAPTERS
    echo "  Created: $DIST_DIR/gorai-book.epub"
}

build_html() {
    echo "Building HTML preview..."
    pandoc \
        --metadata-file=metadata.yaml \
        --css=templates/html.css \
        --standalone \
        --toc \
        --number-sections \
        --top-level-division=chapter \
        -o "$DIST_DIR/gorai-book.html" \
        $CHAPTERS
    echo "  Created: $DIST_DIR/gorai-book.html"
}

# Build requested format(s)
case "$FORMAT" in
    pdf)
        build_pdf
        ;;
    epub)
        build_epub
        ;;
    html)
        build_html
        ;;
    all)
        build_pdf || true
        build_epub
        ;;
esac

if [ "$SERVE" = true ]; then
    echo ""
    echo "Starting preview server on http://localhost:8000"
    cd "$DIST_DIR"
    python3 -m http.server 8000
else
    echo ""
    echo "=========================================="
    echo "Book build complete!"
    echo "=========================================="
    echo ""
    echo "Output: $DIST_DIR"
    echo ""
    echo "Files:"
    ls -la "$DIST_DIR" 2>/dev/null || echo "  (no files)"
    echo ""
    echo "To preview:"
    echo "  $0 --serve"
fi
