#!/bin/bash
# Setup symbolic links from website/docs/book to content/
# This allows MkDocs to consume book content from the canonical location

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PUBLISH_DIR="$(dirname "$SCRIPT_DIR")"
DOCS_DIR="$PUBLISH_DIR/website/docs"
CONTENT="$PUBLISH_DIR/content"

echo "Setting up website symbolic links..."
echo "  Content: $CONTENT"
echo "  Docs: $DOCS_DIR"

# Verify content directory exists
if [ ! -d "$CONTENT" ]; then
    echo "ERROR: Content directory not found: $CONTENT"
    exit 1
fi

# Create book directory in docs if it doesn't exist
BOOK_DOCS="$DOCS_DIR/book"
mkdir -p "$BOOK_DOCS"

# Clear existing symlinks in book directory
find "$BOOK_DOCS" -type l -delete 2>/dev/null || true

# Link book index
if [ -f "$CONTENT/introduction.md" ]; then
    # We'll create a custom index.md for the website book section
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

# Link reference directory (also used outside book)
if [ -d "$CONTENT/reference" ]; then
    # Link to both locations - book/reference and docs/reference
    ln -sf "../../../content/reference" "$BOOK_DOCS/reference"
    echo "  Linked: book/reference/"

    # Also link individual reference files to top-level reference
    # (for backward compatibility with existing nav)
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
    # Link individual example directories to website examples
    # Only link real directories, not symlinks (to avoid recursion)
    mkdir -p "$DOCS_DIR/examples"
    for example in "$CONTENT/examples"/*/; do
        if [ -d "$example" ] && [ ! -L "${example%/}" ]; then
            dirname=$(basename "$example")
            ln -sf "../../../content/examples/$dirname" "$DOCS_DIR/examples/$dirname"
            echo "  Linked: examples/$dirname/"
        fi
    done
fi

echo ""
echo "Website symbolic links created successfully!"
echo ""
echo "Book structure:"
ls -la "$BOOK_DOCS/"
echo ""
echo "Reference structure:"
ls -la "$DOCS_DIR/reference/" 2>/dev/null || echo "  (no reference links)"
echo ""
echo "Examples structure:"
ls -la "$DOCS_DIR/examples/" 2>/dev/null || echo "  (no examples links)"
