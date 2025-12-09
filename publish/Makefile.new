.PHONY: all book website clean serve-book serve-website help book-pdf book-epub website-with-book validate

# Default target
all: book website

# Help
help:
	@echo "Gorai Publishing Build System"
	@echo ""
	@echo "Targets:"
	@echo "  all              - Build book and website (default)"
	@echo "  book             - Build PDF and ePub"
	@echo "  book-pdf         - Build PDF only"
	@echo "  book-epub        - Build ePub only"
	@echo "  website          - Build Hugo website"
	@echo "  website-with-book - Build website and copy book downloads"
	@echo "  serve-book       - Preview book as HTML"
	@echo "  serve-website    - Start Hugo dev server"
	@echo "  validate         - Validate content structure"
	@echo "  clean            - Remove all build artifacts"
	@echo ""
	@echo "Requirements:"
	@echo "  Book:    pandoc, texlive-xetex"
	@echo "  Website: hugo"

# Book targets
book:
	$(MAKE) -C newbook all

book-pdf:
	$(MAKE) -C newbook pdf

book-epub:
	$(MAKE) -C newbook epub

serve-book:
	$(MAKE) -C newbook html
	@echo "Starting preview server at http://localhost:8000"
	python3 -m http.server 8000 --directory newbook/dist

# Website targets
website:
	$(MAKE) -C newsite build

serve-website:
	$(MAKE) -C newsite serve

# Build website with book downloads
website-with-book: book
	$(MAKE) -C newsite downloads
	$(MAKE) -C newsite build

# Clean all
clean:
	$(MAKE) -C newbook clean
	$(MAKE) -C newsite clean
	@echo "All build artifacts cleaned"

# Validate content structure
validate:
	@echo "Validating content structure..."
	@test -f newbook/metadata.yaml || (echo "ERROR: newbook/metadata.yaml missing" && exit 1)
	@test -d newbook/chapters || (echo "ERROR: newbook/chapters missing" && exit 1)
	@test -f newsite/hugo.toml || (echo "ERROR: newsite/hugo.toml missing" && exit 1)
	@test -d newsite/content || (echo "ERROR: newsite/content missing" && exit 1)
	@test -d images || (echo "ERROR: images directory missing" && exit 1)
	@echo "Validation passed!"
