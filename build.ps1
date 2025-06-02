# PowerShell Build Script for MathReX on Windows

# This script should be run from the 'backend/Go/Git/' directory.

Write-Host "Starting Windows build process..."

# Set environment variables for the build
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "1"

# Check if required directories exist
if (-not (Test-Path -Path "./libtokenizers/windows_amd64" -PathType Container)) {
    Write-Host "Creating libtokenizers directory..."
    New-Item -ItemType Directory -Path "./libtokenizers/windows_amd64" -Force | Out-Null
}

# Set CGO_LDFLAGS
# COMMON_LDFLAGS from Makefile: -L./libtokenizers/$(GOOS)_$(GOARCH)/
$env:CGO_LDFLAGS = "-L./libtokenizers/windows_amd64/"

# Output directory
$outputDir = "./bin"
$outputFile = "MathReX-windows-amd64.exe" # Adding .exe for Windows
$outputPath = Join-Path -Path $outputDir -ChildPath $outputFile

# Ensure the output directory exists
if (-not (Test-Path -Path $outputDir -PathType Container)) {
    New-Item -ItemType Directory -Path $outputDir | Out-Null
    Write-Host "Created output directory: $outputDir"
}

# Check if Go is available
Write-Host "Checking Go installation..."
try {
    $goVersion = go version
    Write-Host "Go version: $goVersion"
} catch {
    Write-Error "Go is not installed or not in PATH"
    exit 1
}

# Check for C compiler
Write-Host "Checking C compiler..."
$compilerFound = $false
try {
    # Try MSVC first
    $clVersion = cl.exe 2>&1 | Select-Object -First 1
    Write-Host "MSVC compiler found: $clVersion"
    $env:CC = "cl.exe"
    $compilerFound = $true
} catch {
    try {
        # Fallback to GCC
        $gccVersion = gcc --version | Select-Object -First 1
        Write-Host "GCC compiler found: $gccVersion"
        $env:CC = "gcc"
        $compilerFound = $true
    } catch {
        Write-Host "No suitable C compiler found (tried MSVC cl.exe and GCC)"
    }
}

if (-not $compilerFound) {
    Write-Error "No C compiler available. Please install Visual Studio Build Tools or MinGW."
    exit 1
}

# Go build command
Write-Host "Building for $env:GOOS/$env:GOARCH..."
Write-Host "Using CGO_LDFLAGS: $env:CGO_LDFLAGS"
Write-Host "CGO_ENABLED: $env:CGO_ENABLED"

# Show environment for debugging
Write-Host "Environment variables:"
Write-Host "  GOOS: $env:GOOS"
Write-Host "  GOARCH: $env:GOARCH"
Write-Host "  CGO_ENABLED: $env:CGO_ENABLED"
Write-Host "  CGO_LDFLAGS: $env:CGO_LDFLAGS"

go build -v -o $outputPath ./

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful! Output: $outputPath"
    Write-Host "File size: $((Get-Item $outputPath).Length) bytes"
} else {
    Write-Error "Build failed with exit code: $LASTEXITCODE"
    exit $LASTEXITCODE
}

# Clean up environment variables (optional, as they are set for the current session only)
# Remove-Item Env:GOOS
# Remove-Item Env:GOARCH
# Remove-Item Env:CGO_ENABLED
# Remove-Item Env:CGO_LDFLAGS
