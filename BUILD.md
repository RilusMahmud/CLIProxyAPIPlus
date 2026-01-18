# Build Instructions

Simple guide to building CLI Proxy API Plus using the Makefile.

## Quick Start

```bash
# Show all available commands
make help

# Build for your current platform
make build-macos   # or make build-linux
```

## Build Commands

### macOS Builds

```bash
make build-macos          # Build for current macOS architecture
make build-macos-amd64    # Build for Intel Macs
make build-macos-arm64    # Build for Apple Silicon (M1/M2/M3)
```

### Linux Builds

```bash
make build-linux          # Build for current Linux architecture
make build-linux-amd64    # Build for Linux AMD64
make build-linux-arm64    # Build for Linux ARM64
```

### Build All Platforms

```bash
make build-all            # Build for all platforms at once
```

## Other Commands

```bash
make clean                # Remove all build artifacts
make version              # Show version information
```

## Output Location

All binaries are built to the `build/` directory with this naming format:
```
build/cli-proxy-api-plus-{os}-{arch}
```

Examples:
- `build/cli-proxy-api-plus-darwin-arm64`
- `build/cli-proxy-api-plus-linux-amd64`

## Custom Version

Set custom version when building:

```bash
make build-macos VERSION=1.0.0
```

The build will automatically:
- Append `-plus` suffix to version
- Include git commit hash
- Include build timestamp
