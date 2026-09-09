package repository

import (
	"database/sql"
	"testing"
	"time"

	"todo-list-api/internal/model"

	_ "github.com/mattn/go-sqlite3"
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

func TestCreate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close() //nolint:errcheck

	repo := NewTodoRepository(db)

	// Create a test user first
	_, err := db.Exec("INSERT INTO users (id, name, email, password_hash) VALUES (?, ?, ?, ?)",
		"user-1", "Test User", "test@example.com", "hash")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

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

	// Create test user
	_, err := db.Exec("INSERT INTO users (id, name, email, password_hash) VALUES (?, ?, ?, ?)",
		"user-1", "Test User", "test@example.com", "hash")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Create test todo
	todo := &model.Todo{
		Title:    "Test Todo",
		Status:   model.TodoStatusPending,
		Priority: model.TodoPriorityMedium,
	}
	created, err := repo.Create("user-1", todo)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		found, err := repo.FindByID("user-1", created.ID)
		if err != nil {
			t.Fatalf("FindByID failed: %v", err)
		}
		if found == nil {
			t.Fatal("expected todo to be found")
		}
		if found.ID != created.ID {
			t.Errorf("expected ID %q, got %q", created.ID, found.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		found, err := repo.FindByID("user-1", "non-existent")
		if err != nil {
			t.Fatalf("FindByID failed: %v", err)
		}
		if found != nil {
			t.Error("expected nil for non-existent todo")
		}
	})

	t.Run("soft-deleted", func(t *testing.T) {
		// Soft delete the todo
		_, err := db.Exec("UPDATE todos SET deleted_at = ? WHERE id = ?",
			time.Now(), created.ID)
		if err != nil {
			t.Fatalf("failed to soft delete: %v", err)
		}

		found, err := repo.FindByID("user-1", created.ID)
		if err != nil {
			t.Fatalf("FindByID failed: %v", err)
		}
		if found != nil {
			t.Error("expected nil for soft-deleted todo")
		}
	})

	t.Run("different user", func(t *testing.T) {
		// Create another user
		_, err := db.Exec("INSERT INTO users (id, name, email, password_hash) VALUES (?, ?, ?, ?)",
			"user-2", "Other User", "other@example.com", "hash")
		if err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		// Reset deleted_at for the todo
		_, err = db.Exec("UPDATE todos SET deleted_at = NULL WHERE id = ?", created.ID)
		if err != nil {
			t.Fatalf("failed to reset deleted_at: %v", err)
		}

		found, err := repo.FindByID("user-2", created.ID)
		if err != nil {
			t.Fatalf("FindByID failed: %v", err)
		}
		if found != nil {
			t.Error("expected nil for todo belonging to different user")
		}
	})
}

func TestUpdate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close() //nolint:errcheck

	repo := NewTodoRepository(db)

	// Create test user
	_, err := db.Exec("INSERT INTO users (id, name, email, password_hash) VALUES (?, ?, ?, ?)",
		"user-1", "Test User", "test@example.com", "hash")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Create test todo
	todo := &model.Todo{
		Title:    "Test Todo",
		Status:   model.TodoStatusPending,
		Priority: model.TodoPriorityMedium,
	}
	created, err := repo.Create("user-1", todo)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

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

	// Create test user
	_, err := db.Exec("INSERT INTO users (id, name, email, password_hash) VALUES (?, ?, ?, ?)",
		"user-1", "Test User", "test@example.com", "hash")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Create test todo
	todo := &model.Todo{
		Title:    "Test Todo",
		Status:   model.TodoStatusPending,
		Priority: model.TodoPriorityMedium,
	}
	created, err := repo.Create("user-1", todo)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

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

	// Create test user
	_, err := db.Exec("INSERT INTO users (id, name, email, password_hash) VALUES (?, ?, ?, ?)",
		"user-1", "Test User", "test@example.com", "hash")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Create 5 test todos
	for i := 0; i < 5; i++ {
		todo := &model.Todo{
			Title:    "Todo " + string(rune('A'+i)),
			Status:   model.TodoStatusPending,
			Priority: model.TodoPriorityMedium,
		}
		_, err := repo.Create("user-1", todo)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	t.Run("default pagination", func(t *testing.T) {
		result, err := repo.List("user-1", ListParams{})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if result.Page != 1 {
			t.Errorf("expected Page 1, got %d", result.Page)
		}
		if result.Limit != 20 {
			t.Errorf("expected Limit 20, got %d", result.Limit)
		}
		if result.TotalCount != 5 {
			t.Errorf("expected TotalCount 5, got %d", result.TotalCount)
		}
		if len(result.Todos) != 5 {
			t.Errorf("expected 5 todos, got %d", len(result.Todos))
		}
	})

	t.Run("custom pagination", func(t *testing.T) {
		result, err := repo.List("user-1", ListParams{Page: 2, Limit: 2})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if result.Page != 2 {
			t.Errorf("expected Page 2, got %d", result.Page)
		}
		if len(result.Todos) != 2 {
			t.Errorf("expected 2 todos, got %d", len(result.Todos))
		}
	})

	t.Run("limit cap", func(t *testing.T) {
		result, err := repo.List("user-1", ListParams{Limit: 200})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if result.Limit != 100 {
			t.Errorf("expected Limit 100, got %d", result.Limit)
		}
	})

	t.Run("page beyond total", func(t *testing.T) {
		result, err := repo.List("user-1", ListParams{Page: 100, Limit: 20})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(result.Todos) != 0 {
			t.Errorf("expected 0 todos, got %d", len(result.Todos))
		}
		if result.TotalPages != 1 {
			t.Errorf("expected TotalPages 1, got %d", result.TotalPages)
		}
	})
}

func TestFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close() //nolint:errcheck

	repo := NewTodoRepository(db)

	// Create test user
	_, err := db.Exec("INSERT INTO users (id, name, email, password_hash) VALUES (?, ?, ?, ?)",
		"user-1", "Test User", "test@example.com", "hash")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

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

	for _, todo := range todos {
		_, err := repo.Create("user-1", &model.Todo{
			Title:    todo.title,
			Status:   todo.status,
			Priority: todo.priority,
		})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	t.Run("filter by status", func(t *testing.T) {
		result, err := repo.List("user-1", ListParams{
			Status: []string{"pending"},
		})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if result.TotalCount != 3 {
			t.Errorf("expected 3 pending todos, got %d", result.TotalCount)
		}
	})

	t.Run("filter by priority", func(t *testing.T) {
		result, err := repo.List("user-1", ListParams{
			Priority: []string{"urgent", "high"},
		})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if result.TotalCount != 2 {
			t.Errorf("expected 2 urgent/high todos, got %d", result.TotalCount)
		}
	})

	t.Run("filter by title", func(t *testing.T) {
		result, err := repo.List("user-1", ListParams{
			Title: "milk",
		})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if result.TotalCount != 1 {
			t.Errorf("expected 1 todo with 'milk' in title, got %d", result.TotalCount)
		}
	})

	t.Run("combined filters", func(t *testing.T) {
		result, err := repo.List("user-1", ListParams{
			Status:   []string{"pending"},
			Priority: []string{"high"},
			Title:    "eggs",
		})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if result.TotalCount != 1 {
			t.Errorf("expected 1 todo matching all filters, got %d", result.TotalCount)
		}
	})
}

func TestSort(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close() //nolint:errcheck

	repo := NewTodoRepository(db)

	// Create test user
	_, err := db.Exec("INSERT INTO users (id, name, email, password_hash) VALUES (?, ?, ?, ?)",
		"user-1", "Test User", "test@example.com", "hash")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Create test todos with different priorities
	priorities := []model.TodoPriority{
		model.TodoPriorityLow,
		model.TodoPriorityUrgent,
		model.TodoPriorityMedium,
		model.TodoPriorityHigh,
	}

	for i, p := range priorities {
		todo := &model.Todo{
			Title:    "Todo " + string(rune('A'+i)),
			Status:   model.TodoStatusPending,
			Priority: p,
		}
		_, err := repo.Create("user-1", todo)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
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

	// Create test user
	_, err := db.Exec("INSERT INTO users (id, name, email, password_hash) VALUES (?, ?, ?, ?)",
		"user-1", "Test User", "test@example.com", "hash")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Create 3 test todos
	for i := 0; i < 3; i++ {
		todo := &model.Todo{
			Title:    "Todo " + string(rune('A'+i)),
			Status:   model.TodoStatusPending,
			Priority: model.TodoPriorityMedium,
		}
		_, err := repo.Create("user-1", todo)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Delete all todos for the user
	err = repo.DeleteByUserID("user-1")
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
