# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Arktie is an open-source community platform for artists, built as a Go HTTP server.

- Module: `arktie.org`
- Go version: 1.26.1

## Build & Run Commands

```bash
make init          # Install dev tools (goimports, kessoku)
make gen           # Run go generate (regenerate DI wiring etc.)
make lint          # Format code (gofmt, go vet, goimports)
make test          # Run all tests
make build         # Build all binaries in cmd/ to bin/
make all           # gen + lint + test + build (default target)
make clean         # Remove bin/

# Run a single test
go test ./internal/data -run TestFunctionName

# Run the server directly
go run ./cmd/arktie -config configs/
```

## Architecture

### Dependency Injection with Kessoku

The project uses [kessoku](https://github.com/mazrean/kessoku) for compile-time dependency injection via code generation.

- DI definitions live in `cmd/arktie/kessoku.go` using `kessoku.Inject` and `kessoku.Provide`
- Running `go generate` produces `cmd/arktie/kessoku_band.go` (auto-generated, do not edit)
- When adding new dependencies to the server, register them in `kessoku.go` and re-run `go generate`

### HTTP Server

- Router: [go-chi/chi](https://github.com/go-chi/chi/v5) (`internal/server/handler.go`)
- The server runs with h2c (HTTP/2 cleartext) support
- JSON response helpers are in `internal/lib/libhttp/response.go`

### Configuration

- YAML config files in `configs/` are loaded by `gookit/config` (`internal/data/config.go`)
- Supports environment variable overrides (e.g., `APP_URL`, `APP_KEY`, `SERVER_ADDR`)
- App key (`app.key`) must be prefixed with `hex:` or `base64:` to indicate encoding format
- `Name` and `Version` are injected via `-ldflags` at build time, overriding config defaults

### ORM with Ent

- [ent](https://entgo.io/) is used for database schema, migrations, and queries
- Schema definitions live in `ent/schema/`
- Generated code is in `ent/` (auto-generated, do not edit except `ent/schema/`)
- Running `go generate ./ent` regenerates the ent client code
- Migrations are applied via `arktie-cli migrate` (uses `ent.Schema.Create` auto-migration)

### Project Layout

```
cmd/arktie/       # Application entrypoint and DI wiring
cmd/cli/          # CLI tools (migrate, etc.)
internal/data/    # Configuration, database client
internal/server/  # HTTP handler, routing, middleware
internal/service/ # Service layer (oauth, etc.)
internal/usecase/ # Business logic / use cases
internal/lib/     # Shared utilities (libhttp, libjwt, liblogs, liberrs)
ent/              # Ent ORM (generated code + schema/)
configs/          # YAML configuration files
```

## Custom Skills

This project includes custom Claude Code skills (in `.agents/skills/`):

- `go-style-guide` — invoke with `/go-style-guide` for Go style guidance (Google + Uber conventions)
- `go-project-layout` — invoke with `/go-project-layout` for standard Go project layout reference
