# Stack — Todo List API (Backend)

| Category | Technology | Version | Purpose |
|----------|------------|---------|---------|
| Runtime | Go | 1.22+ | Backend language |
| Framework | Gin | v1.9+ | HTTP router, middleware |
| Database (Prod) | PostgreSQL | 15+ | Persistent storage |
| DB Host (Prod) | Neon | — | Serverless PostgreSQL |
| Database (Test) | SQLite | 3.x | In-memory test DB |
| DB Driver | pgx | v5 | PostgreSQL driver |
| DB Driver | go-sqlite3 | — | SQLite driver |
| Auth Tokens | JWT (HS256) | v5 | Stateless access tokens |
| Auth Library | golang-jwt/jwt | v5 | JWT creation/validation |
| UUIDs | google/uuid | v6 | Primary keys |
| Testing | stdlib testing | — | Unit/integration tests |
| Test HTTP | net/http/httptest | — | Handler testing |
| Linting | golangci-lint | — | Static analysis |
