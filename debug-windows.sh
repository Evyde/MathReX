#!/bin/bash

# Windows Build Debug Script
# This script helps diagnose Windows build issues

echo "=== Windows Build Debug Script ==="

# Check environment
echo "1. Environment Check:"
echo "   GOOS: $GOOS"
echo "   GOARCH: $GOARCH"
echo "   CGO_ENABLED: $CGO_ENABLED"
echo "   CC: $CC"
echo "   CGO_LDFLAGS: $CGO_LDFLAGS"

# Check Go
echo "2. Go Information:"
go version
go env GOOS
go env GOARCH
go env CGO_ENABLED
go env CC

# Check compiler
echo "3. Compiler Check:"
if command -v gcc &> /dev/null; then
    echo "   GCC found:"
    gcc --version | head -1
    which gcc
else
    echo "   GCC not found"
fi

if command -v cl.exe &> /dev/null; then
    echo "   MSVC found:"
    cl.exe 2>&1 | head -1
else
    echo "   MSVC not found"
fi

# Check dependencies
echo "4. Dependencies Check:"
echo "   Tokenizers library:"
ls -la ./libtokenizers/windows_amd64/ 2>/dev/null || echo "   Not found"

echo "   ONNX Runtime:"
ls -la ./onnxruntime/amd64_windows/ 2>/dev/null || echo "   Not found"

# Check CGO packages
echo "5. CGO Packages:"
go list -f '{{if .CgoFiles}}{{.ImportPath}}{{end}}' ./... 2>/dev/null || echo "   No local CGO packages"

# Check dependencies (gohook should now be conditional)
echo "6. Platform-specific Dependencies:"
echo "   Checking gohook usage (should be Unix-only):"
go list -f '{{if .CgoFiles}}{{.ImportPath}} {{.CgoFiles}}{{end}}' ./... 2>/dev/null | grep -v "hotkey_unix.go" | grep "gohook" && echo "   ⚠ gohook found in Windows build!" || echo "   ✓ gohook properly excluded from Windows"
echo "   Dependencies:"
go mod graph 2>/dev/null | grep -E "(tokenizers|onnxruntime|systray|screenshot)" || echo "   Core dependencies found"

# Test simple build
echo "7. Test Builds:"

echo "   Testing CGO disabled build..."
CGO_ENABLED=0 go build -o test-nocgo.exe ./ 2>&1 && echo "   ✓ CGO disabled build successful" || echo "   ✗ CGO disabled build failed"

echo "   Testing with minimal CGO..."
export CGO_ENABLED=1
export CGO_LDFLAGS="-L./libtokenizers/windows_amd64/"
go build -o test-cgo.exe ./ 2>&1 && echo "   ✓ CGO build successful" || {
    echo "   ✗ CGO build failed"
    echo "   Creating fake libdl and retrying..."

    if [ -f "create-fake-libdl.sh" ]; then
        chmod +x create-fake-libdl.sh
        ./create-fake-libdl.sh
        echo "   Retrying build with fake libdl..."
        go build -o test-cgo-libdl.exe ./ 2>&1 && echo "   ✓ CGO build with fake libdl successful" || {
            echo "   ✗ CGO build with fake libdl still failed"
            echo "   Trying with verbose output..."
            go build -v -x -o test-cgo-verbose.exe ./ 2>&1 | tail -20
        }
    else
        echo "   create-fake-libdl.sh not found, trying with verbose output..."
        go build -v -x -o test-cgo-verbose.exe ./ 2>&1 | tail -20
    fi
}

# Clean up test files
rm -f test-*.exe 2>/dev/null

echo "=== Debug Complete ==="
