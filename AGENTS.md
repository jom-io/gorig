# Repository Guidelines

## Project Structure & Module Organization
- `apix/` HTTP helpers and response utilities.
- `httpx/` HTTP server, middleware, and router registration.
- `domainx/` domain helpers (matching, migration).
- `cronx/` scheduled task utilities.
- `cache/` in-memory, SQLite, and Redis cache helpers.
- `global/` shared constants and variables.
- `utils/` configuration, logging, errors, and system helpers.
- `bootstrap/` app startup wiring.
- `serv/` service entry points; `simple/` runnable example (`main.go`).
- `test/` integration/unit tests (`*_test.go`).

## Build, Test, and Development Commands
- Run example: `go run ./simple`
- Build all: `go build ./...`
- Tests (verbose, race, coverage): `go test ./... -v -race -cover`
- Lint basics: `go vet ./...`
- Format: `go fmt ./...` (enforce before commits)

## Coding Style & Naming Conventions
- Follow Go defaults (`gofmt`); tabs; keep lines readable (<120 chars).
- Packages: short, lowercase (no underscores). Files: lowercase with underscores if needed.
- Exported identifiers: `PascalCase`; unexported: `lowerCamelCase`.
- Errors: prefer wrapping with context; use `utils/errors` where applicable. Log via `global`/`utils` helpers, not `fmt.Println`.
- HTTP: register routes via `httpx.RegisterRouter` and reuse middlewares in `httpx`.

## Testing Guidelines
- Place tests under `test/` or alongside packages as `*_test.go`.
- Use Go `testing` and `testify` (in `go.mod`) for assertions.
- Name tests `TestXxx`; keep tests independent and parallelizable when safe (`t.Parallel()`).
- New features require tests; include failure paths. Run `go test ./...` before pushing.

## Commit & Pull Request Guidelines
- Commit messages follow Conventional Commits: `feat:`, `fix:`, `refactor:`, `style:`, with optional scope (`feat(cache): ...`).
- PRs must include: clear description, linked issues, breaking-change notes, and test coverage for changes.
- CI expectations: code is formatted, `go vet` clean, and tests pass locally.

## Security & Configuration Tips
- Do not commit real secrets (e.g., tokens). Use environment variables and `viper`-style configs.
- Be cautious changing public APIs in `httpx`, `apix`, or `domainx`; document and mark breaking changes.
