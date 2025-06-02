# Makefile

# Common LDFLAGS for all platforms
COMMON_LDFLAGS = -L./libtokenizers/$(GOOS)_$(GOARCH)/

# Platform-specific additions to LDFLAGS
LDFLAGS_ADD_darwin = -mmacosx-version-min=10.15 # Set a common minimum macOS version
LDFLAGS_ADD_windows = # No specific additions for Windows by default
LDFLAGS_ADD_linux =   # No specific additions for Linux by default.

# Determine final CGO_LDFLAGS based on GOOS
CGO_LDFLAGS_CONFIG := $(COMMON_LDFLAGS)
ifeq ($(GOOS),darwin)
	CGO_LDFLAGS_CONFIG += $(LDFLAGS_ADD_darwin)
endif
ifeq ($(GOOS),windows)
	CGO_LDFLAGS_CONFIG += $(LDFLAGS_ADD_windows)
endif
ifeq ($(GOOS),linux)
	CGO_LDFLAGS_CONFIG += $(LDFLAGS_ADD_linux)
endif

# Rule to print the calculated CGO_LDFLAGS
# This will be called by the GitHub Actions workflow
print-cgo-ldflags:
	@echo $(CGO_LDFLAGS_CONFIG)

# Main build target
# $(GOOS), $(GOARCH), CGO_ENABLED, and CGO_LDFLAGS are expected to be set in the environment
# by the calling process (e.g., GitHub Actions workflow).
build:
	@echo "Building for $(GOOS)/$(GOARCH)..."
	@echo "GOOS=$(GOOS), GOARCH=$(GOARCH)"
	@echo "CGO_ENABLED=$(CGO_ENABLED)"
	@echo "Using CGO_LDFLAGS from environment: $(CGO_LDFLAGS)"
	go build $(GO_BUILD_FLAGS) -o bin/MathReX-$(GOOS)-$(GOARCH) ./

# Build for all specified platforms
build-all:
	@echo "Starting build for all platforms..."
	GOOS=darwin GOARCH=arm64 $(MAKE) build
	GOOS=darwin GOARCH=amd64 $(MAKE) build
	# Build for Linux
	GOOS=linux GOARCH=amd64 $(MAKE) build
	GOOS=linux GOARCH=arm64 $(MAKE) build
	# Build for Windows
	GOOS=windows GOARCH=amd64 $(MAKE) build
	GOOS=windows GOARCH=arm64 $(MAKE) build
	@echo "All platform build process initiated."

# Declare phony targets to prevent conflicts with files named 'build' or 'build-all'
.PHONY: build build-all build-macos-app

# Target to build macOS .app bundle
# Assumes GOOS=darwin and GOARCH are set appropriately in the environment
# or that this target is called with them set, e.g.,
# GOOS=darwin GOARCH=arm64 $(MAKE) build-macos-app
build-macos-app: build
	@echo "Building macOS .app bundle for $(GOARCH)..."
	@rm -rf bin/MathReX.app # Clean up previous bundle
	@mkdir -p bin/MathReX.app/Contents/MacOS
	@mkdir -p bin/MathReX.app/Contents/Resources
	@cp bin/MathReX-$(GOOS)-$(GOARCH) bin/MathReX.app/Contents/MacOS/MathReX
	@cp Info.plist bin/MathReX.app/Contents/Info.plist
	# If you have an icon file (e.g., AppIcon.icns in the same directory as Makefile):
	# @cp AppIcon.icns bin/MathReX.app/Contents/Resources/AppIcon.icns
	@echo "MathReX.app bundle created in bin/ directory."
	@echo "Note: For distribution, the app may need to be code-signed and notarized."
