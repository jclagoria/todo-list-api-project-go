# Deployment — Todo List API

## Local Development

```bash
# Start PostgreSQL (Docker)
docker run -d --name todo-db -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:15

# Set env vars
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/todo_list?sslmode=disable
export JWT_SECRET=dev-secret-change-in-production

# Run
go run ./cmd/server
```

## Testing

```bash
# Unit tests (SQLite in-memory, no external deps)
go test ./...

# With coverage
go test -cover ./...

# Race detector
go test -race ./...
```

## Build

```bash
# Binary
go build -o bin/todo-api ./cmd/server

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -o bin/todo-api-linux ./cmd/server
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `JWT_SECRET` | Yes | — | HS256 signing key |
| `PORT` | No | `8080` | Server listen port |
| `GIN_MODE` | No | `debug` | `debug` or `release` |

## Production (Neon)

- Neon handles PostgreSQL hosting, connection pooling, branching
- Connect via `DATABASE_URL` from Neon dashboard
- SSL mode required: `sslmode=require`
- Connection pooling via PgBouncer (built into Neon)
