#!/bin/bash
# Setup symbolic links from website/docs/book to content/
#
# NOTE: This script is DEPRECATED. The current publishing system uses
# Hugo directly without needing symlinks from a separate content directory.
# This script is kept for backward compatibility but now exits gracefully.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PUBLISH_DIR="$(dirname "$SCRIPT_DIR")"
CONTENT="$PUBLISH_DIR/content"

# Check if using the new structure (Hugo website with content in place)
if [ -d "$PUBLISH_DIR/website/content" ]; then
    echo "Using new website structure (website/content/) - no symlinks needed."
    exit 0
fi

# Legacy behavior for old content/ structure
echo "Setting up website symbolic links..."
echo "  Content: $CONTENT"

if [ ! -d "$CONTENT" ]; then
    echo "NOTE: Content directory not found at $CONTENT"
    echo "This is expected with the new website structure."
    echo "Website content is now directly in publish/website/content/"
    exit 0
fi

# Legacy symlink setup (kept for reference but should not be reached)
DOCS_DIR="$PUBLISH_DIR/website/docs"
echo "  Docs: $DOCS_DIR"

# Create book directory in docs if it doesn't exist
BOOK_DOCS="$DOCS_DIR/book"
mkdir -p "$BOOK_DOCS"

# Clear existing symlinks in book directory
find "$BOOK_DOCS" -type l -delete 2>/dev/null || true

# Link book index
if [ -f "$CONTENT/introduction.md" ]; then
    echo "  Note: Book index.md will be created separately"
fi

# Link part directories for the book section
for part in part1-getting-started part2-core-framework part3-development part4-advanced; do
    if [ -d "$CONTENT/$part" ]; then
        ln -sf "../../../content/$part" "$BOOK_DOCS/$part"
        echo "  Linked: book/$part/"
    else
        echo "  WARNING: $part not found"
    fi
done

# Link appendices for the book section
if [ -d "$CONTENT/appendices" ]; then
    ln -sf "../../../content/appendices" "$BOOK_DOCS/appendices"
    echo "  Linked: book/appendices/"
fi

# Link reference directory
if [ -d "$CONTENT/reference" ]; then
    ln -sf "../../../content/reference" "$BOOK_DOCS/reference"
    echo "  Linked: book/reference/"

    mkdir -p "$DOCS_DIR/reference"
    for reffile in "$CONTENT/reference"/*.md; do
        if [ -f "$reffile" ]; then
            filename=$(basename "$reffile")
            if [ "$filename" != "_index.md" ] && [ "$filename" != "index.md" ]; then
                ln -sf "../../../content/reference/$filename" "$DOCS_DIR/reference/$filename" 2>/dev/null || true
            fi
        fi
    done
fi

# Link examples directory
if [ -d "$CONTENT/examples" ]; then
    mkdir -p "$DOCS_DIR/examples"
    find "$DOCS_DIR/examples" -maxdepth 1 -type l -delete 2>/dev/null || true

    for entry in "$CONTENT/examples"/*; do
        if [ -d "$entry" ] && [ ! -L "$entry" ]; then
            dirname=$(basename "$entry")
            ln -sf "../../../content/examples/$dirname" "$DOCS_DIR/examples/$dirname"
            echo "  Linked: examples/$dirname/"
        fi
    done
fi

echo ""
echo "Website symbolic links created successfully!"
