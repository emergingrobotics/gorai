#!/bin/bash
set -e

# Setup symlinks before any build
setup_links() {
    echo "Setting up symlinks..."
    /workspace/publish/scripts/setup-book-links.sh
    /workspace/publish/scripts/setup-website-links.sh
}

case "$1" in
    setup)
        # Just setup symlinks
        setup_links
        echo "Symlinks setup complete."
        ;;

    book)
        # Build the book with mdBook
        setup_links
        cd /workspace/publish/book
        mdbook-mermaid install .
        mdbook build --dest-dir /workspace/publish/dist/book
        echo "Book built to publish/dist/book/"
        ;;

    book-serve)
        # Serve book with live reload
        setup_links
        cd /workspace/publish/book
        mdbook-mermaid install .
        mdbook serve --hostname 0.0.0.0 --port 3000
        ;;

    website)
        # Build website with MkDocs
        setup_links
        cd /workspace/publish/website
        mkdocs build --site-dir /workspace/publish/dist/website
        echo "Website built to publish/dist/website/"
        ;;

    website-serve)
        # Serve website with live reload
        setup_links
        cd /workspace/publish/website
        mkdocs serve --dev-addr 0.0.0.0:8000
        ;;

    api)
        # Build API reference with pkgsite
        cd /workspace
        # Generate static pages for local packages
        pkgsite -http=:6060 -list=false . &
        sleep 5
        # TODO: Static export when pkgsite supports it
        # For now, serve dynamically
        echo "API server running on port 6060"
        wait
        ;;

    api-serve)
        # Serve API reference dynamically
        cd /workspace
        pkgsite -http=0.0.0.0:6060 .
        ;;

    all)
        # Build everything
        $0 book
        $0 website
        echo "All documentation built to publish/dist/"
        ;;

    help|*)
        echo "Gorai Publishing Container"
        echo ""
        echo "Commands:"
        echo "  book          Build the mdBook book"
        echo "  book-serve    Serve book with live reload (port 3000)"
        echo "  website       Build the MkDocs website"
        echo "  website-serve Serve website with live reload (port 8000)"
        echo "  api-serve     Serve API reference (port 6060)"
        echo "  all           Build book and website"
        echo "  help          Show this message"
        ;;
esac
