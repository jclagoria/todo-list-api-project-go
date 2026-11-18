package service

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"todo-list-api/internal/model"
	"todo-list-api/internal/repository"
)

// Compile-time assertion: the real repository satisfies the consumer-side interface.
var _ Repository = (*repository.TodoRepository)(nil)

// mockRepo is a hand-rolled Repository (stdlib only — no mock framework).
// Each method records its call arguments and returns its scripted values.
type mockRepo struct {
	created   *model.Todo
	find      *model.Todo
	updated   *model.Todo
	listed    *repository.ListResult
	findErr   error
	createErr error
	updateErr error
	deleteErr error
	listErr   error

	createCalls, findCalls, updateCalls, deleteCalls, listCalls int
	gotUserID                                                   string
	gotTodoID                                                   string
	gotTodo                                                     *model.Todo
	gotParams                                                   repository.ListParams
}

func (m *mockRepo) Create(userID string, todo *model.Todo) (*model.Todo, error) {
	m.createCalls++
	m.gotUserID = userID
	m.gotTodo = todo
	return m.created, m.createErr
}

func (m *mockRepo) FindByID(userID, todoID string) (*model.Todo, error) {
	m.findCalls++
	m.gotUserID = userID
	m.gotTodoID = todoID
	return m.find, m.findErr
}

func (m *mockRepo) Update(userID, todoID string, todo *model.Todo) (*model.Todo, error) {
	m.updateCalls++
	m.gotUserID = userID
	m.gotTodoID = todoID
	m.gotTodo = todo
	return m.updated, m.updateErr
}

func (m *mockRepo) Delete(userID, todoID string) error {
	m.deleteCalls++
	m.gotUserID = userID
	m.gotTodoID = todoID
	return m.deleteErr
}

func (m *mockRepo) List(userID string, params repository.ListParams) (*repository.ListResult, error) {
	m.listCalls++
	m.gotUserID = userID
	m.gotParams = params
	return m.listed, m.listErr
}

func todoOwnedBy(userID string) *model.Todo {
	return &model.Todo{
		ID:          "t1",
		UserID:      userID,
		Title:       "Buy milk",
		Description: sql.NullString{String: "from the shop", Valid: true},
		Status:      model.TodoStatusInProgress,
		Priority:    model.TodoPriorityHigh,
		DueDate:     sql.NullTime{Time: time.Now().Add(24 * time.Hour), Valid: true},
	}
}

// --- Create: defaults ---

func TestCreateDefaultsAllOmitted(t *testing.T) {
	mock := &mockRepo{created: &model.Todo{ID: "new"}}
	svc := NewTodoService(mock)

	got, err := svc.Create("u1", &model.Todo{Title: "Buy milk"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if mock.createCalls != 1 {
		t.Errorf("repo Create calls = %d, want 1", mock.createCalls)
	}
	if mock.gotUserID != "u1" {
		t.Errorf("repo userID = %q, want u1", mock.gotUserID)
	}
	if mock.gotTodo.Status != model.TodoStatusPending {
		t.Errorf("status = %q, want pending", mock.gotTodo.Status)
	}
	if mock.gotTodo.Priority != model.TodoPriorityMedium {
		t.Errorf("priority = %q, want medium", mock.gotTodo.Priority)
	}
	if mock.gotTodo.Title != "Buy milk" {
		t.Errorf("title = %q, want Buy milk", mock.gotTodo.Title)
	}
	if got != mock.created {
		t.Errorf("returned todo = %v, want repository result", got)
	}
}

func TestCreateDefaultsPriorityOnly(t *testing.T) {
	mock := &mockRepo{}
	svc := NewTodoService(mock)

	_, err := svc.Create("u1", &model.Todo{Title: "x", Status: model.TodoStatusInProgress})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if mock.gotTodo.Status != model.TodoStatusInProgress {
		t.Errorf("status = %q, want in_progress preserved", mock.gotTodo.Status)
	}
	if mock.gotTodo.Priority != model.TodoPriorityMedium {
		t.Errorf("priority = %q, want medium default", mock.gotTodo.Priority)
	}
}

func TestCreateExplicitValuesPassThrough(t *testing.T) {
	mock := &mockRepo{}
	svc := NewTodoService(mock)

	_, err := svc.Create("u1", &model.Todo{
		Title:    "x",
		Status:   model.TodoStatusDone,
		Priority: model.TodoPriorityUrgent,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if mock.gotTodo.Status != model.TodoStatusDone || mock.gotTodo.Priority != model.TodoPriorityUrgent {
		t.Errorf("got %q/%q, want done/urgent", mock.gotTodo.Status, mock.gotTodo.Priority)
	}
}

func TestCreateDoesNotMutateCallerTodo(t *testing.T) {
	mock := &mockRepo{}
	svc := NewTodoService(mock)

	in := &model.Todo{Title: "x"}
	if _, err := svc.Create("u1", in); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if in.Status != "" || in.Priority != "" {
		t.Errorf("caller todo mutated: status=%q priority=%q", in.Status, in.Priority)
	}
}

// --- Get: ownership + not found ---

func TestGetOwned(t *testing.T) {
	want := todoOwnedBy("u1")
	mock := &mockRepo{find: want}
	svc := NewTodoService(mock)

	got, err := svc.Get("u1", "t1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != want {
		t.Errorf("got %v, want repository todo", got)
	}
	if mock.gotUserID != "u1" || mock.gotTodoID != "t1" {
		t.Errorf("repo got (%q, %q), want (u1, t1)", mock.gotUserID, mock.gotTodoID)
	}
}

func TestGetNotFound(t *testing.T) {
	mock := &mockRepo{find: nil}
	svc := NewTodoService(mock)

	if _, err := svc.Get("u1", "missing"); !errors.Is(err, repository.ErrTodoNotFound) {
		t.Errorf("err = %v, want ErrTodoNotFound", err)
	}
}

func TestGetNotOwner(t *testing.T) {
	mock := &mockRepo{find: todoOwnedBy("someone-else")}
	svc := NewTodoService(mock)

	if _, err := svc.Get("u1", "t1"); !errors.Is(err, repository.ErrNotOwner) {
		t.Errorf("err = %v, want ErrNotOwner", err)
	}
	if mock.updateCalls != 0 || mock.deleteCalls != 0 {
		t.Errorf("mutation invoked on foreign todo: update=%d delete=%d", mock.updateCalls, mock.deleteCalls)
	}
}

// --- Update: ownership, merge, race ---

func TestUpdateNotFound(t *testing.T) {
	mock := &mockRepo{}
	svc := NewTodoService(mock)

	if _, err := svc.Update("u1", "missing", &model.Todo{Title: "x"}); !errors.Is(err, repository.ErrTodoNotFound) {
		t.Errorf("err = %v, want ErrTodoNotFound", err)
	}
	if mock.updateCalls != 0 {
		t.Errorf("repo Update invoked %d times, want 0", mock.updateCalls)
	}
}

func TestUpdateNotOwner(t *testing.T) {
	mock := &mockRepo{find: todoOwnedBy("someone-else")}
	svc := NewTodoService(mock)

	if _, err := svc.Update("u1", "t1", &model.Todo{Title: "x"}); !errors.Is(err, repository.ErrNotOwner) {
		t.Errorf("err = %v, want ErrNotOwner", err)
	}
	if mock.updateCalls != 0 {
		t.Errorf("repo Update invoked %d times, want 0", mock.updateCalls)
	}
}

func TestUpdateMergeStatusOnlyPreservesExisting(t *testing.T) {
	existing := todoOwnedBy("u1")
	mock := &mockRepo{find: existing, updated: existing}
	svc := NewTodoService(mock)

	changes := &model.Todo{Status: model.TodoStatusDone}
	got, err := svc.Update("u1", "t1", changes)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got != existing {
		t.Errorf("returned todo = %v, want repository result", got)
	}
	if mock.updateCalls != 1 {
		t.Fatalf("repo Update calls = %d, want 1", mock.updateCalls)
	}
	if mock.gotUserID != "u1" || mock.gotTodoID != "t1" {
		t.Errorf("repo got (%q, %q), want (u1, t1)", mock.gotUserID, mock.gotTodoID)
	}
	gotMerged := mock.gotTodo
	if gotMerged.Status != model.TodoStatusDone {
		t.Errorf("status = %q, want done", gotMerged.Status)
	}
	if gotMerged.Title != existing.Title ||
		gotMerged.Description != existing.Description ||
		gotMerged.Priority != existing.Priority ||
		gotMerged.DueDate != existing.DueDate {
		t.Errorf("omitted fields not preserved: %+v", gotMerged)
	}
}

func TestUpdateMergeAllFields(t *testing.T) {
	existing := todoOwnedBy("u1")
	mock := &mockRepo{find: existing}
	svc := NewTodoService(mock)

	changes := &model.Todo{
		Title:       "New title",
		Description: sql.NullString{String: "New description", Valid: true},
		Status:      model.TodoStatusDone,
		Priority:    model.TodoPriorityLow,
		DueDate:     sql.NullTime{Time: time.Now(), Valid: true},
	}
	if _, err := svc.Update("u1", "t1", changes); err != nil {
		t.Fatalf("Update: %v", err)
	}
	m := mock.gotTodo
	if m.Title != changes.Title || m.Status != changes.Status || m.Priority != changes.Priority {
		t.Errorf("scalar fields not overlaid: %+v", m)
	}
	if m.Description != changes.Description || m.DueDate != changes.DueDate {
		t.Errorf("nullable fields not overlaid: %+v", m)
	}
}

func TestUpdateEmptyStatusPriorityNotProvided(t *testing.T) {
	existing := todoOwnedBy("u1")
	mock := &mockRepo{find: existing}
	svc := NewTodoService(mock)

	if _, err := svc.Update("u1", "t1", &model.Todo{Title: "New title"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	m := mock.gotTodo
	if m.Status != existing.Status || m.Priority != existing.Priority {
		t.Errorf("empty status/priority overwrote existing: %q/%q", m.Status, m.Priority)
	}
	if m.Title != "New title" {
		t.Errorf("title = %q, want New title", m.Title)
	}
}

func TestUpdatePassesThroughRepositoryNotFound(t *testing.T) {
	// Concurrent delete: ownership fetch succeeds, repository UPDATE hits 0 rows.
	existing := todoOwnedBy("u1")
	mock := &mockRepo{find: existing, updateErr: repository.ErrTodoNotFound}
	svc := NewTodoService(mock)

	if _, err := svc.Update("u1", "t1", &model.Todo{Title: "x"}); !errors.Is(err, repository.ErrTodoNotFound) {
		t.Errorf("err = %v, want ErrTodoNotFound passthrough", err)
	}
}

// --- Delete ---

func TestDeleteNotFound(t *testing.T) {
	mock := &mockRepo{}
	svc := NewTodoService(mock)

	if err := svc.Delete("u1", "missing"); !errors.Is(err, repository.ErrTodoNotFound) {
		t.Errorf("err = %v, want ErrTodoNotFound", err)
	}
	if mock.deleteCalls != 0 {
		t.Errorf("repo Delete invoked %d times, want 0", mock.deleteCalls)
	}
}

func TestDeleteNotOwner(t *testing.T) {
	mock := &mockRepo{find: todoOwnedBy("someone-else")}
	svc := NewTodoService(mock)

	if err := svc.Delete("u1", "t1"); !errors.Is(err, repository.ErrNotOwner) {
		t.Errorf("err = %v, want ErrNotOwner", err)
	}
	if mock.deleteCalls != 0 {
		t.Errorf("repo Delete invoked %d times, want 0", mock.deleteCalls)
	}
}

func TestDeleteOwned(t *testing.T) {
	existing := todoOwnedBy("u1")
	mock := &mockRepo{find: existing}
	svc := NewTodoService(mock)

	if err := svc.Delete("u1", "t1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if mock.deleteCalls != 1 || mock.gotUserID != "u1" || mock.gotTodoID != "t1" {
		t.Errorf("repo Delete got calls=%d (%q, %q), want 1 (u1, t1)",
			mock.deleteCalls, mock.gotUserID, mock.gotTodoID)
	}
}

// --- List: filters, sort, pagination ---

func listParams(t *testing.T, q ListQuery) repository.ListParams {
	t.Helper()
	mock := &mockRepo{}
	if _, err := NewTodoService(mock).List("u1", q); err != nil {
		t.Fatalf("List: %v", err)
	}
	if mock.listCalls != 1 {
		t.Fatalf("repo List calls = %d, want 1", mock.listCalls)
	}
	return mock.gotParams
}

func TestListCSVFilters(t *testing.T) {
	p := listParams(t, ListQuery{
		Status:   "pending, done ,",
		Priority: "urgent",
		Title:    "buy",
	})
	if !reflect.DeepEqual(p.Status, []string{"pending", "done"}) {
		t.Errorf("status = %#v, want [pending done]", p.Status)
	}
	if !reflect.DeepEqual(p.Priority, []string{"urgent"}) {
		t.Errorf("priority = %#v, want [urgent]", p.Priority)
	}
	if p.Title != "buy" {
		t.Errorf("title = %q, want buy", p.Title)
	}
}

func TestListSortPassthrough(t *testing.T) {
	p := listParams(t, ListQuery{Sort: "due_date", Order: "asc"})
	if p.Sort != "due_date" || p.Order != "asc" {
		t.Errorf("sort/order = %q/%q, want due_date/asc", p.Sort, p.Order)
	}
}

func TestListNoFilters(t *testing.T) {
	p := listParams(t, ListQuery{})
	if len(p.Status) != 0 || len(p.Priority) != 0 {
		t.Errorf("filters = %#v/%#v, want empty", p.Status, p.Priority)
	}
	if p.Title != "" || p.Sort != "" || p.Order != "" {
		t.Errorf("raw fields = %q/%q/%q, want empty", p.Title, p.Sort, p.Order)
	}
	if p.Page != 1 || p.Limit != 20 {
		t.Errorf("pagination = %d/%d, want 1/20", p.Page, p.Limit)
	}
}

func TestListPaginationParsed(t *testing.T) {
	p := listParams(t, ListQuery{Page: "2", Limit: "10"})
	if p.Page != 2 || p.Limit != 10 {
		t.Errorf("pagination = %d/%d, want 2/10", p.Page, p.Limit)
	}
}

func TestListPaginationDefaultsOnInvalid(t *testing.T) {
	p := listParams(t, ListQuery{Page: "abc", Limit: "-5"})
	if p.Page != 1 || p.Limit != 20 {
		t.Errorf("pagination = %d/%d, want 1/20", p.Page, p.Limit)
	}
}

func TestListLimitClamped(t *testing.T) {
	p := listParams(t, ListQuery{Limit: "500"})
	if p.Limit != 100 {
		t.Errorf("limit = %d, want 100", p.Limit)
	}
}

func TestListResultReturnedUnchanged(t *testing.T) {
	want := &repository.ListResult{
		Todos:      []model.Todo{*todoOwnedBy("u1")},
		TotalCount: 1,
		Page:       1,
		Limit:      20,
		TotalPages: 1,
	}
	mock := &mockRepo{listed: want}
	got, err := NewTodoService(mock).List("u1", ListQuery{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got != want {
		t.Errorf("got %v, want repository result unchanged", got)
	}
}
