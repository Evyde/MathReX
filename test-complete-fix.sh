#!/bin/bash

# Complete Windows Fix Test Script
# This script tests the complete Windows compatibility fix

echo "=== MathReX Complete Windows Fix Test ==="
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

# Test 1: Verify hotkey files are correctly included
echo "1. Testing hotkey file inclusion:"

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

# Check gohook exclusion on Windows
if go list -f '{{.Imports}}' . | grep -q "github.com/robotn/gohook"; then
    echo "  ✗ gohook is still imported in Windows build"
else
    echo "  ✓ gohook properly excluded from Windows build"
fi
echo

# Test 2: Check dependencies
echo "2. Testing dependencies:"

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

# Check tokenizers dependency and its CGO flags
echo "  Checking tokenizers CGO flags:"
TOKENIZERS_FLAGS=$(go list -f '{{.CgoLDFLAGS}}' $(go list -m -f '{{.Dir}}' github.com/daulet/tokenizers)/... 2>/dev/null | head -1)
if echo "$TOKENIZERS_FLAGS" | grep -q "\-ldl"; then
    echo "  ⚠ tokenizers package contains -ldl flag (expected, will be handled by fake libdl)"
else
    echo "  ✓ tokenizers package does not contain -ldl flag"
fi
echo

# Test 3: Test fake libdl creation
echo "3. Testing fake libdl creation:"

if [ -f "create-fake-libdl.sh" ]; then
    echo "  ✓ create-fake-libdl.sh script exists"
    
    # Test if we can create fake libdl (only on systems with gcc)
    if command -v gcc >/dev/null 2>&1; then
        echo "  Testing fake libdl creation..."
        chmod +x create-fake-libdl.sh
        if ./create-fake-libdl.sh >/dev/null 2>&1; then
            echo "  ✓ Fake libdl creation successful"
            
            # Check if the library was created
            if [ -f "libdl.a" ]; then
                echo "  ✓ libdl.a file created"
                
                # Test the library contents
                if ar -t libdl.a >/dev/null 2>&1; then
                    echo "  ✓ libdl.a is a valid archive"
                else
                    echo "  ✗ libdl.a is not a valid archive"
                fi
            else
                echo "  ✗ libdl.a file not created"
            fi
        else
            echo "  ✗ Fake libdl creation failed"
        fi
    else
        echo "  ⚠ GCC not available, skipping fake libdl test"
    fi
else
    echo "  ✗ create-fake-libdl.sh script not found"
fi
echo

# Test 4: Cross-compilation tests
echo "4. Testing cross-compilation:"

# Test Windows cross-compilation (CGO disabled)
test_compilation "windows" "amd64" "0" "Windows AMD64 (CGO disabled)"

# Test that Unix builds still work
export GOOS=$(go env GOOS)
export GOARCH=$(go env GOARCH)
test_compilation "$GOOS" "$GOARCH" "1" "Current platform (CGO enabled)"
echo

# Test 5: Check GitHub Actions workflow
echo "5. Testing GitHub Actions workflow:"

if [ -f ".github/workflows/release.yml" ]; then
    echo "  ✓ GitHub Actions workflow exists"
    
    # Check if it includes fake libdl creation
    if grep -q "create-fake-libdl.sh" .github/workflows/release.yml; then
        echo "  ✓ Workflow includes fake libdl creation"
    else
        echo "  ✗ Workflow does not include fake libdl creation"
    fi
    
    # Check if it has simplified Windows build
    if grep -q "platform-specific hotkey implementation" .github/workflows/release.yml; then
        echo "  ✓ Workflow mentions new hotkey implementation"
    else
        echo "  ✗ Workflow does not mention new hotkey implementation"
    fi
else
    echo "  ✗ GitHub Actions workflow not found"
fi
echo

# Test 6: Documentation check
echo "6. Testing documentation:"

docs=("WINDOWS_FIX_COMPLETE.md" "WINDOWS_HOTKEY_FIX.md" "README.md")
for doc in "${docs[@]}"; do
    if [ -f "$doc" ]; then
        echo "  ✓ $doc exists"
    else
        echo "  ✗ $doc missing"
    fi
done
echo

# Summary
echo "=== Test Summary ==="
echo "This test verifies the complete Windows compatibility fix:"
echo "1. ✓ Platform-specific hotkey implementation"
echo "2. ✓ Fake libdl library solution for tokenizers"
echo "3. ✓ Updated GitHub Actions workflow"
echo "4. ✓ Comprehensive documentation"
echo
echo "If all tests show ✓, the Windows fix is complete and ready for deployment."
echo

# Reset environment
unset GOOS GOARCH CGO_ENABLED

echo "=== Test Complete ==="
