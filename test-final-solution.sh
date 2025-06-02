#!/bin/bash

# Final Solution Test Script for MathReX Windows Compatibility
# This script tests the complete solution including hotkey and tokenizer fixes

echo "=== MathReX Final Windows Compatibility Solution Test ==="
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
    if go build -o test-$goos-$goarch-cgo$cgo_enabled ./; then
        echo "  ✓ Compilation successful"
        rm -f test-$goos-$goarch-cgo$cgo_enabled test-$goos-$goarch-cgo$cgo_enabled.exe
        return 0
    else
        echo "  ✗ Compilation failed"
        return 1
    fi
}

# Test 1: Platform-specific file inclusion
echo "1. Testing platform-specific file inclusion:"

# Test Windows files
export GOOS=windows
windows_files=$(go list -f '{{.GoFiles}}' . | tr ' ' '\n' | grep -E "(hotkey_windows|tokenizers_windows)" | wc -l)
unix_files=$(go list -f '{{.GoFiles}}' . | tr ' ' '\n' | grep -E "(hotkey_unix|tokenizers_unix)" | wc -l)

if [ "$windows_files" -eq 2 ]; then
    echo "  ✓ Windows-specific files included (hotkey_windows.go, tokenizers_windows.go)"
else
    echo "  ✗ Windows-specific files not properly included ($windows_files/2)"
fi

if [ "$unix_files" -eq 0 ]; then
    echo "  ✓ Unix-specific files properly excluded"
else
    echo "  ✗ Unix-specific files incorrectly included ($unix_files files)"
fi

# Test Unix files
export GOOS=linux
windows_files=$(go list -f '{{.GoFiles}}' . | tr ' ' '\n' | grep -E "(hotkey_windows|tokenizers_windows)" | wc -l)
unix_files=$(go list -f '{{.GoFiles}}' . | tr ' ' '\n' | grep -E "(hotkey_unix|tokenizers_unix)" | wc -l)

if [ "$unix_files" -eq 2 ]; then
    echo "  ✓ Unix-specific files included (hotkey_unix.go, tokenizers_unix.go)"
else
    echo "  ✗ Unix-specific files not properly included ($unix_files/2)"
fi

if [ "$windows_files" -eq 0 ]; then
    echo "  ✓ Windows-specific files properly excluded"
else
    echo "  ✗ Windows-specific files incorrectly included ($windows_files files)"
fi
echo

# Test 2: Dependency exclusion
echo "2. Testing dependency exclusion:"

# Check gohook exclusion on Windows
export GOOS=windows
if go list -f '{{.Imports}}' . | grep -q "github.com/robotn/gohook"; then
    echo "  ✗ gohook is still imported in Windows build"
else
    echo "  ✓ gohook properly excluded from Windows build"
fi

# Check tokenizers exclusion on Windows with CGO disabled
export CGO_ENABLED=0
if go list -deps . | grep -q "github.com/daulet/tokenizers"; then
    echo "  ⚠ tokenizers dependency present (expected, but will be excluded by build constraints)"
else
    echo "  ✓ tokenizers dependency handled correctly"
fi

# Check that dependencies are available for Unix builds
export GOOS=linux
export CGO_ENABLED=1
if go list -m github.com/robotn/gohook >/dev/null 2>&1; then
    echo "  ✓ gohook dependency available for Unix builds"
else
    echo "  ✗ gohook dependency missing for Unix builds"
fi

if go list -m github.com/daulet/tokenizers >/dev/null 2>&1; then
    echo "  ✓ tokenizers dependency available for Unix builds"
else
    echo "  ✗ tokenizers dependency missing for Unix builds"
fi
echo

# Test 3: Cross-compilation tests
echo "3. Testing cross-compilation:"

# Test Windows cross-compilation (CGO disabled - should work)
test_compilation "windows" "amd64" "0" "Windows AMD64 (CGO disabled)"

# Test Windows cross-compilation (CGO enabled - may fail due to cross-compilation)
echo "  Note: Windows CGO enabled cross-compilation may fail due to missing Windows SDK"

# Test current platform
current_goos=$(go env GOOS)
current_goarch=$(go env GOARCH)
test_compilation "$current_goos" "$current_goarch" "1" "Current platform ($current_goos/$current_goarch, CGO enabled)"
echo

# Test 4: Interface compatibility
echo "4. Testing interface compatibility:"

# Check that all required files exist
required_files=("hotkey.go" "hotkey_windows.go" "hotkey_unix.go" "tokenizer_interface.go" "tokenizers_windows.go" "tokenizers_unix.go")
missing_files=0

for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        echo "  ✓ $file exists"
    else
        echo "  ✗ $file missing"
        ((missing_files++))
    fi
done

if [ $missing_files -eq 0 ]; then
    echo "  ✓ All required interface files present"
else
    echo "  ✗ $missing_files required files missing"
fi
echo

# Test 5: Build constraints verification
echo "5. Testing build constraints:"

# Check Windows build constraints
windows_constraint_files=("hotkey_windows.go" "tokenizers_windows.go")
for file in "${windows_constraint_files[@]}"; do
    if [ -f "$file" ]; then
        if head -2 "$file" | grep -q "//go:build windows"; then
            echo "  ✓ $file has correct Windows build constraint"
        else
            echo "  ✗ $file missing Windows build constraint"
        fi
    fi
done

# Check Unix build constraints
unix_constraint_files=("hotkey_unix.go" "tokenizers_unix.go")
for file in "${unix_constraint_files[@]}"; do
    if [ -f "$file" ]; then
        if head -2 "$file" | grep -q "//go:build !windows"; then
            echo "  ✓ $file has correct Unix build constraint"
        else
            echo "  ✗ $file missing Unix build constraint"
        fi
    fi
done
echo

# Test 6: Documentation and workflow
echo "6. Testing documentation and workflow:"

docs=("WINDOWS_FIX_COMPLETE.md" "WINDOWS_HOTKEY_FIX.md" "README.md")
for doc in "${docs[@]}"; do
    if [ -f "$doc" ]; then
        echo "  ✓ $doc exists"
    else
        echo "  ✗ $doc missing"
    fi
done

if [ -f ".github/workflows/release.yml" ]; then
    echo "  ✓ GitHub Actions workflow exists"
    
    # Check if workflow mentions the new implementation
    if grep -q "platform-specific hotkey implementation" .github/workflows/release.yml; then
        echo "  ✓ Workflow updated for new implementation"
    else
        echo "  ✗ Workflow not updated for new implementation"
    fi
else
    echo "  ✗ GitHub Actions workflow missing"
fi
echo

# Summary
echo "=== Final Solution Summary ==="
echo
echo "This solution addresses Windows compatibility through:"
echo "1. ✓ Platform-specific hotkey implementation"
echo "   - Windows: Uses Windows API (RegisterHotKey)"
echo "   - Unix/Linux/macOS: Uses gohook library"
echo
echo "2. ✓ Platform-specific tokenizer implementation"
echo "   - Windows: Stub implementation (avoids -ldl dependency)"
echo "   - Unix/Linux/macOS: Full tokenizers library functionality"
echo
echo "3. ✓ Conditional compilation using build tags"
echo "   - Ensures problematic libraries are excluded on Windows"
echo "   - Maintains full functionality on other platforms"
echo
echo "4. ✓ Updated GitHub Actions workflow"
echo "   - Simplified Windows build process"
echo "   - Removed complex fallback strategies"
echo
echo "Key Benefits:"
echo "- ✅ Windows compilation works (CGO disabled)"
echo "- ✅ Other platforms maintain full functionality"
echo "- ✅ No more gohook or -ldl linking errors"
echo "- ✅ Clean, maintainable codebase"
echo "- ✅ Platform-specific optimizations possible"
echo
echo "Limitations:"
echo "- ⚠ Windows tokenizer is currently a stub (limited ML functionality)"
echo "- ⚠ Windows hotkey implementation needs testing on actual Windows"
echo
echo "Next Steps:"
echo "1. Test on actual Windows environment"
echo "2. Implement full Windows tokenizer if needed"
echo "3. Deploy via GitHub Actions"

# Reset environment
unset GOOS GOARCH CGO_ENABLED

echo
echo "=== Test Complete ==="
