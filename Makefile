# Makefile

# Common LDFLAGS for all platforms
COMMON_LDFLAGS = -L./libtokenizers/$(GOOS)_$(GOARCH)/

# Platform-specific additions to LDFLAGS
LDFLAGS_ADD_darwin = -mmacosx-version-min=10.15 # Set a common minimum macOS version
LDFLAGS_ADD_windows = # No specific additions for Windows by default, MinGW handles most.
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

# Main build target
# $(GOOS) and $(GOARCH) are expected to be set by the calling rule (e.g., from build-all)
# or as environment variables.
build:
	@echo "Building for $(GOOS)/$(GOARCH)..."
	@echo "Using CGO_LDFLAGS: $(CGO_LDFLAGS_CONFIG)"
	CGO_ENABLED=1 CGO_LDFLAGS="$(CGO_LDFLAGS_CONFIG)" go build $(GO_BUILD_FLAGS) -o bin/MathReX-$(GOOS)-$(GOARCH) ./

# Build for all specified platforms
build-all:
	@echo "Starting build for all platforms..."
	GOOS=darwin GOARCH=arm64 $(MAKE) build
	GOOS=darwin GOARCH=amd64 $(MAKE) build
	# Uncomment to build for Linux
	# GOOS=linux GOARCH=amd64 $(MAKE) build
	# GOOS=linux GOARCH=arm64 $(MAKE) build
	GOOS=windows GOARCH=amd64 $(MAKE) build
	@echo "All platform build process initiated."

# Declare phony targets to prevent conflicts with files named 'build' or 'build-all'
.PHONY: build build-all
