#!/bin/bash
# Deploy Gorai publishing artifacts
# Usage: ./scripts/deploy.sh [target]
#   target: local (default), staging, production

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
TARGET="${1:-local}"

cd "$ROOT_DIR"

echo "=== Gorai Publishing Deployment ==="
echo "Target: $TARGET"
echo ""

# Validate first
echo "Validating content structure..."
make validate

# Build everything
echo ""
echo "Building book..."
make book

echo ""
echo "Building website with book downloads..."
make website-with-book

echo ""
echo "=== Build Summary ==="
if [ -f "book/dist/gorai-book.pdf" ]; then
    PDF_SIZE=$(du -h book/dist/gorai-book.pdf | cut -f1)
    echo "PDF:     book/dist/gorai-book.pdf ($PDF_SIZE)"
fi

if [ -f "book/dist/gorai-book.epub" ]; then
    EPUB_SIZE=$(du -h book/dist/gorai-book.epub | cut -f1)
    echo "ePub:    book/dist/gorai-book.epub ($EPUB_SIZE)"
fi

if [ -d "website/public" ]; then
    PAGES=$(find website/public -name "*.html" | wc -l)
    echo "Website: website/public/ ($PAGES HTML pages)"
fi

echo ""

case "$TARGET" in
    local)
        echo "Local build complete. No deployment needed."
        echo ""
        echo "To preview:"
        echo "  Book:    make serve-book"
        echo "  Website: make serve-website"
        ;;
    staging)
        echo "Staging deployment not yet configured."
        echo "Add your staging deployment commands here."
        ;;
    production)
        echo "Production deployment not yet configured."
        echo "Add your production deployment commands here."
        ;;
    *)
        echo "Unknown target: $TARGET"
        echo "Valid targets: local, staging, production"
        exit 1
        ;;
esac

echo ""
echo "=== Deployment Complete ==="
