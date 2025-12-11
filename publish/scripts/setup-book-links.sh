#!/bin/bash
# Setup symbolic links from book/src to content/
#
# NOTE: This script is DEPRECATED. The current publishing system uses
# Pandoc directly on book/chapters/ without needing symlinks.
# This script is kept for backward compatibility but now exits gracefully.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PUBLISH_DIR="$(dirname "$SCRIPT_DIR")"
CONTENT="$PUBLISH_DIR/content"

# Check if using the new structure (chapters directly in book/)
if [ -d "$PUBLISH_DIR/book/chapters" ]; then
    echo "Using new book structure (book/chapters/) - no symlinks needed."
    exit 0
fi

# Legacy behavior for old content/ structure
echo "Setting up book symbolic links..."
echo "  Content: $CONTENT"

if [ ! -d "$CONTENT" ]; then
    echo "NOTE: Content directory not found at $CONTENT"
    echo "This is expected with the new book structure."
    echo "Book chapters are now directly in publish/book/chapters/"
    exit 0
fi

# Legacy symlink setup (kept for reference but should not be reached)
BOOK_SRC="$PUBLISH_DIR/book/src"
echo "  Book src: $BOOK_SRC"

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
