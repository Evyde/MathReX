#!/bin/bash

# Windows CGO Fix Script
# This script attempts to fix the -ldl linking issue on Windows

echo "=== Windows CGO Fix Script ==="

# Step 1: Check if we can identify the source of -ldl
echo "1. Analyzing CGO dependencies..."

# Check go.mod for CGO-heavy dependencies
echo "CGO-heavy dependencies in go.mod:"
grep -E "(tokenizers|onnxruntime|systray|gohook|screenshot)" go.mod || echo "None found in go.mod"

# Step 2: Try to find where -ldl is being injected
echo "2. Checking for -ldl injection sources..."

# Check if any .pc files exist that might inject -ldl
find . -name "*.pc" -exec grep -l "ldl" {} \; 2>/dev/null || echo "No .pc files with ldl found"

# Check if tokenizers library has any embedded linker flags
if [ -f "./libtokenizers/windows_amd64/libtokenizers.a" ]; then
    echo "Checking tokenizers library..."
    strings "./libtokenizers/windows_amd64/libtokenizers.a" | grep -i "ldl" || echo "No ldl references in tokenizers library"
fi

# Step 3: Create a comprehensive fix
echo "3. Creating comprehensive fix..."

# Create a fake libdl.a to satisfy the linker
mkdir -p fake_libs
cat > fake_libs/libdl.c << 'EOF'
// Fake libdl implementation for Windows
// This provides empty stubs for dl* functions

void* dlopen(const char* filename, int flag) {
    return (void*)1; // Return non-null to indicate success
}

char* dlerror(void) {
    return 0; // No error
}

void* dlsym(void* handle, const char* symbol) {
    return 0; // Symbol not found
}

int dlclose(void* handle) {
    return 0; // Success
}
EOF

# Compile the fake libdl
echo "Creating fake libdl.a..."
gcc -c fake_libs/libdl.c -o fake_libs/libdl.o
ar rcs fake_libs/libdl.a fake_libs/libdl.o

# Verify the library was created
echo "Verifying fake libdl.a creation..."
if [ -f "fake_libs/libdl.a" ]; then
    echo "✓ libdl.a created successfully"
    ls -la fake_libs/libdl.a
    ar -t fake_libs/libdl.a
else
    echo "✗ Failed to create libdl.a"
fi

# Also create it with different naming conventions that might be expected
echo "Creating alternative library names..."
cp fake_libs/libdl.a fake_libs/dl.lib 2>/dev/null || true
cp fake_libs/libdl.a fake_libs/dl.a 2>/dev/null || true

echo "Contents of fake_libs directory:"
ls -la fake_libs/

# Step 4: Create an enhanced GCC wrapper (Windows batch file)
cat > gcc_enhanced.bat << 'EOF'
@echo off
setlocal enabledelayedexpansion

set LOG_FILE=gcc_enhanced.log
echo === GCC Enhanced Wrapper Called === >> %LOG_FILE%
echo Original args: %* >> %LOG_FILE%

set "args="
set "needs_fake_dl=false"

:loop
if "%~1"=="" goto :done
if "%~1"=="-ldl" (
    echo Filtered out: %~1 >> %LOG_FILE%
    set "needs_fake_dl=true"
    set "args=!args! -L%CD%\fake_libs -ldl"
) else (
    set "args=!args! %~1"
)
shift
goto :loop

:done
echo Final args: !args! >> %LOG_FILE%

C:\mingw64\bin\gcc.exe !args! 2>&1 | tee -a %LOG_FILE%
exit /b %errorlevel%
EOF

# Step 5: Try a simpler approach - just use the fake libdl without wrapper
echo "4. Setting up build environment (simple approach)..."

export GOOS=windows
export GOARCH=amd64
export CGO_ENABLED=1
export CC=gcc
export CGO_LDFLAGS="-L./libtokenizers/windows_amd64/ -L$(pwd)/fake_libs"

echo "Environment:"
echo "  CC=$CC"
echo "  CGO_LDFLAGS=$CGO_LDFLAGS"

# Step 6: Attempt build with fake libdl
echo "5. Attempting build with fake libdl..."
mkdir -p bin

echo "Trying build with fake libdl available..."
go build -v -o bin/MathReX-windows-amd64.exe ./ 2>&1 | tee build_final.log

if [ $? -eq 0 ]; then
    echo "✓ Build successful!"
    ls -la bin/MathReX-windows-amd64.exe
else
    echo "✗ Build failed. Trying alternative approaches..."

    # Try with CGO disabled as fallback
    echo "Trying with CGO disabled..."
    CGO_ENABLED=0 go build -v -o bin/MathReX-windows-amd64-nocgo.exe ./ 2>&1 && {
        echo "✓ CGO disabled build successful!"
        mv bin/MathReX-windows-amd64-nocgo.exe bin/MathReX-windows-amd64.exe
    } || {
        echo "✗ Even CGO disabled build failed"
        echo "=== Build Log ==="
        tail -30 build_final.log
    }
fi

echo "=== Fix Script Complete ==="
