# Gorai Documentation Build System
#
# This Makefile builds both the book (lean-back) and website (lean-forward)
# experiences from a single source of content.
#
# Usage:
#   make all          - Build both book and website
#   make book         - Build only the book
#   make website      - Build only the website
#   make serve-book   - Start book development server
#   make serve-web    - Start website development server
#   make clean        - Remove all build artifacts

.PHONY: all book website serve-book serve-web clean setup help

SCRIPTS_DIR := scripts
DIST_DIR := dist

help:
	@echo "Gorai Documentation Build System"
	@echo ""
	@echo "Targets:"
	@echo "  all          Build both book and website"
	@echo "  book         Build the book (mdBook)"
	@echo "  website      Build the website (MkDocs)"
	@echo "  serve-book   Start book development server (port 8000)"
	@echo "  serve-web    Start website development server (port 8001)"
	@echo "  setup        Setup symlinks only"
	@echo "  clean        Remove all build artifacts"
	@echo ""
	@echo "Prerequisites:"
	@echo "  - mdbook (cargo install mdbook)"
	@echo "  - mkdocs-material (pip install mkdocs-material)"

all:
	@$(SCRIPTS_DIR)/build-all.sh

book:
	@$(SCRIPTS_DIR)/build-book.sh

website:
	@$(SCRIPTS_DIR)/build-website.sh

serve-book:
	@$(SCRIPTS_DIR)/build-book.sh --serve

serve-web:
	@$(SCRIPTS_DIR)/build-website.sh --serve

setup:
	@$(SCRIPTS_DIR)/setup-book-links.sh
	@$(SCRIPTS_DIR)/setup-website-links.sh

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(DIST_DIR)
	@echo "Clean complete."
