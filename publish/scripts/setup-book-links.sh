#!/bin/bash
# Setup symbolic links from book/src to content/
# This allows mdBook to consume content from the canonical location

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PUBLISH_DIR="$(dirname "$SCRIPT_DIR")"
BOOK_SRC="$PUBLISH_DIR/book/src"
CONTENT="$PUBLISH_DIR/content"

echo "Setting up book symbolic links..."
echo "  Content: $CONTENT"
echo "  Book src: $BOOK_SRC"

# Verify content directory exists
if [ ! -d "$CONTENT" ]; then
    echo "ERROR: Content directory not found: $CONTENT"
    exit 1
fi

# Clear existing symlinks (but preserve README.md and SUMMARY.md)
find "$BOOK_SRC" -type l -delete 2>/dev/null || true

# Link introduction
if [ -f "$CONTENT/introduction.md" ]; then
    ln -sf "../../content/introduction.md" "$BOOK_SRC/introduction.md"
    echo "  Linked: introduction.md"
fi

# Link part directories
for part in part1-getting-started part2-core-framework part3-development part4-advanced; do
    if [ -d "$CONTENT/$part" ]; then
        ln -sf "../../content/$part" "$BOOK_SRC/$part"
        echo "  Linked: $part/"
    else
        echo "  WARNING: $part not found"
    fi
done

# Link appendices
if [ -d "$CONTENT/appendices" ]; then
    ln -sf "../../content/appendices" "$BOOK_SRC/appendices"
    echo "  Linked: appendices/"
fi

# Link reference
if [ -d "$CONTENT/reference" ]; then
    ln -sf "../../content/reference" "$BOOK_SRC/reference"
    echo "  Linked: reference/"
fi

# Link examples
if [ -d "$CONTENT/examples" ]; then
    ln -sf "../../content/examples" "$BOOK_SRC/examples"
    echo "  Linked: examples/"
fi

echo ""
echo "Book symbolic links created successfully!"
echo ""
echo "Book structure:"
ls -la "$BOOK_SRC/"
