# ADR-0002: Todo Repository Pattern

- Status: Accepted
- Date: 2026-09-09

## Context

The todo-list API needs a data access layer for todos. The repository pattern provides a clean abstraction over database operations, enabling testability and separation of concerns.

## Decision

We will implement a TodoRepository struct with methods for CRUD operations, ownership enforcement, pagination, filtering, sorting, and cascading soft-delete.

### Key Design Choices

1. **Repository pattern over direct DB calls**
   - Clean separation of concerns
   - Easy to test with SQLite in-memory
   - Follows the pattern established in LAG-403 (auth-service)

2. **Ownership enforcement via user_id parameter**
   - Every query filters by user_id
   - Prevents cross-user data access
   - Simple and explicit

3. **Soft-delete via deleted_at timestamp**
   - Audit trail
   - Reversible
   - Consistent with existing schema

4. **Cascading soft-delete at repository level**
   - Ensures consistency when users are deleted
   - Single responsibility for user-related todo operations

5. **Priority sort always urgent → high → medium → low**
   - Business requirement
   - Order parameter ignored for priority sort

## Consequences

### Positive

- Testable: SQLite in-memory for fast tests
- Maintainable: clear separation between business logic and data access
- Consistent: follows established patterns from auth-service

### Negative

- Extra layer: slightly more code than direct DB calls
- SQLite differences: test behavior may differ from PostgreSQL

### Neutral

- Repository accepts validated input from handlers
- No HTTP handlers in this change (LAG-405)

## Alternatives Considered

1. **Direct SQL in handlers** — Rejected: tight coupling, hard to test
2. **ORM (GORM)** — Rejected: adds dependency, less control over queries
3. **Query builder** — Rejected: adds complexity for simple queries

## References

- LAG-404: Todo Repository ticket
- LAG-403: Auth Service (established repository pattern)
- LAG-401: Project Scaffolding (existing schema)
