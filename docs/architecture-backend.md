# Architecture — Todo List API (Backend)

## Layered Architecture

```
cmd/server/main.go           Entry point, graceful shutdown
internal/
  handler/                    HTTP handlers (one file per domain)
  service/                    Business logic layer
  repository/                 Database access layer
  middleware/                 Gin middleware (auth, request ID, rate limit)
  model/                      Data structs and enums
  db/                         Database connection, migrations, test helpers
```

## Dependency Flow

```
handler → service → repository → database
handler ← middleware (auth, request ID)
```

- Handlers parse HTTP, call services, return JSON
- Services own business logic, call repositories
- Repositories own SQL, return domain models
- Middleware wraps handlers (JWT, request ID, rate limit)

## Request Lifecycle

```
HTTP Request
  → Request ID middleware (assign UUID)
  → JWT middleware (validate token, inject user_id)
  → Rate limit middleware (check window)
  → Handler (parse params, validate input)
  → Service (business rules)
  → Repository (SQL query)
  → Handler (serialize JSON response)
  → HTTP Response (with X-Request-ID header)
```

## Database Strategy

- **Prod**: PostgreSQL on Neon (serverless, scales to zero)
- **Test**: SQLite in-memory (zero config, fast)
- **Driver**: `database/sql` interface — both drivers implement it
- **Migrations**: Raw SQL on startup, IF NOT EXISTS for idempotency
