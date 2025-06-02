#!/bin/bash

# Test script to verify the hotkey fix works correctly
# This script tests compilation for different platforms

echo "=== MathReX Hotkey Fix Verification ==="
echo

# Function to test compilation for a platform
test_compilation() {
    local goos=$1
    local goarch=$2
    local cgo_enabled=$3
    local description=$4
    
    echo "Testing: $description"
    echo "  GOOS=$goos GOARCH=$goarch CGO_ENABLED=$cgo_enabled"
    
    # Set environment
    export GOOS=$goos
    export GOARCH=$goarch
    export CGO_ENABLED=$cgo_enabled
    
    # Try compilation
    if go build -o test-$goos-$goarch ./; then
        echo "  ✓ Compilation successful"
        rm -f test-$goos-$goarch test-$goos-$goarch.exe
        return 0
    else
        echo "  ✗ Compilation failed"
        return 1
    fi
}

# Test current platform (should work)
echo "1. Testing current platform:"
if go build -o test-current ./; then
    echo "  ✓ Current platform compilation successful"
    rm -f test-current
else
    echo "  ✗ Current platform compilation failed"
fi
echo

# Test Windows cross-compilation (the main fix)
echo "2. Testing Windows cross-compilation:"
test_compilation "windows" "amd64" "0" "Windows AMD64 (CGO disabled)"
echo

# Test that gohook is properly excluded on Windows
echo "3. Testing gohook exclusion on Windows:"
export GOOS=windows
export GOARCH=amd64
export CGO_ENABLED=0

# Check if gohook is imported in Windows build
if go list -f '{{.Imports}}' . | grep -q "github.com/robotn/gohook"; then
    echo "  ✗ gohook is still imported in Windows build"
else
    echo "  ✓ gohook properly excluded from Windows build"
fi
echo

# Test that our hotkey files are included correctly
echo "4. Testing hotkey file inclusion:"

# Check Windows hotkey file
export GOOS=windows
if go list -f '{{.GoFiles}}' . | grep -q "hotkey_windows.go"; then
    echo "  ✓ hotkey_windows.go included in Windows build"
else
    echo "  ✗ hotkey_windows.go not included in Windows build"
fi

# Check Unix hotkey file exclusion on Windows
if go list -f '{{.GoFiles}}' . | grep -q "hotkey_unix.go"; then
    echo "  ✗ hotkey_unix.go incorrectly included in Windows build"
else
    echo "  ✓ hotkey_unix.go properly excluded from Windows build"
fi

# Check Unix hotkey file inclusion on Unix
export GOOS=linux
if go list -f '{{.GoFiles}}' . | grep -q "hotkey_unix.go"; then
    echo "  ✓ hotkey_unix.go included in Linux build"
else
    echo "  ✗ hotkey_unix.go not included in Linux build"
fi

# Check Windows hotkey file exclusion on Unix
if go list -f '{{.GoFiles}}' . | grep -q "hotkey_windows.go"; then
    echo "  ✗ hotkey_windows.go incorrectly included in Linux build"
else
    echo "  ✓ hotkey_windows.go properly excluded from Linux build"
fi
echo

# Test dependencies
echo "5. Testing dependencies:"

# Check that golang.org/x/sys is available
if go list -m golang.org/x/sys >/dev/null 2>&1; then
    echo "  ✓ golang.org/x/sys dependency available"
else
    echo "  ✗ golang.org/x/sys dependency missing"
fi

# Check that gohook is still available for Unix builds
if go list -m github.com/robotn/gohook >/dev/null 2>&1; then
    echo "  ✓ github.com/robotn/gohook dependency available"
else
    echo "  ✗ github.com/robotn/gohook dependency missing"
fi
echo

# Summary
echo "=== Summary ==="
echo "If all tests show ✓, the hotkey fix is working correctly."
echo "The main fix replaces gohook with platform-specific implementations:"
echo "  - Windows: Uses Windows API (RegisterHotKey)"
echo "  - Unix/Linux/macOS: Uses gohook (existing functionality)"
echo
echo "This should resolve the Windows compilation issues while maintaining"
echo "compatibility with other platforms."
echo

# Reset environment
unset GOOS GOARCH CGO_ENABLED

echo "=== Test Complete ==="
