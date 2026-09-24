package repository

import (
	"database/sql"
	"errors"
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
	ErrTodoNotFound  = errors.New("todo not found")
	ErrNotOwner      = errors.New("not owner")
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

	where, filterArgs := todoFilters(params)
	args := append([]interface{}{userID}, filterArgs...)

	// Dynamic SQL is safe here: every user-supplied value (userID, status,
	// priority, title) is bound as a `?` placeholder, never concatenated.
	// The appended `where` fragment contains only literal `?,` placeholders
	// (todoFilters) and orderByClause whitelists sort/order to fixed strings.
	countQuery := "SELECT COUNT(*) FROM todos WHERE user_id = ? AND deleted_at IS NULL" + where
	var totalCount int
	if err := r.db.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
		return nil, err
	}

	query := "SELECT id, user_id, title, description, status, priority, due_date, created_at, updated_at, deleted_at FROM todos WHERE user_id = ? AND deleted_at IS NULL"
	query += where + orderByClause(params.Sort, params.Order)
	query += " LIMIT ? OFFSET ?"
	args = append(args, params.Limit, (params.Page-1)*params.Limit)

	rows, err := r.db.Query(query, args...)
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

	totalPages := totalCount / params.Limit
	if totalCount%params.Limit > 0 {
		totalPages++
	}

	return &ListResult{
		Todos:      todos,
		TotalCount: totalCount,
		Page:       params.Page,
		Limit:      params.Limit,
		TotalPages: totalPages,
	}, nil
}

// ponytail: extracted from List to keep its cognitive complexity under the linter limit
func todoFilters(params ListParams) (string, []interface{}) {
	var where string
	var args []interface{}
	if len(params.Status) > 0 {
		where += " AND status IN (" + placeholders(len(params.Status)) + ")"
		for _, s := range params.Status {
			args = append(args, s)
		}
	}
	if len(params.Priority) > 0 {
		where += " AND priority IN (" + placeholders(len(params.Priority)) + ")"
		for _, p := range params.Priority {
			args = append(args, p)
		}
	}
	if params.Title != "" {
		where += " AND title LIKE ?"
		args = append(args, "%"+params.Title+"%")
	}
	return where, args
}

func placeholders(n int) string {
	return strings.TrimRight(strings.Repeat("?,", n), ",")
}

func orderByClause(sortField, order string) string {
	dir := "DESC"
	if order == "asc" {
		dir = "ASC"
	}
	switch sortField {
	case "priority":
		return " ORDER BY CASE priority WHEN 'urgent' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 WHEN 'low' THEN 4 END ASC"
	case "due_date":
		return " ORDER BY due_date IS NULL, due_date " + dir
	case "updated_at":
		return " ORDER BY updated_at " + dir
	default:
		return " ORDER BY created_at " + dir
	}
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
