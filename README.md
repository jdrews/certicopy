# CertiCopy
Move files with certainty; hashed and verified.

CertiCopy is a cross-platform GUI desktop application for safely offloading files from media sources (camera cards, SD cards, etc.) with integrity verification. It is built with Go (backend) and Svelte (frontend) using Wails, focusing on data safety over speed.

## Features

- **Cross-platform**: Runs on Linux and soon Windows/macOS.
- **Checksum Verification**: Verifies every byte copied using xxHash (default), BLAKE2b, SHA-256, etc.
- **Safety First**: Reads from source in read-only mode to prevent accidental modification.
- **Detailed Progress**: Tracks progress per file and overall transfer speed.
- **Resilient**: Retries failed files and provides a report of transfer status.

## Development

### Prerequisites

- Go 1.21+
- Node.js & npm
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Building

```bash
# Install dependencies
go mod download
cd frontend && npm install && cd ..

# Run in development mode
wails dev

# Build for production
wails build
```

## Architecture

- **Backend**: Go (Wails)
- **Frontend**: Svelte + TypeScript
- **Filesystem**: `spf13/afero` abstraction
- **Hashing**: `cespare/xxhash/v2`

## License

MIT License
