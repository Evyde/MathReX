# PowerShell Build Script for MathReX on Windows

# This script should be run from the 'backend/Go/Git/' directory.

# Set environment variables for the build
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "1"

# Set CGO_LDFLAGS
# COMMON_LDFLAGS from Makefile: -L./libtokenizers/$(GOOS)_$(GOARCH)/
# LDFLAGS_ADD_windows is empty
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

# Go build command
# GO_BUILD_FLAGS is not defined in the Makefile for the general build,
# so it's omitted here. Add them if needed.
Write-Host "Building for $env:GOOS/$env:GOARCH..."
Write-Host "Using CGO_LDFLAGS: $env:CGO_LDFLAGS"

go build -o $outputPath ./

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful! Output: $outputPath"
} else {
    Write-Error "Build failed!"
}

# Clean up environment variables (optional, as they are set for the current session only)
# Remove-Item Env:GOOS
# Remove-Item Env:GOARCH
# Remove-Item Env:CGO_ENABLED
# Remove-Item Env:CGO_LDFLAGS
