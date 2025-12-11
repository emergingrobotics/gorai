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
	$(MAKE) -C book all

book-pdf:
	$(MAKE) -C book pdf

book-epub:
	$(MAKE) -C book epub

serve-book:
	$(MAKE) -C book html
	@echo "Starting preview server at http://localhost:8000"
	python3 -m http.server 8000 --directory book/dist

# Website targets
website:
	$(MAKE) -C website build

serve-website:
	$(MAKE) -C website serve

# Build website with book downloads
website-with-book: book
	$(MAKE) -C website downloads
	$(MAKE) -C website build

# Clean all
clean:
	$(MAKE) -C book clean
	$(MAKE) -C website clean
	@echo "All build artifacts cleaned"

# Validate content structure
validate:
	@echo "Validating content structure..."
	@test -f book/metadata.yaml || (echo "ERROR: book/metadata.yaml missing" && exit 1)
	@test -d book/chapters || (echo "ERROR: book/chapters missing" && exit 1)
	@test -f website/hugo.toml || (echo "ERROR: website/hugo.toml missing" && exit 1)
	@test -d website/content || (echo "ERROR: website/content missing" && exit 1)
	@test -d images || (echo "ERROR: images directory missing" && exit 1)
	@echo "Validation passed!"
