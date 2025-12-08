#!/bin/bash
# Verify the publication structure is correct
# Run this before attempting builds

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PUBLISH_DIR="$(dirname "$SCRIPT_DIR")"

echo "=========================================="
echo "Verifying Publication Structure"
echo "=========================================="
echo ""

ERRORS=0

# Function to check file exists
check_file() {
    if [ -f "$1" ]; then
        echo "  ✓ $2"
    else
        echo "  ✗ MISSING: $2"
        ((ERRORS++))
    fi
}

# Function to check directory exists
check_dir() {
    if [ -d "$1" ]; then
        echo "  ✓ $2"
    else
        echo "  ✗ MISSING: $2"
        ((ERRORS++))
    fi
}

# Function to check symlink
check_link() {
    if [ -L "$1" ]; then
        echo "  ✓ $2 -> $(readlink "$1")"
    else
        echo "  ✗ NOT A LINK: $2"
        ((ERRORS++))
    fi
}

echo "Checking canonical content structure..."
check_dir "$PUBLISH_DIR/content" "content/"
check_file "$PUBLISH_DIR/content/introduction.md" "content/introduction.md"
check_dir "$PUBLISH_DIR/content/part1-getting-started" "content/part1-getting-started/"
check_dir "$PUBLISH_DIR/content/part2-core-framework" "content/part2-core-framework/"
check_dir "$PUBLISH_DIR/content/part3-development" "content/part3-development/"
check_dir "$PUBLISH_DIR/content/part4-advanced" "content/part4-advanced/"
check_dir "$PUBLISH_DIR/content/appendices" "content/appendices/"
check_dir "$PUBLISH_DIR/content/reference" "content/reference/"
check_dir "$PUBLISH_DIR/content/examples" "content/examples/"

echo ""
echo "Checking mdBook structure..."
check_file "$PUBLISH_DIR/book/book.toml" "book/book.toml"
check_file "$PUBLISH_DIR/book/src/SUMMARY.md" "book/src/SUMMARY.md"
check_file "$PUBLISH_DIR/book/src/README.md" "book/src/README.md"
check_link "$PUBLISH_DIR/book/src/part1-getting-started" "book/src/part1-getting-started"
check_link "$PUBLISH_DIR/book/src/part2-core-framework" "book/src/part2-core-framework"
check_link "$PUBLISH_DIR/book/src/part3-development" "book/src/part3-development"
check_link "$PUBLISH_DIR/book/src/part4-advanced" "book/src/part4-advanced"
check_link "$PUBLISH_DIR/book/src/appendices" "book/src/appendices"

echo ""
echo "Checking MkDocs structure..."
check_file "$PUBLISH_DIR/website/mkdocs.yml" "website/mkdocs.yml"
check_file "$PUBLISH_DIR/website/docs/index.md" "website/docs/index.md"
check_file "$PUBLISH_DIR/website/docs/book/index.md" "website/docs/book/index.md"
check_link "$PUBLISH_DIR/website/docs/book/part1-getting-started" "website/docs/book/part1-getting-started"
check_link "$PUBLISH_DIR/website/docs/book/part2-core-framework" "website/docs/book/part2-core-framework"
check_link "$PUBLISH_DIR/website/docs/book/part3-development" "website/docs/book/part3-development"
check_link "$PUBLISH_DIR/website/docs/book/part4-advanced" "website/docs/book/part4-advanced"

echo ""
echo "Checking chapter content..."
# Check some key chapter files exist
check_file "$PUBLISH_DIR/content/part1-getting-started/ch01-why-gorai/landscape.md" "ch01: landscape.md"
check_file "$PUBLISH_DIR/content/part2-core-framework/ch03-nats/whynats.md" "ch03: whynats.md"
check_file "$PUBLISH_DIR/content/part3-development/ch11-hello-sensor/overview.md" "ch11: overview.md"

echo ""
echo "Checking build scripts..."
check_file "$PUBLISH_DIR/scripts/build-all.sh" "scripts/build-all.sh"
check_file "$PUBLISH_DIR/scripts/build-book.sh" "scripts/build-book.sh"
check_file "$PUBLISH_DIR/scripts/build-website.sh" "scripts/build-website.sh"
check_file "$PUBLISH_DIR/scripts/setup-book-links.sh" "scripts/setup-book-links.sh"
check_file "$PUBLISH_DIR/scripts/setup-website-links.sh" "scripts/setup-website-links.sh"

echo ""
echo "=========================================="
if [ $ERRORS -eq 0 ]; then
    echo "All checks passed! ✓"
    echo "=========================================="
    echo ""
    echo "Build tools required:"
    echo "  - mdbook (cargo install mdbook)"
    echo "  - mkdocs-material (pip install mkdocs-material)"
    echo ""
    echo "To build:"
    echo "  make all       # Build both"
    echo "  make book      # Build book only"
    echo "  make website   # Build website only"
    exit 0
else
    echo "FAILED: $ERRORS errors found"
    echo "=========================================="
    exit 1
fi
