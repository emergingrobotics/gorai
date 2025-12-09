# Gorai Documentation Build System
#
# This Makefile builds both the book (lean-back) and website (lean-forward)
# experiences from a single source of content.
#
# Usage (with container - recommended):
#   make container     - Build the publishing container
#   make docker-all    - Build both book and website using container
#   make docker-book   - Build only the book using container
#   make docker-web    - Build only the website using container
#
# Usage (local tools):
#   make all          - Build both book and website
#   make book         - Build only the book
#   make website      - Build only the website
#   make serve-book   - Start book development server
#   make serve-web    - Start website development server
#   make clean        - Remove all build artifacts

.PHONY: all book website serve-book serve-web clean setup help
.PHONY: container docker-all docker-book docker-web docker-serve-book docker-serve-web

SCRIPTS_DIR := scripts
DIST_DIR := dist
CONTAINER_NAME := gorai-publish
CONTAINER_CMD := podman

# Detect if podman is available, fallback to docker
ifeq ($(shell which podman 2>/dev/null),)
    CONTAINER_CMD := docker
endif

help:
	@echo "Gorai Documentation Build System"
	@echo ""
	@echo "Container targets (recommended - no local installs needed):"
	@echo "  container        Build the publishing container"
	@echo "  docker-all       Build both book and website"
	@echo "  docker-book      Build only the book"
	@echo "  docker-web       Build only the website"
	@echo "  docker-serve-book  Serve book with live reload (port 3000)"
	@echo "  docker-serve-web   Serve website with live reload (port 8000)"
	@echo ""
	@echo "Local targets (requires mdbook, mkdocs installed locally):"
	@echo "  all              Build both book and website"
	@echo "  book             Build the book (mdBook)"
	@echo "  website          Build the website (MkDocs)"
	@echo "  serve-book       Start book development server"
	@echo "  serve-web        Start website development server"
	@echo ""
	@echo "Other:"
	@echo "  setup            Setup symlinks only"
	@echo "  clean            Remove all build artifacts"

# =============================================================================
# Container targets (recommended)
# =============================================================================

container:
	@echo "Building publishing container..."
	$(CONTAINER_CMD) build -t $(CONTAINER_NAME) -f container/Containerfile container/

docker-all: container
	$(CONTAINER_CMD) run --rm -v $(CURDIR)/..:/workspace:Z $(CONTAINER_NAME) all

docker-book: container
	$(CONTAINER_CMD) run --rm -v $(CURDIR)/..:/workspace:Z $(CONTAINER_NAME) book

docker-web: container
	$(CONTAINER_CMD) run --rm -v $(CURDIR)/..:/workspace:Z $(CONTAINER_NAME) website

docker-serve-book: container
	@echo "Starting book server on http://localhost:3000"
	$(CONTAINER_CMD) run --rm -p 3000:3000 -v $(CURDIR)/..:/workspace:Z $(CONTAINER_NAME) book-serve

docker-serve-web: container
	@echo "Starting website server on http://localhost:8000"
	$(CONTAINER_CMD) run --rm -p 8000:8000 -v $(CURDIR)/..:/workspace:Z $(CONTAINER_NAME) website-serve

# =============================================================================
# Local targets (requires tools installed locally)
# =============================================================================

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
