# CertiCopy
Move files with certainty; hashed and verified.

CertiCopy is a Linux GUI desktop application for safely offloading files from media sources (camera disks, SD cards, etc.) with integrity verification. Made with experience acting as a DIT on film sets where data safety is of the utmost importance. Built with Go (backend) and Svelte (frontend) using Wails, focusing on data safety over speed.

## Features

- **Linux first**: Built on Linux, tested on Linux. Plans to support Windows/macOS in the future.
- **Checksum Verification**: Verifies every byte copied. Supports **xxHash** (default), **BLAKE2b**, **SHA-256**, **SHA-1**, and **MD5**.
- **Post-transfer Integrity**: Optional full verification check after all files are copied.
- **Safety First**: Reads from source in read-only mode to prevent accidental modification.
- **Detailed Progress**: Tracks progress per file and overall transfer speed.
- **Resilient**: Retries failed files and provides a report of transfer status.

## Development

### Prerequisites

- Go 1.24+
- Node.js & npm
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Linux Packages: `libgtk-3-dev`, `libwebkit2gtk-4.0-dev`, `pkg-config`, `build-essential`

### Building

```bash
# Verify setup
wails doctor

# Install dependencies
go mod download
cd frontend && npm install && cd ..

# Run in development mode
wails dev

# Build for production (output in build/bin/)
wails build
```

## Architecture

- **Backend**: Go (Wails v2)
- **Frontend**: Svelte 5 + TypeScript
- **Filesystem**: `spf13/afero` abstraction

## License

MIT License
