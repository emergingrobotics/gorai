.PHONY: all build serve clean help downloads

# Default target
all: build

# Help
help:
	@echo "Gorai Website Build System (Hugo)"
	@echo ""
	@echo "Targets:"
	@echo "  all       - Build the website (default)"
	@echo "  build     - Build the website"
	@echo "  serve     - Start development server with live reload"
	@echo "  downloads - Copy book files to static/downloads"
	@echo "  clean     - Remove build artifacts"
	@echo ""
	@echo "Requirements:"
	@echo "  - hugo (extended version recommended)"

# Build website
build:
	@echo "Building website..."
	hugo --minify
	@echo "Website built in public/"

# Development server
serve:
	hugo server --buildDrafts --buildFuture

# Copy book downloads from book build
downloads:
	@mkdir -p static/downloads
	@if [ -f ../newbook/dist/gorai-book.pdf ]; then \
		cp ../newbook/dist/gorai-book.pdf static/downloads/; \
		echo "Copied PDF to static/downloads/"; \
	else \
		echo "Warning: PDF not found at ../newbook/dist/gorai-book.pdf"; \
	fi
	@if [ -f ../newbook/dist/gorai-book.epub ]; then \
		cp ../newbook/dist/gorai-book.epub static/downloads/; \
		echo "Copied ePub to static/downloads/"; \
	else \
		echo "Warning: ePub not found at ../newbook/dist/gorai-book.epub"; \
	fi

# Clean build artifacts
clean:
	rm -rf public/
	rm -rf resources/_gen/
	@echo "Build artifacts cleaned"
