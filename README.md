# Chirpy — Twitter-like app server in Go

> Chirpy is a Twitter-like app server written in Go for posting short messages ("chirps"). This repository contains the server, SQL schema and queries (sqlc), generated DB bindings, utilities, and tests.

## Quick links
- Source: `cmd/main.go`
- Chirp handlers: `internal/handlers/chirps.go`
- Auth helpers: `internal/auth/auth.go`
- DB generated code: `internal/database`
- SQL migrations: `database/schema`

## Requirements
- Go (compatible with the repository `go.mod`)
- PostgreSQL (for runtime database)
- (Optional) sqlc if you want to regenerate DB bindings

## Setup (development)
1. Ensure PostgreSQL is running and reachable. Create a database for the app.
2. Set environment variables or a `.env` file with your DB connection and secrets (for example `DATABASE_URL`, `JWT_SECRET`).
3. Apply migrations found in `database/schema` (use your preferred tool; migrations are standard SQL files).

Build and run locally:

```sh
go build -o chirpy ./cmd
./chirpy
```

or

```sh
go run ./cmd/main.go
```

## Regenerate SQL bindings
This project uses sqlc configuration in `sqlc.yaml`. To regenerate the typed DB code after changing SQL files:

```sh
sqlc generate
```

## Tests
Unit tests live under the `test/` directory. Run them with:

```sh
go test ./...
```

## API docs
Detailed HTTP API documentation is in `./docs/API.md`.

## Contributing
- Keep code style consistent with existing files.
- Add unit tests for new behavior in `test/`.

