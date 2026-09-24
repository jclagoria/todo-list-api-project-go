package repository

import (
	"database/sql"
	"testing"
	"time"

	"todo-list-api/internal/model"

	_ "github.com/mattn/go-sqlite3" // Register SQLite driver for database/sql
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// Run migrations
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id            TEXT PRIMARY KEY,
			name          VARCHAR(255) NOT NULL,
			email         VARCHAR(255) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at    TIMESTAMP NULL
		);
		CREATE TABLE IF NOT EXISTS todos (
			id          TEXT PRIMARY KEY,
			user_id     TEXT NOT NULL REFERENCES users(id),
			title       VARCHAR(255) NOT NULL,
			description TEXT,
			status      VARCHAR(20) NOT NULL DEFAULT 'pending',
			priority    VARCHAR(20) NOT NULL DEFAULT 'medium',
			due_date    DATE,
			created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at  TIMESTAMP NULL
		);
	`); err != nil {
		_ = db.Close()
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

// ponytail: helpers for repeated test setup
func seedUser(t *testing.T, db *sql.DB, id, name, email string) {
	t.Helper()
	_, err := db.Exec("INSERT INTO users (id, name, email, password_hash) VALUES (?, ?, ?, ?)",
		id, name, email, "hash")
	if err != nil {
		t.Fatalf("failed to create user %s: %v", id, err)
	}
}

func seedTodo(t *testing.T, repo *TodoRepository, userID, title string, status model.TodoStatus, priority model.TodoPriority) *model.Todo {
	t.Helper()
	created, err := repo.Create(userID, &model.Todo{Title: title, Status: status, Priority: priority})
	if err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}
	return created
}

func TestCreate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close() //nolint:errcheck

	repo := NewTodoRepository(db)
	seedUser(t, db, "user-1", "Test User", "test@example.com")

	todo := &model.Todo{
		Title:       "Test Todo",
		Description: sql.NullString{String: "Test description", Valid: true},
		Status:      model.TodoStatusPending,
		Priority:    model.TodoPriorityMedium,
		DueDate:     sql.NullTime{Time: time.Now().Add(24 * time.Hour), Valid: true},
	}

	created, err := repo.Create("user-1", todo)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if created.ID == "" {
		t.Error("expected non-empty ID")
	}
	if created.UserID != "user-1" {
		t.Errorf("expected UserID 'user-1', got %q", created.UserID)
	}
	if created.Title != "Test Todo" {
		t.Errorf("expected Title 'Test Todo', got %q", created.Title)
	}
	if created.Status != model.TodoStatusPending {
		t.Errorf("expected Status 'pending', got %q", created.Status)
	}
	if created.Priority != model.TodoPriorityMedium {
		t.Errorf("expected Priority 'medium', got %q", created.Priority)
	}
	if created.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
	if created.UpdatedAt.IsZero() {
		t.Error("expected non-zero UpdatedAt")
	}
}

func TestFindByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close() //nolint:errcheck

	repo := NewTodoRepository(db)
	seedUser(t, db, "user-1", "Test User", "test@example.com")

	created := seedTodo(t, repo, "user-1", "Test Todo", model.TodoStatusPending, model.TodoPriorityMedium)

	t.Run("found", func(t *testing.T) {
		found := mustFindByID(t, repo, "user-1", created.ID)
		if found == nil {
			t.Fatal("expected todo to be found")
		}
		if found.ID != created.ID {
			t.Errorf("expected ID %q, got %q", created.ID, found.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		if found := mustFindByID(t, repo, "user-1", "non-existent"); found != nil {
			t.Error("expected nil for non-existent todo")
		}
	})

	t.Run("soft-deleted", func(t *testing.T) {
		softDeleteTodo(t, db, created.ID)

		if found := mustFindByID(t, repo, "user-1", created.ID); found != nil {
			t.Error("expected nil for soft-deleted todo")
		}
	})

	t.Run("different user", func(t *testing.T) {
		seedUser(t, db, "user-2", "Other User", "other@example.com")
		restoreTodo(t, db, created.ID)

		if found := mustFindByID(t, repo, "user-2", created.ID); found != nil {
			t.Error("expected nil for todo belonging to different user")
		}
	})
}

// ponytail: extracted to keep TestFindByID under the cognitive complexity limit
func mustFindByID(t *testing.T, repo *TodoRepository, userID, todoID string) *model.Todo {
	t.Helper()
	found, err := repo.FindByID(userID, todoID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	return found
}

func softDeleteTodo(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	if _, err := db.Exec("UPDATE todos SET deleted_at = ? WHERE id = ?", time.Now(), id); err != nil {
		t.Fatalf("failed to soft delete: %v", err)
	}
}

func restoreTodo(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	if _, err := db.Exec("UPDATE todos SET deleted_at = NULL WHERE id = ?", id); err != nil {
		t.Fatalf("failed to reset deleted_at: %v", err)
	}
}

func TestUpdate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close() //nolint:errcheck

	repo := NewTodoRepository(db)
	seedUser(t, db, "user-1", "Test User", "test@example.com")

	created := seedTodo(t, repo, "user-1", "Test Todo", model.TodoStatusPending, model.TodoPriorityMedium)

	t.Run("success", func(t *testing.T) {
		updated := &model.Todo{
			Title:    "Updated Todo",
			Status:   model.TodoStatusInProgress,
			Priority: model.TodoPriorityHigh,
			DueDate:  created.DueDate,
		}
		result, err := repo.Update("user-1", created.ID, updated)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if result.Title != "Updated Todo" {
			t.Errorf("expected Title 'Updated Todo', got %q", result.Title)
		}
		if result.Status != model.TodoStatusInProgress {
			t.Errorf("expected Status 'in_progress', got %q", result.Status)
		}
		if result.Priority != model.TodoPriorityHigh {
			t.Errorf("expected Priority 'high', got %q", result.Priority)
		}
	})

	t.Run("not found", func(t *testing.T) {
		updated := &model.Todo{
			Title:    "Updated",
			Status:   model.TodoStatusPending,
			Priority: model.TodoPriorityMedium,
		}
		_, err := repo.Update("user-1", "non-existent", updated)
		if err != ErrTodoNotFound {
			t.Errorf("expected ErrTodoNotFound, got %v", err)
		}
	})

	t.Run("not owner", func(t *testing.T) {
		// Create another user
		_, err := db.Exec("INSERT INTO users (id, name, email, password_hash) VALUES (?, ?, ?, ?)",
			"user-2", "Other User", "other@example.com", "hash")
		if err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		updated := &model.Todo{
			Title:    "Updated",
			Status:   model.TodoStatusPending,
			Priority: model.TodoPriorityMedium,
		}
		_, err = repo.Update("user-2", created.ID, updated)
		if err != ErrTodoNotFound {
			t.Errorf("expected ErrTodoNotFound, got %v", err)
		}
	})
}

func TestDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close() //nolint:errcheck

	repo := NewTodoRepository(db)
	seedUser(t, db, "user-1", "Test User", "test@example.com")

	created := seedTodo(t, repo, "user-1", "Test Todo", model.TodoStatusPending, model.TodoPriorityMedium)

	t.Run("success", func(t *testing.T) {
		err := repo.Delete("user-1", created.ID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify it's soft deleted
		found, err := repo.FindByID("user-1", created.ID)
		if err != nil {
			t.Fatalf("FindByID failed: %v", err)
		}
		if found != nil {
			t.Error("expected todo to be soft deleted")
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := repo.Delete("user-1", "non-existent")
		if err != ErrTodoNotFound {
			t.Errorf("expected ErrTodoNotFound, got %v", err)
		}
	})

	t.Run("not owner", func(t *testing.T) {
		// Create another user
		_, err := db.Exec("INSERT INTO users (id, name, email, password_hash) VALUES (?, ?, ?, ?)",
			"user-2", "Other User", "other@example.com", "hash")
		if err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		err = repo.Delete("user-2", created.ID)
		if err != ErrTodoNotFound {
			t.Errorf("expected ErrTodoNotFound, got %v", err)
		}
	})
}

func TestList(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close() //nolint:errcheck

	repo := NewTodoRepository(db)
	seedUser(t, db, "user-1", "Test User", "test@example.com")

	// Create 5 test todos
	for i := 0; i < 5; i++ {
		seedTodo(t, repo, "user-1", "Todo "+string(rune('A'+i)),
			model.TodoStatusPending, model.TodoPriorityMedium)
	}

	tests := []listTestCase{
		{"default pagination", ListParams{}, 1, 20, 5, 5, 1},
		{"custom pagination", ListParams{Page: 2, Limit: 2}, 2, 2, 5, 2, 3},
		{"limit cap", ListParams{Limit: 200}, 1, 100, 5, 5, 1},
		{"page beyond total", ListParams{Page: 100, Limit: 20}, 100, 20, 5, 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertListResult(t, repo, tt)
		})
	}
}

// ponytail: assertions extracted so TestList stays under the cognitive complexity limit
type listTestCase struct {
	name           string
	params         ListParams
	wantPage       int
	wantLimit      int
	wantTotalCount int
	wantTodos      int
	wantTotalPages int
}

func assertListResult(t *testing.T, repo *TodoRepository, tt listTestCase) {
	t.Helper()
	result := mustList(t, repo, tt.params)
	if result.Page != tt.wantPage {
		t.Errorf("expected Page %d, got %d", tt.wantPage, result.Page)
	}
	if result.Limit != tt.wantLimit {
		t.Errorf("expected Limit %d, got %d", tt.wantLimit, result.Limit)
	}
	if result.TotalCount != tt.wantTotalCount {
		t.Errorf("expected TotalCount %d, got %d", tt.wantTotalCount, result.TotalCount)
	}
	if len(result.Todos) != tt.wantTodos {
		t.Errorf("expected %d todos, got %d", tt.wantTodos, len(result.Todos))
	}
	if result.TotalPages != tt.wantTotalPages {
		t.Errorf("expected TotalPages %d, got %d", tt.wantTotalPages, result.TotalPages)
	}
}

func TestFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close() //nolint:errcheck

	repo := NewTodoRepository(db)
	seedUser(t, db, "user-1", "Test User", "test@example.com")

	// Create test todos with different statuses and priorities
	todos := []struct {
		title    string
		status   model.TodoStatus
		priority model.TodoPriority
	}{
		{"Buy milk", model.TodoStatusPending, model.TodoPriorityLow},
		{"Buy eggs", model.TodoStatusPending, model.TodoPriorityHigh},
		{"Buy bread", model.TodoStatusInProgress, model.TodoPriorityMedium},
		{"Buy butter", model.TodoStatusDone, model.TodoPriorityUrgent},
		{"Buy cheese", model.TodoStatusPending, model.TodoPriorityMedium},
	}

	for _, td := range todos {
		seedTodo(t, repo, "user-1", td.title, td.status, td.priority)
	}

	t.Run("filter by status", func(t *testing.T) {
		result := mustList(t, repo, ListParams{Status: []string{"pending"}})
		if result.TotalCount != 3 {
			t.Errorf("expected 3 pending todos, got %d", result.TotalCount)
		}
	})

	t.Run("filter by priority", func(t *testing.T) {
		result := mustList(t, repo, ListParams{Priority: []string{"urgent", "high"}})
		if result.TotalCount != 2 {
			t.Errorf("expected 2 urgent/high todos, got %d", result.TotalCount)
		}
	})

	t.Run("filter by title", func(t *testing.T) {
		result := mustList(t, repo, ListParams{Title: "milk"})
		if result.TotalCount != 1 {
			t.Errorf("expected 1 todo with 'milk' in title, got %d", result.TotalCount)
		}
	})

	t.Run("combined filters", func(t *testing.T) {
		result := mustList(t, repo, ListParams{
			Status:   []string{"pending"},
			Priority: []string{"high"},
			Title:    "eggs",
		})
		if result.TotalCount != 1 {
			t.Errorf("expected 1 todo matching all filters, got %d", result.TotalCount)
		}
	})
}

// ponytail: extracted so TestFilter stays under the cognitive complexity limit
func mustList(t *testing.T, repo *TodoRepository, params ListParams) *ListResult {
	t.Helper()
	result, err := repo.List("user-1", params)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	return result
}

func TestSort(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close() //nolint:errcheck

	repo := NewTodoRepository(db)
	seedUser(t, db, "user-1", "Test User", "test@example.com")

	// Create test todos with different priorities
	priorities := []model.TodoPriority{
		model.TodoPriorityLow,
		model.TodoPriorityUrgent,
		model.TodoPriorityMedium,
		model.TodoPriorityHigh,
	}

	for i, p := range priorities {
		seedTodo(t, repo, "user-1", "Todo "+string(rune('A'+i)),
			model.TodoStatusPending, p)
	}

	t.Run("sort by priority", func(t *testing.T) {
		result, err := repo.List("user-1", ListParams{
			Sort: "priority",
		})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}

		expectedOrder := []model.TodoPriority{
			model.TodoPriorityUrgent,
			model.TodoPriorityHigh,
			model.TodoPriorityMedium,
			model.TodoPriorityLow,
		}

		for i, todo := range result.Todos {
			if todo.Priority != expectedOrder[i] {
				t.Errorf("expected priority %q at position %d, got %q", expectedOrder[i], i, todo.Priority)
			}
		}
	})

	t.Run("sort by created_at desc", func(t *testing.T) {
		result, err := repo.List("user-1", ListParams{
			Sort:  "created_at",
			Order: "desc",
		})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}

		for i := 1; i < len(result.Todos); i++ {
			if result.Todos[i].CreatedAt.After(result.Todos[i-1].CreatedAt) {
				t.Errorf("expected created_at to be descending")
			}
		}
	})
}

func TestDeleteByUserID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close() //nolint:errcheck

	repo := NewTodoRepository(db)
	seedUser(t, db, "user-1", "Test User", "test@example.com")

	// Create 3 test todos
	for i := 0; i < 3; i++ {
		seedTodo(t, repo, "user-1", "Todo "+string(rune('A'+i)),
			model.TodoStatusPending, model.TodoPriorityMedium)
	}

	// Delete all todos for the user
	err := repo.DeleteByUserID("user-1")
	if err != nil {
		t.Fatalf("DeleteByUserID failed: %v", err)
	}

	// Verify all todos are soft deleted
	result, err := repo.List("user-1", ListParams{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if result.TotalCount != 0 {
		t.Errorf("expected 0 todos after DeleteByUserID, got %d", result.TotalCount)
	}
}
