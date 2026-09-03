# ADR-0001: Go Project Directory Structure

- Status: Accepted
- Date: 2026-09-02

## Context

The project needs a directory structure that follows Go conventions, supports clear layer separation, and prevents circular dependencies.

## Decision

Use the standard Go project layout:

```
cmd/server/           — Entry point
internal/handler/     — HTTP handlers
internal/service/     — Business logic
internal/repository/  — Database access
internal/middleware/   — Gin middleware
internal/model/       — Data structs
internal/db/          — Database connection, migrations
```

## Consequences

- `internal/` prevents external packages from importing project code
- Each layer depends only on the layer below it: handler → service → repository
- Easy to find code by responsibility
- Standard Go convention familiar to contributors

## Alternatives Considered

- **Flat structure** (`handlers/`, `services/`): Simpler but no import protection
- **Domain-based** (`auth/`, `todos/`): premature for a small API, harder to share infra code
