# Test script for Windows build environment
# This script tests the Windows build environment without actually building

Write-Host "=== Windows Build Environment Test ==="

# Test 1: Check Go installation
Write-Host "`n1. Testing Go installation..."
try {
    $goVersion = go version
    Write-Host "✓ Go is installed: $goVersion"
} catch {
    Write-Host "✗ Go is not installed or not in PATH"
    exit 1
}

# Test 2: Check CGO support
Write-Host "`n2. Testing CGO support..."
$env:CGO_ENABLED = "1"
try {
    $cgoSupport = go env CGO_ENABLED
    if ($cgoSupport -eq "1") {
        Write-Host "✓ CGO is enabled"
    } else {
        Write-Host "✗ CGO is disabled"
    }
} catch {
    Write-Host "✗ Failed to check CGO status"
}

# Test 3: Check C compiler
Write-Host "`n3. Testing C compiler..."
$compilerFound = $false
try {
    $clVersion = cl.exe 2>&1 | Select-Object -First 1
    Write-Host "✓ MSVC is available: $clVersion"
    $compilerFound = $true
} catch {
    Write-Host "✗ MSVC (cl.exe) is not available"
}

try {
    $gccVersion = gcc --version | Select-Object -First 1
    Write-Host "✓ GCC is available: $gccVersion"
    $compilerFound = $true
} catch {
    Write-Host "✗ GCC is not available in PATH"
}

if (-not $compilerFound) {
    Write-Host "  You need to install either:"
    Write-Host "  - Visual Studio Build Tools (recommended)"
    Write-Host "  - MinGW or TDM-GCC"
}

# Test 4: Check required directories
Write-Host "`n4. Testing directory structure..."
$requiredDirs = @(
    "./libtokenizers/windows_amd64",
    "./onnxruntime",
    "./model_controller"
)

foreach ($dir in $requiredDirs) {
    if (Test-Path -Path $dir -PathType Container) {
        Write-Host "✓ Directory exists: $dir"
    } else {
        Write-Host "✗ Directory missing: $dir"
    }
}

# Test 5: Check Go modules
Write-Host "`n5. Testing Go modules..."
try {
    go mod verify
    Write-Host "✓ Go modules are valid"
} catch {
    Write-Host "✗ Go modules verification failed"
    Write-Host "  Try running: go mod tidy"
}

# Test 6: Test simple Go build (without CGO)
Write-Host "`n6. Testing simple Go compilation..."
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"

try {
    go build -o test-simple.exe ./
    if (Test-Path -Path "./test-simple.exe") {
        Write-Host "✓ Simple Go build successful"
        Remove-Item "./test-simple.exe" -Force
    } else {
        Write-Host "✗ Simple Go build failed"
    }
} catch {
    Write-Host "✗ Simple Go build failed with error"
}

Write-Host "`n=== Test Complete ==="
Write-Host "If all tests pass, the Windows build environment should work."
Write-Host "If any tests fail, please address the issues before attempting to build."
