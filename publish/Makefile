.PHONY: book book-pdf book-epub website clean help container \
        serve-book serve serve-api publish versions shell all

# Container settings
CONTAINER_IMAGE := gorai-publish
CONTAINER_RUNTIME := $(shell command -v podman 2>/dev/null || command -v docker 2>/dev/null)
WORKSPACE := $(shell cd .. && pwd)

# SELinux label for volume mounts (use :Z for podman on SELinux systems)
# Check if runtime contains 'podman' anywhere in the path
ifneq (,$(findstring podman,$(CONTAINER_RUNTIME)))
    VOLUME_OPTS := :Z
else
    VOLUME_OPTS :=
endif

# Container run command
RUN := $(CONTAINER_RUNTIME) run --rm -v $(WORKSPACE):/workspace$(VOLUME_OPTS)

# Default target - show help
.DEFAULT_GOAL := help

# Help
help:
	@echo "Gorai Publishing Build System"
	@echo ""
	@echo "All builds run inside a container - no local tool installation required."
	@echo ""
	@echo "Setup:"
	@echo "  container        Build the publishing container image"
	@echo ""
	@echo "Book Targets:"
	@echo "  book             Build PDF and ePub"
	@echo "  book-pdf         Build PDF only"
	@echo "  book-epub        Build ePub only"
	@echo "  serve-book       Preview book as HTML (port 8000)"
	@echo ""
	@echo "Website Targets:"
	@echo "  website          Build Hugo website"
	@echo "  serve            Start Hugo dev server (port 1313)"
	@echo ""
	@echo "Combined Targets:"
	@echo "  all              Build book, copy downloads, build website"
	@echo ""
	@echo "API Documentation:"
	@echo "  serve-api        Start pkgsite server (port 6060)"
	@echo ""
	@echo "Deployment:"
	@echo "  publish          Build and deploy to remote host"
	@echo "                   Usage: make publish DEST=host:path"
	@echo ""
	@echo "Utility:"
	@echo "  versions         Show tool versions in container"
	@echo "  shell            Start interactive shell in container"
	@echo "  clean            Remove all build artifacts"
	@echo ""
	@echo "Container runtime: $(CONTAINER_RUNTIME)"
	@echo "Workspace: $(WORKSPACE)"

# Build all (book, copy downloads, website)
all: check-container
	$(RUN) $(CONTAINER_IMAGE) website-with-book

# Build container image
container:
	@echo "Building container image..."
	$(CONTAINER_RUNTIME) build -t $(CONTAINER_IMAGE) container/

# Check container exists (works with both podman and docker)
check-container:
	@$(CONTAINER_RUNTIME) image inspect $(CONTAINER_IMAGE) >/dev/null 2>&1 || \
		(echo "Container image '$(CONTAINER_IMAGE)' not found. Run 'make container' first." && exit 1)

# Book targets
book: check-container
	$(RUN) $(CONTAINER_IMAGE) book

book-pdf: check-container
	$(RUN) $(CONTAINER_IMAGE) book-pdf

book-epub: check-container
	$(RUN) $(CONTAINER_IMAGE) book-epub

serve-book: check-container
	$(RUN) -it -p 8000:8000 $(CONTAINER_IMAGE) serve-book

# Website targets
website: check-container
	$(RUN) $(CONTAINER_IMAGE) website

serve: check-container
	$(RUN) -it -p 1313:1313 $(CONTAINER_IMAGE) serve-website


# API documentation
serve-api: check-container
	$(RUN) -it -p 6060:6060 $(CONTAINER_IMAGE) serve-api

# Clean all
clean: check-container
	$(RUN) $(CONTAINER_IMAGE) clean
	@echo "All build artifacts cleaned"

# Show versions
versions: check-container
	$(RUN) $(CONTAINER_IMAGE) versions

# Interactive shell
shell: check-container
	$(RUN) -it $(CONTAINER_IMAGE) shell

# Publish website to remote host
# Usage: make publish DEST=user@host:/path/to/webroot
publish: website
ifndef DEST
	$(error DEST is required. Usage: make publish DEST=host:path)
endif
	@echo "Publishing website to $(DEST)..."
	scp -r dist/website/* $(DEST)
	@echo "Published successfully to $(DEST)"

# Validate content structure
validate: check-container
	$(RUN) $(CONTAINER_IMAGE) validate
