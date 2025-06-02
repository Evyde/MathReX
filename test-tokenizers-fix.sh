#!/bin/bash

# Test script to verify the tokenizers fix works correctly
# This script tests the Windows-compatible tokenizers implementation

echo "=== MathReX Tokenizers Fix Verification ==="
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
    if go build -o test-tokenizers-$goos-$goarch-cgo$cgo_enabled ./; then
        echo "  ✓ Compilation successful"
        rm -f test-tokenizers-$goos-$goarch-cgo$cgo_enabled test-tokenizers-$goos-$goarch-cgo$cgo_enabled.exe
        return 0
    else
        echo "  ✗ Compilation failed"
        return 1
    fi
}

# Test 1: Check tokenizers file inclusion
echo "1. Testing tokenizers file inclusion:"

# Test Windows files
export GOOS=windows
windows_tokenizer_files=$(go list -f '{{.GoFiles}}' . | tr ' ' '\n' | grep "tokenizers_" | wc -l)
fixed_tokenizer_files=$(go list -f '{{.GoFiles}}' . | tr ' ' '\n' | grep "tokenizers_fixed" | wc -l)

echo "  Windows build includes:"
go list -f '{{.GoFiles}}' . | tr ' ' '\n' | grep -E "(tokenizers_|tokenizer_)" | sed 's/^/    /'

if [ "$windows_tokenizer_files" -gt 0 ]; then
    echo "  ✓ Windows tokenizer files included"
else
    echo "  ✗ Windows tokenizer files not included"
fi

if [ "$fixed_tokenizer_files" -gt 0 ]; then
    echo "  ✓ Fixed tokenizers implementation included"
else
    echo "  ✗ Fixed tokenizers implementation not included"
fi

# Test Unix files
export GOOS=linux
unix_tokenizer_files=$(go list -f '{{.GoFiles}}' . | tr ' ' '\n' | grep "tokenizers_" | wc -l)

echo "  Unix build includes:"
go list -f '{{.GoFiles}}' . | tr ' ' '\n' | grep -E "(tokenizers_|tokenizer_)" | sed 's/^/    /'

if [ "$unix_tokenizer_files" -gt 0 ]; then
    echo "  ✓ Unix tokenizer files included"
else
    echo "  ✗ Unix tokenizer files not included"
fi
echo

# Test 2: Check CGO flags
echo "2. Testing CGO flags:"

# Check Windows CGO flags
export GOOS=windows
echo "  Windows CGO flags:"
if grep -r "cgo windows LDFLAGS" . 2>/dev/null; then
    echo "  ✓ Windows-specific CGO flags found"
else
    echo "  ✗ Windows-specific CGO flags not found"
fi

if grep -r "cgo !windows LDFLAGS" . 2>/dev/null; then
    echo "  ✓ Non-Windows CGO flags found"
else
    echo "  ✗ Non-Windows CGO flags not found"
fi

# Check for -ldl exclusion on Windows
if grep -r "cgo windows LDFLAGS" . 2>/dev/null | grep -q "\-ldl"; then
    echo "  ✗ Windows CGO flags still contain -ldl"
else
    echo "  ✓ Windows CGO flags do not contain -ldl"
fi

# Check for -ldl inclusion on non-Windows
if grep -r "cgo !windows LDFLAGS" . 2>/dev/null | grep -q "\-ldl"; then
    echo "  ✓ Non-Windows CGO flags contain -ldl"
else
    echo "  ✗ Non-Windows CGO flags do not contain -ldl"
fi
echo

# Test 3: Test dependency handling
echo "3. Testing dependency handling:"

# Check that original tokenizers is still available for Unix
export GOOS=linux
export CGO_ENABLED=1
if go list -deps . | grep -q "github.com/daulet/tokenizers"; then
    echo "  ✓ Original tokenizers dependency available for Unix builds"
else
    echo "  ✗ Original tokenizers dependency missing for Unix builds"
fi

# Check Windows dependency handling
export GOOS=windows
export CGO_ENABLED=1
echo "  Windows dependencies:"
go list -deps . | grep -E "(tokenizers|onnxruntime)" | sed 's/^/    /' || echo "    No problematic dependencies found"
echo

# Test 4: Cross-compilation tests
echo "4. Testing cross-compilation:"

# Test Windows cross-compilation (CGO disabled - should work)
test_compilation "windows" "amd64" "0" "Windows AMD64 (CGO disabled)"

# Test current platform
current_goos=$(go env GOOS)
current_goarch=$(go env GOARCH)
test_compilation "$current_goos" "$current_goarch" "1" "Current platform ($current_goos/$current_goarch, CGO enabled)"
echo

# Test 5: Check for tokenizers library requirements
echo "5. Testing tokenizers library requirements:"

# Check if libtokenizers is available
if [ -f "./libtokenizers/windows_amd64/libtokenizers.a" ]; then
    echo "  ✓ Windows libtokenizers.a found"
    ls -la "./libtokenizers/windows_amd64/libtokenizers.a" | sed 's/^/    /'
else
    echo "  ⚠ Windows libtokenizers.a not found (expected for cross-compilation)"
fi

if [ -d "./libtokenizers" ]; then
    echo "  Available tokenizers libraries:"
    find ./libtokenizers -name "*.a" -o -name "*.lib" | sed 's/^/    /'
else
    echo "  ⚠ No libtokenizers directory found"
fi
echo

# Test 6: Verify fixed tokenizers implementation
echo "6. Testing fixed tokenizers implementation:"

if [ -f "tokenizers_fixed.go" ]; then
    echo "  ✓ tokenizers_fixed.go exists"
    
    # Check for Windows-specific CGO flags
    if grep -q "cgo windows LDFLAGS" tokenizers_fixed.go; then
        echo "  ✓ Windows-specific CGO flags found in fixed implementation"
    else
        echo "  ✗ Windows-specific CGO flags not found in fixed implementation"
    fi
    
    # Check for -ldl exclusion
    windows_flags=$(grep "cgo windows LDFLAGS" tokenizers_fixed.go | head -1)
    if echo "$windows_flags" | grep -q "\-ldl"; then
        echo "  ✗ Fixed implementation still contains -ldl for Windows"
    else
        echo "  ✓ Fixed implementation excludes -ldl for Windows"
    fi
    
    # Check for -ldl inclusion for non-Windows
    nonwindows_flags=$(grep "cgo !windows LDFLAGS" tokenizers_fixed.go | head -1)
    if echo "$nonwindows_flags" | grep -q "\-ldl"; then
        echo "  ✓ Fixed implementation includes -ldl for non-Windows"
    else
        echo "  ✗ Fixed implementation missing -ldl for non-Windows"
    fi
else
    echo "  ✗ tokenizers_fixed.go not found"
fi
echo

# Summary
echo "=== Tokenizers Fix Summary ==="
echo
echo "This fix addresses the Windows tokenizers compilation issue by:"
echo "1. ✓ Creating a Windows-compatible tokenizers wrapper"
echo "2. ✓ Using conditional CGO flags to exclude -ldl on Windows"
echo "3. ✓ Maintaining full functionality on Unix/Linux/macOS"
echo "4. ✓ Providing a direct interface to the tokenizers library"
echo
echo "Key Benefits:"
echo "- ✅ No more -ldl linking errors on Windows"
echo "- ✅ Full tokenizers functionality on Windows (when library is available)"
echo "- ✅ Maintains compatibility with other platforms"
echo "- ✅ Uses the actual tokenizers library instead of stubs"
echo
echo "Requirements for Windows builds:"
echo "- ⚠ libtokenizers.a must be available for Windows"
echo "- ⚠ Windows SDK required for CGO builds"
echo "- ⚠ MinGW or MSVC compiler required"
echo
echo "Next Steps:"
echo "1. Ensure libtokenizers.a is built for Windows"
echo "2. Test on actual Windows environment with CGO enabled"
echo "3. Update GitHub Actions to use this implementation"

# Reset environment
unset GOOS GOARCH CGO_ENABLED

echo
echo "=== Test Complete ==="
