package repository

import (
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"

	"todo-list-api/internal/model"

	"github.com/google/uuid"
)

type TodoRepository struct {
	db *sql.DB
}

func NewTodoRepository(db *sql.DB) *TodoRepository {
	return &TodoRepository{db: db}
}

type ListParams struct {
	Page     int
	Limit    int
	Status   []string
	Priority []string
	Title    string
	Sort     string
	Order    string
}

type ListResult struct {
	Todos      []model.Todo
	TotalCount int
	Page       int
	Limit      int
	TotalPages int
}

func (r *TodoRepository) Create(userID string, todo *model.Todo) (*model.Todo, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := r.db.Exec(
		`INSERT INTO todos (id, user_id, title, description, status, priority, due_date, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, userID, todo.Title, todo.Description, todo.Status, todo.Priority, todo.DueDate, now, now,
	)
	if err != nil {
		return nil, err
	}

	return &model.Todo{
		ID:          id,
		UserID:      userID,
		Title:       todo.Title,
		Description: todo.Description,
		Status:      todo.Status,
		Priority:    todo.Priority,
		DueDate:     todo.DueDate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (r *TodoRepository) FindByID(userID, todoID string) (*model.Todo, error) {
	t := &model.Todo{}
	err := r.db.QueryRow(
		`SELECT id, user_id, title, description, status, priority, due_date, created_at, updated_at, deleted_at
		 FROM todos WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		todoID, userID,
	).Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.DueDate, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

var (
	ErrTodoNotFound = errors.New("todo not found")
	ErrNotOwner     = errors.New("not owner")
)

func (r *TodoRepository) Update(userID, todoID string, todo *model.Todo) (*model.Todo, error) {
	now := time.Now()
	result, err := r.db.Exec(
		`UPDATE todos SET title = ?, description = ?, status = ?, priority = ?, due_date = ?, updated_at = ?
		 WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		todo.Title, todo.Description, todo.Status, todo.Priority, todo.DueDate, now, todoID, userID,
	)
	if err != nil {
		return nil, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, ErrTodoNotFound
	}
	return &model.Todo{
		ID:          todoID,
		UserID:      userID,
		Title:       todo.Title,
		Description: todo.Description,
		Status:      todo.Status,
		Priority:    todo.Priority,
		DueDate:     todo.DueDate,
		CreatedAt:   todo.CreatedAt,
		UpdatedAt:   now,
	}, nil
}

func (r *TodoRepository) Delete(userID, todoID string) error {
	now := time.Now()
	result, err := r.db.Exec(
		`UPDATE todos SET deleted_at = ? WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		now, todoID, userID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrTodoNotFound
	}
	return nil
}

// listQuery is a fixed-shape literal: every value is bound as a `?`
// placeholder, nothing is ever concatenated into the SQL. Optional filters
// use `(? IS NULL OR ...)` so the query text never changes.
const listQuery = `SELECT id, user_id, title, description, status, priority, due_date, created_at, updated_at, deleted_at
		FROM todos
		WHERE user_id = ?
		  AND deleted_at IS NULL
		  AND (? IS NULL OR instr(?, ',' || status || ',') > 0)
		  AND (? IS NULL OR instr(?, ',' || priority || ',') > 0)
		  AND (? IS NULL OR title LIKE ?)`

func (r *TodoRepository) List(userID string, params ListParams) (*ListResult, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	rows, err := r.db.Query(listQuery, listArgs(userID, params)...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	todos, err := scanTodos(rows)
	if err != nil {
		return nil, err
	}
	if todos == nil {
		todos = []model.Todo{}
	}
	totalCount := len(todos)

	// ponytail: ORDER BY/LIMIT applied in Go so the SQL stays a literal;
	// push sorting and pagination back into SQL if datasets outgrow memory.
	sortTodos(todos, params.Sort, params.Order)
	start := (params.Page - 1) * params.Limit
	if start > totalCount {
		start = totalCount
	}
	end := start + params.Limit
	if end > totalCount {
		end = totalCount
	}

	totalPages := totalCount / params.Limit
	if totalCount%params.Limit > 0 {
		totalPages++
	}

	return &ListResult{
		Todos:      todos[start:end],
		TotalCount: totalCount,
		Page:       params.Page,
		Limit:      params.Limit,
		TotalPages: totalPages,
	}, nil
}

// listArgs matches listQuery's WHERE clause: userID plus a (flag, value)
// pair per optional filter; binding nil disables that filter.
func listArgs(userID string, p ListParams) []interface{} {
	return []interface{}{
		userID,
		instrSet(p.Status), instrSet(p.Status),
		instrSet(p.Priority), instrSet(p.Priority),
		titlePattern(p.Title), titlePattern(p.Title),
	}
}

// instrSet returns ",v1,v2," for token matching against a column value,
// or nil when the filter is absent.
func instrSet(values []string) interface{} {
	if len(values) == 0 {
		return nil
	}
	return "," + strings.Join(values, ",") + ","
}

func titlePattern(title string) interface{} {
	if title == "" {
		return nil
	}
	return "%" + title + "%"
}

var priorityRanks = map[model.TodoPriority]int{
	model.TodoPriorityUrgent: 1,
	model.TodoPriorityHigh:   2,
	model.TodoPriorityMedium: 3,
	model.TodoPriorityLow:    4,
}

// sortTodos applies the API sort contract: priority ranks ascending
// regardless of order, due_date nulls last, everything else honors order.
func sortTodos(todos []model.Todo, field, order string) {
	desc := order != "asc"
	sort.SliceStable(todos, func(i, j int) bool {
		a, b := todos[i], todos[j]
		switch field {
		case "priority":
			return priorityRanks[a.Priority] < priorityRanks[b.Priority]
		case "due_date":
			if a.DueDate.Valid != b.DueDate.Valid {
				return !a.DueDate.Valid
			}
			return compareTime(a.DueDate.Time, b.DueDate.Time, desc)
		case "updated_at":
			return compareTime(a.UpdatedAt, b.UpdatedAt, desc)
		default: // created_at and unknown fields
			return compareTime(a.CreatedAt, b.CreatedAt, desc)
		}
	})
}

func compareTime(a, b time.Time, desc bool) bool {
	if a.Equal(b) {
		return false
	}
	if desc {
		return a.After(b)
	}
	return a.Before(b)
}

func scanTodos(rows *sql.Rows) ([]model.Todo, error) {
	var todos []model.Todo
	for rows.Next() {
		var t model.Todo
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.DueDate, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt); err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}
	return todos, rows.Err()
}

func (r *TodoRepository) DeleteByUserID(userID string) error {
	now := time.Now()
	_, err := r.db.Exec(
		`UPDATE todos SET deleted_at = ? WHERE user_id = ? AND deleted_at IS NULL`,
		now, userID,
	)
	return err
}
