#!/bin/bash
# Check dependencies for Gorai publishing
# Usage: ./scripts/check-deps.sh

set -e

echo "=== Checking Gorai Publishing Dependencies ==="
echo ""

MISSING=0

check_cmd() {
    local cmd=$1
    local pkg=$2
    local purpose=$3

    if command -v "$cmd" &> /dev/null; then
        local version=$("$cmd" --version 2>&1 | head -1)
        echo "[OK] $cmd: $version"
    else
        echo "[MISSING] $cmd - Install: $pkg (for $purpose)"
        MISSING=$((MISSING + 1))
    fi
}

echo "Book Generation:"
echo "----------------"
check_cmd "pandoc" "pandoc" "Markdown to PDF/ePub conversion"
check_cmd "xelatex" "texlive-xetex" "PDF generation"

echo ""
echo "Website Generation:"
echo "-------------------"
check_cmd "hugo" "hugo" "Static site generation"

echo ""
echo "Optional Tools:"
echo "---------------"
check_cmd "python3" "python3" "Local preview server"

echo ""

if [ $MISSING -gt 0 ]; then
    echo "=== $MISSING missing dependencies ==="
    echo ""
    echo "Install on Ubuntu/Debian:"
    echo "  sudo apt install pandoc texlive-xetex texlive-fonts-recommended hugo"
    echo ""
    echo "Install on macOS:"
    echo "  brew install pandoc hugo"
    echo "  brew install --cask mactex-no-gui"
    exit 1
else
    echo "=== All dependencies satisfied ==="
fi
