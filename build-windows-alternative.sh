#!/bin/bash

# Alternative Windows Build Script
# This script tries a completely different approach to Windows building

echo "=== Alternative Windows Build Script ==="

# Step 1: Analyze the problem more deeply
echo "1. Deep analysis of the problem..."

# Check which packages actually use CGO
echo "Packages with CGO files:"
go list -f '{{if .CgoFiles}}{{.ImportPath}}: {{.CgoFiles}}{{end}}' ./... 2>/dev/null

# Check the actual error in more detail
echo "2. Testing minimal CGO build..."

# Create a minimal test program to isolate the issue
cat > test_cgo.go << 'EOF'
package main

/*
#include <stdio.h>
void hello() {
    printf("Hello from C!\n");
}
*/
import "C"

func main() {
    C.hello()
}
EOF

echo "Testing basic CGO functionality..."
export GOOS=windows
export GOARCH=amd64
export CGO_ENABLED=1
export CC=gcc

go build -o test_cgo.exe test_cgo.go 2>&1 && {
    echo "✓ Basic CGO works"
    rm -f test_cgo.exe test_cgo.go
} || {
    echo "✗ Basic CGO fails - this indicates a fundamental CGO issue"
    rm -f test_cgo.go
}

# Step 3: Try building without problematic dependencies
echo "3. Trying to build without problematic CGO dependencies..."

# Create a temporary go.mod that excludes problematic packages
cp go.mod go.mod.backup
cp go.sum go.sum.backup 2>/dev/null || true

# Try to identify and replace problematic dependencies
echo "Checking for alternative packages..."

# Check if we can build with CGO disabled for specific packages
echo "4. Attempting selective CGO disable..."

# Set build tags to disable CGO for specific packages
export CGO_ENABLED=1
export GOOS=windows
export GOARCH=amd64
export CC=gcc
export CGO_LDFLAGS="-L./libtokenizers/windows_amd64/"

# Try building with various build tags
echo "Trying build with nocgo tag..."
go build -tags "nocgo" -o bin/MathReX-windows-amd64.exe ./ 2>&1 && {
    echo "✓ Build with nocgo tag successful!"
    exit 0
}

echo "Trying build with static tag..."
go build -tags "static" -o bin/MathReX-windows-amd64.exe ./ 2>&1 && {
    echo "✓ Build with static tag successful!"
    exit 0
}

echo "Trying build with netgo tag..."
go build -tags "netgo" -o bin/MathReX-windows-amd64.exe ./ 2>&1 && {
    echo "✓ Build with netgo tag successful!"
    exit 0
}

# Step 5: Try with modified linker flags
echo "5. Trying with modified linker flags..."

# Try with different linker modes
echo "Trying with internal linking..."
go build -ldflags="-linkmode=internal" -o bin/MathReX-windows-amd64.exe ./ 2>&1 && {
    echo "✓ Internal linking successful!"
    exit 0
}

echo "Trying with external linking and static flags..."
go build -ldflags="-linkmode=external -extldflags=-static" -o bin/MathReX-windows-amd64.exe ./ 2>&1 && {
    echo "✓ External static linking successful!"
    exit 0
}

# Step 6: Try building individual problematic packages
echo "6. Testing individual problematic packages..."

problematic_packages=(
    "github.com/daulet/tokenizers"
    "github.com/yalue/onnxruntime_go"
    "github.com/getlantern/systray"
    "github.com/kbinani/screenshot"
    "github.com/robotn/gohook"
)

for pkg in "${problematic_packages[@]}"; do
    echo "Testing package: $pkg"
    go build -v "$pkg" 2>&1 | grep -i "ldl" && {
        echo "Found -ldl issue in package: $pkg"
    }
done

# Step 7: Last resort - try to build without CGO entirely
echo "7. Last resort: building without CGO..."
export CGO_ENABLED=0
go build -o bin/MathReX-windows-amd64.exe ./ 2>&1 && {
    echo "✓ CGO disabled build successful!"
    echo "Note: This build may have limited functionality due to disabled CGO"
    exit 0
} || {
    echo "✗ Even CGO disabled build failed"
}

# Restore original files
mv go.mod.backup go.mod 2>/dev/null || true
mv go.sum.backup go.sum 2>/dev/null || true

echo "=== All build attempts failed ==="
echo "This suggests a deeper issue with the Go toolchain or dependencies"

exit 1
