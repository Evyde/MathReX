#!/bin/bash

# Create fake libdl for Windows builds
# This script creates a minimal libdl.a library to satisfy the -ldl requirement

echo "=== Creating fake libdl for Windows ==="

# Create directory for fake libraries
mkdir -p fake_libs

# Create fake libdl.c with stub implementations
cat > fake_libs/libdl.c << 'EOF'
// Fake libdl implementation for Windows
// These are stub implementations to satisfy linking requirements

#include <windows.h>

// dlopen equivalent - just return a handle
void* dlopen(const char* filename, int flag) {
    if (filename == NULL) {
        return GetModuleHandle(NULL);
    }
    return LoadLibraryA(filename);
}

// dlerror equivalent
char* dlerror(void) {
    static char error_msg[256];
    DWORD error = GetLastError();
    if (error == 0) {
        return NULL;
    }
    FormatMessageA(FORMAT_MESSAGE_FROM_SYSTEM | FORMAT_MESSAGE_IGNORE_INSERTS,
                   NULL, error, 0, error_msg, sizeof(error_msg), NULL);
    return error_msg;
}

// dlsym equivalent
void* dlsym(void* handle, const char* symbol) {
    if (handle == NULL) {
        return NULL;
    }
    return GetProcAddress((HMODULE)handle, symbol);
}

// dlclose equivalent
int dlclose(void* handle) {
    if (handle == NULL) {
        return 0;
    }
    return FreeLibrary((HMODULE)handle) ? 0 : -1;
}

// Additional symbols that might be needed
int dladdr(void* addr, void* info) {
    // Stub implementation
    return 0;
}

void* dlvsym(void* handle, const char* symbol, const char* version) {
    // Just call dlsym, ignore version
    return dlsym(handle, symbol);
}
EOF

echo "Created fake libdl.c"

# Compile the fake library
echo "Compiling fake libdl..."
gcc -c fake_libs/libdl.c -o fake_libs/libdl.o

if [ $? -eq 0 ]; then
    echo "✓ Compilation successful"
else
    echo "✗ Compilation failed"
    exit 1
fi

# Create the static library
echo "Creating libdl.a..."
ar rcs fake_libs/libdl.a fake_libs/libdl.o

if [ $? -eq 0 ]; then
    echo "✓ Library creation successful"
    echo "Created: $(ls -la fake_libs/libdl.a)"
else
    echo "✗ Library creation failed"
    exit 1
fi

# Verify the library contents
echo "Library contents:"
ar -t fake_libs/libdl.a

# Test the library
echo "Testing the fake library..."
cat > fake_libs/test.c << 'EOF'
#include <stdio.h>

// Declare the functions we expect to be in libdl
extern void* dlopen(const char* filename, int flag);
extern char* dlerror(void);
extern void* dlsym(void* handle, const char* symbol);
extern int dlclose(void* handle);

int main() {
    printf("Testing fake libdl...\n");
    
    // Test dlopen
    void* handle = dlopen(NULL, 0);
    printf("dlopen(NULL, 0) = %p\n", handle);
    
    // Test dlerror
    char* error = dlerror();
    printf("dlerror() = %s\n", error ? error : "NULL");
    
    // Test dlclose
    int result = dlclose(handle);
    printf("dlclose() = %d\n", result);
    
    printf("Fake libdl test completed successfully!\n");
    return 0;
}
EOF

# Compile and run the test
gcc fake_libs/test.c -L./fake_libs -ldl -o fake_libs/test.exe 2>&1 && {
    echo "✓ Test compilation successful"
    if [ -f fake_libs/test.exe ]; then
        echo "✓ Test executable created"
        # Don't run the test on non-Windows platforms
        echo "Note: Test executable created but not run (cross-platform compatibility)"
    fi
} || {
    echo "✗ Test compilation failed"
}

# Clean up test files
rm -f fake_libs/test.c fake_libs/test.exe fake_libs/libdl.o

# Copy the library to a location where the linker can find it
echo "Installing fake libdl..."

# Try to copy to MinGW lib directory if it exists and is writable
MINGW_LIB="/c/mingw64/x86_64-w64-mingw32/lib"
if [ -d "$MINGW_LIB" ] && [ -w "$MINGW_LIB" ]; then
    cp fake_libs/libdl.a "$MINGW_LIB/"
    echo "✓ Copied to MinGW lib directory: $MINGW_LIB"
else
    echo "Note: MinGW lib directory not accessible, library remains in fake_libs/"
fi

# Also copy to current directory for local builds
cp fake_libs/libdl.a ./
echo "✓ Copied to current directory"

echo "=== Fake libdl creation complete ==="
echo "Files created:"
echo "  - fake_libs/libdl.a (source)"
echo "  - ./libdl.a (for local builds)"
if [ -f "$MINGW_LIB/libdl.a" ]; then
    echo "  - $MINGW_LIB/libdl.a (system-wide)"
fi

echo ""
echo "Usage:"
echo "  The fake libdl.a library will now satisfy -ldl linking requirements"
echo "  on Windows builds. The library provides Windows-compatible implementations"
echo "  of dlopen, dlsym, dlclose, and dlerror using Windows LoadLibrary APIs."
