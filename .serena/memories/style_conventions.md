# Style and Conventions

## Backend (Go)
- Follow standard Go idioms and naming conventions.
- Exported functions, structs, and methods should be CamelCase and documented.
- Use `New[StructName]` for constructor functions.
- Use `afero.Fs` abstraction instead of direct `os` calls to facilitate testing with in-memory filesystems.
- Use `context.Context` for long-running operations and Wails integration.

## Frontend (Svelte/TypeScript)
- Use TypeScript for all frontend logic.
- Follow Svelte best practices for component-based architecture.
- Styles should be placed in the `style.css` or scoped within components.

## Testing
- backend tests should use the standard `testing` package.
- Prefer table-driven tests for complex logic.
- Use `afero.NewMemMapFs()` for mocking file system operations in tests.
