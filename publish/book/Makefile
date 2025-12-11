.PHONY: all pdf epub html clean help

# This Makefile is for standalone use outside the container.
# For container-based builds, use: make -C .. book
#
# Requirements:
#   - pandoc
#   - texlive-xetex
#   - texlive-fonts-recommended

# Collect all chapter files in order
FRONTMATTER := $(wildcard chapters/00-frontmatter.md)
CHAPTERS := $(sort $(wildcard chapters/[0-1][0-9]-*.md))
APPENDICES := $(wildcard chapters/99-appendices.md)
ALL_CHAPTERS := $(FRONTMATTER) $(CHAPTERS) $(APPENDICES)

# Default target
all: pdf epub

# Help
help:
	@echo "Gorai Book Build System (Pandoc)"
	@echo ""
	@echo "NOTE: For container-based builds, use: make -C .. book"
	@echo ""
	@echo "Targets:"
	@echo "  all      - Build PDF and ePub (default)"
	@echo "  pdf      - Build PDF only"
	@echo "  epub     - Build ePub only"
	@echo "  html     - Build single-page HTML preview"
	@echo "  clean    - Remove all build artifacts"
	@echo ""
	@echo "Requirements:"
	@echo "  - pandoc"
	@echo "  - texlive-xetex (for PDF)"
	@echo "  - texlive-fonts-recommended (for PDF)"

# PDF output
pdf: dist/gorai-book.pdf

dist/gorai-book.pdf: $(ALL_CHAPTERS) metadata.yaml templates/pdf-template.tex
	@mkdir -p dist
	@echo "Building PDF..."
	pandoc \
		--metadata-file=metadata.yaml \
		--template=templates/pdf-template.tex \
		--pdf-engine=xelatex \
		--resource-path=.:images:../images:../images/logos \
		--toc \
		--number-sections \
		--highlight-style=tango \
		--top-level-division=chapter \
		-o $@ \
		$(ALL_CHAPTERS)
	@echo "PDF created: $@"

# ePub output
epub: dist/gorai-book.epub

dist/gorai-book.epub: $(ALL_CHAPTERS) metadata.yaml templates/epub.css
	@mkdir -p dist
	@echo "Building ePub..."
	pandoc \
		--metadata-file=metadata.yaml \
		--css=templates/epub.css \
		--toc \
		--number-sections \
		--top-level-division=chapter \
		-o $@ \
		$(ALL_CHAPTERS)
	@echo "ePub created: $@"

# HTML preview (single page)
html: dist/gorai-book.html

dist/gorai-book.html: $(ALL_CHAPTERS) metadata.yaml templates/html.css
	@mkdir -p dist
	@echo "Building HTML preview..."
	pandoc \
		--metadata-file=metadata.yaml \
		--css=templates/html.css \
		--standalone \
		--toc \
		--number-sections \
		--top-level-division=chapter \
		-o $@ \
		$(ALL_CHAPTERS)
	@echo "HTML created: $@"

# Clean build artifacts
clean:
	rm -rf dist/
	@echo "Build artifacts cleaned"
