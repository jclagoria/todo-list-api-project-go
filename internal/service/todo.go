package service

import (
	"strconv"
	"strings"

	"todo-list-api/internal/model"
	"todo-list-api/internal/repository"
)

// Repository is the persistence contract this service needs. It is defined
// on the consumer side (Go idiom); *repository.TodoRepository satisfies it
// structurally — asserted in todo_test.go.
type Repository interface {
	Create(userID string, todo *model.Todo) (*model.Todo, error)
	FindByID(userID, todoID string) (*model.Todo, error)
	Update(userID, todoID string, todo *model.Todo) (*model.Todo, error)
	Delete(userID, todoID string) error
	List(userID string, params repository.ListParams) (*repository.ListResult, error)
}

// TodoService owns todo business rules between handlers (LAG-408/409) and
// the repository: creation defaults, ownership (403 vs 404), partial-update
// merge, and list parameter parsing. It never touches SQL or HTTP.
type TodoService struct {
	repo Repository
}

func NewTodoService(repo Repository) *TodoService {
	return &TodoService{repo: repo}
}

func (s *TodoService) Create(userID string, todo *model.Todo) (*model.Todo, error) {
	t := *todo
	if t.Status == "" {
		t.Status = model.TodoStatusPending
	}
	if t.Priority == "" {
		t.Priority = model.TodoPriorityMedium
	}
	return s.repo.Create(userID, &t)
}

// owned fetches a todo scoped to the caller and enforces ownership:
// nil lookup → repository.ErrTodoNotFound (404), foreign owner →
// repository.ErrNotOwner (403).
func (s *TodoService) owned(userID, todoID string) (*model.Todo, error) {
	todo, err := s.repo.FindByID(userID, todoID)
	if err != nil {
		return nil, err
	}
	if todo == nil {
		return nil, repository.ErrTodoNotFound
	}
	if todo.UserID != userID {
		return nil, repository.ErrNotOwner
	}
	return todo, nil
}

func (s *TodoService) Get(userID, todoID string) (*model.Todo, error) {
	return s.owned(userID, todoID)
}

// Update overlays only provided fields onto the ownership-fetched todo
// (OpenAPI TodoUpdate is all-optional; the repository writes every column).
// repository.ErrTodoNotFound from a concurrent delete passes through.
func (s *TodoService) Update(userID, todoID string, changes *model.Todo) (*model.Todo, error) {
	existing, err := s.owned(userID, todoID)
	if err != nil {
		return nil, err
	}
	merged := *existing
	if changes.Title != "" {
		merged.Title = changes.Title
	}
	if changes.Status != "" {
		merged.Status = changes.Status
	}
	if changes.Priority != "" {
		merged.Priority = changes.Priority
	}
	if changes.Description.Valid {
		merged.Description = changes.Description
	}
	if changes.DueDate.Valid {
		merged.DueDate = changes.DueDate
	}
	return s.repo.Update(userID, todoID, &merged)
}

func (s *TodoService) Delete(userID, todoID string) error {
	if _, err := s.owned(userID, todoID); err != nil {
		return err
	}
	return s.repo.Delete(userID, todoID)
}

// ListQuery carries raw list query values (strings only — no HTTP types);
// handlers parse request query params into it.
type ListQuery struct {
	Page     string
	Limit    string
	Status   string
	Priority string
	Title    string
	Sort     string
	Order    string
}

func (s *TodoService) List(userID string, q ListQuery) (*repository.ListResult, error) {
	return s.repo.List(userID, repository.ListParams{
		Page:     parsePositiveInt(q.Page, 1),
		Limit:    clampLimit(parsePositiveInt(q.Limit, 20)),
		Status:   parseCSV(q.Status),
		Priority: parseCSV(q.Priority),
		Title:    q.Title,
		Sort:     q.Sort,
		Order:    q.Order,
	})
}

// parsePositiveInt returns def for absent or invalid values (page default 1,
// limit default 20). The repository re-defaults as defense in depth.
func parsePositiveInt(raw string, def int) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return def
	}
	return n
}

func clampLimit(limit int) int {
	if limit > 100 {
		return 100
	}
	return limit
}

// parseCSV splits a comma-separated filter, trimming tokens and dropping
// empties; an absent filter yields nil.
func parseCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	for _, tok := range strings.Split(raw, ",") {
		if tok = strings.TrimSpace(tok); tok != "" {
			out = append(out, tok)
		}
	}
	return out
}
