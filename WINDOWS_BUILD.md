# Windows Build Guide for MathReX

This document provides instructions for building MathReX on Windows, including troubleshooting common issues.

## Prerequisites

### Required Software

1. **Go 1.21 or later**
   - Download from: https://golang.org/dl/
   - Ensure `go` is in your PATH

2. **Rust toolchain**
   - Download from: https://rustup.rs/
   - Install the GNU target: `rustup target add x86_64-pc-windows-gnu`

3. **C Compiler (MinGW recommended for compatibility)**
   - **Option A: MinGW-w64 (Recommended)**
     - Download from: https://www.mingw-w64.org/downloads/
     - Or use TDM-GCC: https://jmeubank.github.io/tdm-gcc/
   - **Option B: Visual Studio Build Tools**
     - Download from: https://visualstudio.microsoft.com/downloads/
     - Install "C++ build tools" workload

4. **Git**
   - Download from: https://git-scm.com/download/win

### Environment Setup

1. Ensure all tools are in your PATH:
   ```cmd
   go version
   rustc --version
   gcc --version   # for MinGW (recommended)
   # OR
   cl.exe          # for Visual Studio
   git --version
   ```

2. Set environment variables for CGO:
   ```cmd
   set CGO_ENABLED=1
   ```

## Building

### Method 1: Using PowerShell Script (Recommended)

1. Open PowerShell as Administrator
2. Navigate to the project directory:
   ```powershell
   cd backend\Go\Git
   ```
3. Run the build script:
   ```powershell
   .\build.ps1
   ```

### Method 2: Using Makefile (with MinGW)

1. Open Command Prompt or PowerShell
2. Navigate to the project directory:
   ```cmd
   cd backend\Go\Git
   ```
3. Build using make:
   ```cmd
   make build
   ```

### Method 3: Manual Build

1. Set environment variables:
   ```cmd
   set GOOS=windows
   set GOARCH=amd64
   set CGO_ENABLED=1
   set CGO_LDFLAGS=-L./libtokenizers/windows_amd64/
   ```

2. Build the application:
   ```cmd
   go build -o bin\MathReX-windows-amd64.exe .\
   ```

## Testing Build Environment

Before attempting a full build, test your environment:

```powershell
.\test-windows-build.ps1
```

This script will verify:
- Go installation
- CGO support
- C compiler availability
- Required directories
- Go modules

## Troubleshooting

### Common Issues

#### 1. "gcc: command not found"
**Solution:** Install MinGW-w64 or TDM-GCC and add to PATH.

#### 2. "cgo: C compiler not found"
**Solutions:**
- Install a C compiler (MinGW recommended for compatibility)
- Set `CC` environment variable: `set CC=gcc` (for MinGW)
- Or for MSVC: `set CC=cl`

#### 3. "undefined reference to..." errors
**Solutions:**
- Ensure ONNX Runtime libraries are downloaded: `python download_onnxruntime.py`
- Check that `libtokenizers` library exists in `./libtokenizers/windows_amd64/`
- Verify CGO_LDFLAGS points to correct library paths

#### 4. "go: updates to go.mod needed"
**Solution:** Run `go mod tidy` to update dependencies.

#### 5. Rust compilation fails
**Solutions:**
- Install GNU target: `rustup target add x86_64-pc-windows-gnu`
- Update Rust: `rustup update`
- Check internet connection for dependency downloads

### Debug Build

For more verbose output during build:

```cmd
go build -v -x -o bin\MathReX-windows-amd64.exe .\
```

### Clean Build

To start fresh:

```cmd
go clean -cache -modcache
go mod download
```

## GitHub Actions

The project includes automated Windows builds via GitHub Actions. The workflow:

1. Sets up Go, Rust, and MinGW
2. Downloads ONNX Runtime libraries
3. Builds tokenizers library from source
4. Compiles the main application
5. Creates release artifacts

To trigger a release build, push a tag starting with 'v':

```bash
git tag v1.0.0
git push origin v1.0.0
```

## Dependencies

### Runtime Dependencies
- ONNX Runtime (automatically downloaded)
- Tokenizers library (built from source)

### Build Dependencies
- Go modules (see go.mod)
- Rust crates (for tokenizers)

## Support

If you encounter issues not covered here:

1. Check the GitHub Actions logs for similar errors
2. Verify all prerequisites are correctly installed
3. Run the test script to identify environment issues
4. Create an issue on GitHub with:
   - Your Windows version
   - Go version (`go version`)
   - Rust version (`rustc --version`)
   - Complete error output
   - Output from `test-windows-build.ps1`
