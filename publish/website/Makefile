.PHONY: all build serve clean help downloads

# This Makefile is for standalone use outside the container.
# For container-based builds, use: make -C .. website
#
# Requirements:
#   - hugo (extended version 0.146.0+)

# Default target
all: build

# Help
help:
	@echo "Gorai Website Build System (Hugo)"
	@echo ""
	@echo "NOTE: For container-based builds, use: make -C .. website"
	@echo ""
	@echo "Targets:"
	@echo "  all       - Build the website (default)"
	@echo "  build     - Build the website"
	@echo "  serve     - Start development server with live reload"
	@echo "  downloads - Copy book files to static/downloads"
	@echo "  clean     - Remove build artifacts"
	@echo ""
	@echo "Requirements:"
	@echo "  - hugo extended version 0.146.0+"

# Build website
build:
	@echo "Building website..."
	hugo --minify
	@echo "Website built in public/"

# Development server
serve:
	hugo server --bind 0.0.0.0 --buildDrafts --buildFuture

# Copy book downloads from book build
downloads:
	@mkdir -p static/downloads
	@if [ -f ../book/dist/gorai-book.pdf ]; then \
		cp ../book/dist/gorai-book.pdf static/downloads/; \
		echo "Copied PDF to static/downloads/"; \
	elif [ -f ../dist/book/gorai-book.pdf ]; then \
		cp ../dist/book/gorai-book.pdf static/downloads/; \
		echo "Copied PDF to static/downloads/"; \
	else \
		echo "Warning: PDF not found"; \
	fi
	@if [ -f ../book/dist/gorai-book.epub ]; then \
		cp ../book/dist/gorai-book.epub static/downloads/; \
		echo "Copied ePub to static/downloads/"; \
	elif [ -f ../dist/book/gorai-book.epub ]; then \
		cp ../dist/book/gorai-book.epub static/downloads/; \
		echo "Copied ePub to static/downloads/"; \
	else \
		echo "Warning: ePub not found"; \
	fi

# Clean build artifacts
clean:
	rm -rf public/
	rm -rf resources/_gen/
	@echo "Build artifacts cleaned"
