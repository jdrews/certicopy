# Project Overview: CertiCopy

## Purpose
CertiCopy is a cross-platform GUI desktop application designed for safely offloading files from media sources (like SD cards) with integrity verification using checksums.

## Tech Stack
- **Backend**: Go (Wails framework)
- **Frontend**: Svelte + TypeScript
- **Filesystem Abstraction**: `github.com/spf13/afero`
- **Hashing**: `github.com/cespare/xxhash/v2` for fast, non-cryptographic hashing.

## Code Structure
- `main.go`: Application entry point.
- `app.go`: Wails bridge, contains methods exposed to the frontend.
- `internal/core/`: Core business logic (copier, scanner, checksum, queue).
- `internal/models/`: Data structures and constants.
- `internal/services/`: High-level orchestration and Wails integration.
- `frontend/`: Svelte UI code.
- `pkg/`: Reusable utilities (currently minimal).
