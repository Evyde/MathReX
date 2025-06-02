#!/bin/bash

# Debug script to understand the -ldl linking issue

echo "=== Debug -ldl Linking Issue ==="

# Step 1: Check what the linker is actually looking for
echo "1. Analyzing linker behavior..."

echo "GCC search directories:"
gcc -print-search-dirs

echo "Library search paths:"
gcc -print-search-dirs | grep libraries | sed 's/libraries: =//' | tr ':' '\n'

# Step 2: Check what libraries exist in the search paths
echo "2. Checking existing libraries in search paths..."

search_paths=$(gcc -print-search-dirs | grep libraries | sed 's/libraries: =//' | tr ':' '\n')

echo "Looking for any dl-related libraries:"
for path in $search_paths; do
    if [ -d "$path" ]; then
        echo "In $path:"
        find "$path" -name "*dl*" 2>/dev/null || echo "  No dl libraries found"
    fi
done

# Step 3: Create our fake library and test it
echo "3. Creating and testing fake libdl..."

# Create fake libdl
cat > fake_dl.c << 'EOF'
void* dlopen(const char* filename, int flag) { return (void*)1; }
char* dlerror(void) { return 0; }
void* dlsym(void* handle, const char* symbol) { return 0; }
int dlclose(void* handle) { return 0; }
EOF

gcc -c fake_dl.c -o fake_dl.o
ar rcs libdl.a fake_dl.o

echo "Created libdl.a:"
ls -la libdl.a
ar -t libdl.a

# Test if we can link against it
echo "Testing direct linking:"
echo 'int main() { return 0; }' > test.c
gcc test.c -L. -ldl -o test.exe 2>&1 && {
    echo "✓ Direct linking works"
    rm -f test.exe
} || {
    echo "✗ Direct linking fails"
}
rm -f test.c

# Step 4: Try to understand what Go is doing
echo "4. Analyzing Go's linking behavior..."

# Create a minimal CGO program to see what Go does
cat > minimal_cgo.go << 'EOF'
package main

/*
#include <stdio.h>
void hello() {
    printf("Hello\n");
}
*/
import "C"

func main() {
    C.hello()
}
EOF

echo "Testing minimal CGO with verbose output:"
export CGO_ENABLED=1
export GOOS=windows
export GOARCH=amd64
export CGO_LDFLAGS="-L."

go build -x -v minimal_cgo.go 2>&1 | grep -E "(gcc|ld\.exe)" | head -10

rm -f minimal_cgo.go minimal_cgo.exe

# Step 5: Check if the issue is with library naming or location
echo "5. Testing different library locations and names..."

# Try copying to MinGW lib directory
mingw_lib="/c/mingw64/x86_64-w64-mingw32/lib"
if [ -d "$mingw_lib" ] && [ -w "$mingw_lib" ]; then
    echo "Copying libdl.a to MinGW lib directory..."
    cp libdl.a "$mingw_lib/"
    echo "✓ Copied to $mingw_lib"
else
    echo "Cannot write to MinGW lib directory: $mingw_lib"
fi

# Try different naming conventions
echo "Creating alternative library names:"
cp libdl.a dl.a
cp libdl.a libdl.lib 2>/dev/null || true

echo "All created libraries:"
ls -la *dl*

# Step 6: Final test with our libraries in place
echo "6. Final linking test..."

echo "Testing with current directory in library path:"
gcc -print-search-dirs | grep libraries
echo "Our libraries:"
ls -la libdl.a dl.a 2>/dev/null

# Test the exact command that's failing
echo "Simulating the failing link command..."
echo 'int main() { return 0; }' > final_test.c

# Simulate the command from the error log
gcc final_test.c -L. -ldl -o final_test.exe 2>&1 && {
    echo "✓ Final test successful"
    rm -f final_test.exe
} || {
    echo "✗ Final test failed - this confirms the issue"
    
    # Try with explicit library file
    gcc final_test.c libdl.a -o final_test.exe 2>&1 && {
        echo "✓ Explicit library file works"
        rm -f final_test.exe
    } || {
        echo "✗ Even explicit library file fails"
    }
}

rm -f final_test.c

# Cleanup
rm -f fake_dl.c fake_dl.o

echo "=== Debug Complete ==="
echo "Summary:"
echo "- Created libdl.a: $(ls libdl.a 2>/dev/null && echo 'YES' || echo 'NO')"
echo "- Library contains symbols: $(ar -t libdl.a 2>/dev/null | wc -l) objects"
echo "- Direct linking test: Run the script to see results"
