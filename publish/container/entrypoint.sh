#!/bin/bash
set -e

# Gorai Publishing Container Entrypoint
# All builds happen inside the container using the tools installed here

WORKSPACE="/workspace"
PUBLISH_DIR="${WORKSPACE}/publish"
BOOK_DIR="${PUBLISH_DIR}/book"
WEBSITE_DIR="${PUBLISH_DIR}/website"
DIST_DIR="${PUBLISH_DIR}/dist"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Validate workspace structure
validate() {
    info "Validating workspace structure..."
    local errors=0

    if [ ! -d "${WORKSPACE}" ]; then
        error "Workspace not mounted at ${WORKSPACE}"
        errors=$((errors + 1))
    fi

    if [ ! -d "${PUBLISH_DIR}" ]; then
        error "publish directory not found"
        errors=$((errors + 1))
    fi

    if [ "$errors" -gt 0 ]; then
        error "Validation failed with $errors errors"
        exit 1
    fi

    info "Validation passed!"
}

# Build PDF book using Pandoc
build_book_pdf() {
    info "Building PDF book..."

    if [ ! -d "${BOOK_DIR}" ]; then
        error "Book directory not found at ${BOOK_DIR}"
        exit 1
    fi

    cd "${BOOK_DIR}"
    mkdir -p "${DIST_DIR}/book"

    # Collect chapters in order
    CHAPTERS=$(find chapters -name '*.md' | sort)

    if [ -z "$CHAPTERS" ]; then
        warn "No chapters found in ${BOOK_DIR}/chapters/"
        return
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
        -o "${DIST_DIR}/book/gorai-book.pdf" \
        ${CHAPTERS}

    info "PDF created: ${DIST_DIR}/book/gorai-book.pdf"
}

# Build ePub book using Pandoc
build_book_epub() {
    info "Building ePub book..."

    if [ ! -d "${BOOK_DIR}" ]; then
        error "Book directory not found at ${BOOK_DIR}"
        exit 1
    fi

    cd "${BOOK_DIR}"
    mkdir -p "${DIST_DIR}/book"

    # Collect chapters in order
    CHAPTERS=$(find chapters -name '*.md' | sort)

    if [ -z "$CHAPTERS" ]; then
        warn "No chapters found in ${BOOK_DIR}/chapters/"
        return
    fi

    pandoc \
        --metadata-file=metadata.yaml \
        --css=templates/epub.css \
        --toc \
        --number-sections \
        --top-level-division=chapter \
        -o "${DIST_DIR}/book/gorai-book.epub" \
        ${CHAPTERS}

    info "ePub created: ${DIST_DIR}/book/gorai-book.epub"
}

# Build HTML preview of book
build_book_html() {
    info "Building HTML book preview..."

    if [ ! -d "${BOOK_DIR}" ]; then
        error "Book directory not found at ${BOOK_DIR}"
        exit 1
    fi

    cd "${BOOK_DIR}"
    mkdir -p "${DIST_DIR}/book"

    # Collect chapters in order
    CHAPTERS=$(find chapters -name '*.md' | sort)

    if [ -z "$CHAPTERS" ]; then
        warn "No chapters found in ${BOOK_DIR}/chapters/"
        return
    fi

    pandoc \
        --metadata-file=metadata.yaml \
        --css=templates/html.css \
        --standalone \
        --toc \
        --number-sections \
        --top-level-division=chapter \
        -o "${DIST_DIR}/book/gorai-book.html" \
        ${CHAPTERS}

    info "HTML created: ${DIST_DIR}/book/gorai-book.html"
}

# Build all book formats
build_book() {
    build_book_pdf
    build_book_epub
}

# Build Hugo website
build_website() {
    info "Building Hugo website..."

    if [ ! -d "${WEBSITE_DIR}" ]; then
        error "Website directory not found at ${WEBSITE_DIR}"
        exit 1
    fi

    cd "${WEBSITE_DIR}"
    mkdir -p "${DIST_DIR}/website"

    # Initialize/update Hugo modules if go.mod exists
    if [ -f "go.mod" ]; then
        hugo mod get -u
    fi

    # Build with minification
    hugo --minify --destination "${DIST_DIR}/website"

    info "Website built: ${DIST_DIR}/website/"
}

# Serve Hugo website with live reload
serve_website() {
    info "Starting Hugo development server..."

    if [ ! -d "${WEBSITE_DIR}" ]; then
        error "Website directory not found at ${WEBSITE_DIR}"
        exit 1
    fi

    cd "${WEBSITE_DIR}"
    hugo server \
        --bind 0.0.0.0 \
        --port 1313 \
        --buildDrafts \
        --buildFuture \
        --disableFastRender
}

# Serve book HTML preview
serve_book() {
    build_book_html
    info "Starting book preview server on port 8000..."
    cd "${DIST_DIR}/book"
    python3 -m http.server 8000 --bind 0.0.0.0
}

# Serve Go API documentation
serve_api() {
    info "Starting pkgsite API server on port 6060..."
    cd "${WORKSPACE}"
    pkgsite -http=0.0.0.0:6060 .
}

# Copy book downloads to website static directory
copy_downloads() {
    info "Copying book downloads to website..."

    mkdir -p "${WEBSITE_DIR}/static/downloads"

    if [ -f "${DIST_DIR}/book/gorai-book.pdf" ]; then
        cp "${DIST_DIR}/book/gorai-book.pdf" "${WEBSITE_DIR}/static/downloads/"
        info "Copied PDF"
    else
        warn "PDF not found"
    fi

    if [ -f "${DIST_DIR}/book/gorai-book.epub" ]; then
        cp "${DIST_DIR}/book/gorai-book.epub" "${WEBSITE_DIR}/static/downloads/"
        info "Copied ePub"
    else
        warn "ePub not found"
    fi
}

# Build everything
build_all() {
    validate
    build_book
    copy_downloads
    build_website
    info "All documentation built to ${DIST_DIR}/"
}

# Clean build artifacts
clean() {
    info "Cleaning build artifacts..."
    rm -rf "${DIST_DIR}"
    rm -rf "${WEBSITE_DIR}/public"
    rm -rf "${WEBSITE_DIR}/resources/_gen"
    rm -rf "${BOOK_DIR}/dist"
    info "Clean complete"
}

# Show tool versions
versions() {
    echo "Tool versions:"
    echo "  Hugo:       $(hugo version 2>/dev/null | head -1 || echo 'not found')"
    echo "  Pandoc:     $(pandoc --version 2>/dev/null | head -1 || echo 'not found')"
    echo "  XeLaTeX:    $(xelatex --version 2>/dev/null | head -1 || echo 'not found')"
    echo "  ImageMagick: $(convert --version 2>/dev/null | head -1 || echo 'not found')"
    echo "  Go:         $(go version 2>/dev/null || echo 'not found')"
    echo "  Python:     $(python3 --version 2>/dev/null || echo 'not found')"
}

# Main command dispatcher
case "$1" in
    # Book commands
    book)
        validate
        build_book
        ;;
    book-pdf)
        validate
        build_book_pdf
        ;;
    book-epub)
        validate
        build_book_epub
        ;;
    book-html)
        validate
        build_book_html
        ;;
    book-serve|serve-book)
        validate
        serve_book
        ;;

    # Website commands
    website)
        validate
        build_website
        ;;
    website-serve|serve-website)
        validate
        serve_website
        ;;

    # Combined commands
    website-with-book)
        validate
        build_book
        copy_downloads
        build_website
        ;;
    all)
        build_all
        ;;

    # API documentation
    api-serve|serve-api)
        serve_api
        ;;

    # Utility commands
    validate)
        validate
        ;;
    clean)
        clean
        ;;
    versions)
        versions
        ;;
    shell)
        exec /bin/bash
        ;;

    help|*)
        echo "Gorai Publishing Container"
        echo ""
        echo "Book Commands:"
        echo "  book            Build PDF and ePub"
        echo "  book-pdf        Build PDF only"
        echo "  book-epub       Build ePub only"
        echo "  book-html       Build HTML preview"
        echo "  serve-book      Serve HTML preview (port 8000)"
        echo ""
        echo "Website Commands:"
        echo "  website         Build Hugo website"
        echo "  serve-website   Start Hugo dev server (port 1313)"
        echo ""
        echo "Combined Commands:"
        echo "  website-with-book  Build book, copy to website, build website"
        echo "  all                Build everything"
        echo ""
        echo "API Documentation:"
        echo "  serve-api       Start pkgsite server (port 6060)"
        echo ""
        echo "Utility Commands:"
        echo "  validate        Check workspace structure"
        echo "  clean           Remove build artifacts"
        echo "  versions        Show installed tool versions"
        echo "  shell           Start interactive shell"
        echo "  help            Show this message"
        echo ""
        echo "Output is written to publish/dist/"
        ;;
esac
