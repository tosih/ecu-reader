#!/bin/bash

# Build script for Motronic M2.1 GTK GUI
# This script checks for dependencies and builds the GTK application

set -e  # Exit on error

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Motronic M2.1 ECU Tool - GTK GUI Build Script"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo

# Check if pkg-config is installed
if ! command -v pkg-config &> /dev/null; then
    echo "❌ ERROR: pkg-config is not installed"
    echo "   Install it with: sudo apt install pkg-config"
    exit 1
fi

# Check for GTK4 dependencies
echo "🔍 Checking for GTK4 dependencies..."

if ! pkg-config --exists gtk4; then
    echo "❌ ERROR: GTK4 development libraries not found"
    echo
    echo "Please install GTK4 dependencies:"
    echo
    echo "  Ubuntu/Debian:"
    echo "    sudo apt install libgtk-4-dev gobject-introspection libgirepository1.0-dev"
    echo
    echo "  Fedora:"
    echo "    sudo dnf install gtk4-devel gobject-introspection-devel"
    echo
    echo "  Arch:"
    echo "    sudo pacman -S gtk4 gobject-introspection"
    echo
    echo "See GTK_BUILD.md for more information."
    exit 1
fi

GTK_VERSION=$(pkg-config --modversion gtk4)
echo "✅ Found GTK4 version $GTK_VERSION"

# Check Go version
echo "🔍 Checking Go version..."
if ! command -v go &> /dev/null; then
    echo "❌ ERROR: Go is not installed"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo "✅ Found Go $GO_VERSION"

# Download dependencies if needed
echo
echo "📦 Downloading Go dependencies..."
go mod download

# Build the application
echo
echo "🔨 Building GTK application..."
echo "⏱️  NOTE: First build will take 10-15 minutes (this is normal)"
echo

if go build -o motronic-gtk main-gtk.go; then
    echo
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "✅ Build successful!"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo
    echo "Run the application with:"
    echo "  ./motronic-gtk"
    echo
else
    echo
    echo "❌ Build failed!"
    echo "See error messages above for details."
    echo "Check GTK_BUILD.md for troubleshooting."
    exit 1
fi
