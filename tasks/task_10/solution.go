package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type TaskRepo interface {
	Create(title string) (Task, error)
	Get(id string) (Task, bool)
	List(done bool) []Task
	SetDone(id string, done bool) (Task, error)
}

type Clock interface {
	Now() time.Time
}

var (
	ErrNotFound     = errors.New("task not found")
	ErrInvalidTitle = errors.New("invalid title")
)

var globalTaskID atomic.Uint64

type inMemoryTaskRepo struct {
	mu    sync.RWMutex
	clock Clock
	tasks map[string]Task
}

// Создание репозитория
func NewInMemoryTaskRepo(clock Clock) TaskRepo {
	return &inMemoryTaskRepo{
		clock: clock,
		tasks: make(map[string]Task),
	}
}

// Создание задачи
func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	title = strings.TrimSpace(title)

	if title == "" {
		return Task{}, ErrInvalidTitle
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	id := globalTaskID.Add(1)

	task := Task{
		ID:        fmt.Sprint(id),
		Title:     title,
		Done:      false,
		UpdatedAt: r.clock.Now(),
	}

	r.tasks[task.ID] = task

	return task, nil
}

// Получение задачи по ID
func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, ok := r.tasks[id]

	return task, ok
}

// Получение списка задач
func (r *inMemoryTaskRepo) List(done bool) []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Task, 0)

	for _, task := range r.tasks {
		if task.Done == done {
			result = append(result, task)
		}
	}

	return result
}

// Изменение статуса задачи
func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]

	if !ok {
		return Task{}, ErrNotFound
	}

	if task.Done == done {
		return task, nil
	}

	task.Done = done
	task.UpdatedAt = r.clock.Now()

	r.tasks[id] = task

	return task, nil
}

// HTTP Handler
type httpHandler struct {
	repo TaskRepo
}

func NewHTTPHandler(repo TaskRepo) http.Handler {
	return &httpHandler{
		repo: repo,
	}
}

// Маршрутизация запросов
func (h *httpHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	path := r.URL.Path

	if path == "/tasks" {
		switch r.Method {
		case http.MethodPost:
			h.handleCreate(w, r)

		case http.MethodGet:
			h.handleList(w, r)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}

		return
	}

	if !strings.HasPrefix(path, "/tasks/") {
		http.NotFound(w, r)
		return
	}

	id := strings.TrimPrefix(path, "/tasks/")

	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, id)

	case http.MethodPatch:
		h.handlePatch(w, r, id)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// POST /tasks
func (h *httpHandler) handleCreate(
	w http.ResponseWriter,
	r *http.Request,
) {
	var body struct {
		Title *string `json:"title"`
	}

	if err := decodeStrictJSON(r.Body, &body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if body.Title == nil {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	task, err := h.repo.Create(*body.Title)

	if err != nil {
		if errors.Is(err, ErrInvalidTitle) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

// GET /tasks/{id}
func (h *httpHandler) handleGet(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	task, ok := h.repo.Get(id)

	if !ok {
		http.NotFound(w, r)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// GET /tasks
func (h *httpHandler) handleList(
	w http.ResponseWriter,
	r *http.Request,
) {
	done := false

	values, exists := r.URL.Query()["done"]

	if exists {
		if len(values) != 1 {
			http.Error(w, "invalid done", http.StatusBadRequest)
			return
		}

		switch values[0] {
		case "true":
			done = true

		case "false":
			done = false

		default:
			http.Error(w, "invalid done", http.StatusBadRequest)
			return
		}
	}

	tasks := h.repo.List(done)

	// Сортировка по времени, затем по ID
	slices.SortFunc(tasks, func(a, b Task) int {
		if a.UpdatedAt.After(b.UpdatedAt) {
			return -1
		}

		if a.UpdatedAt.Before(b.UpdatedAt) {
			return 1
		}

		return strings.Compare(a.ID, b.ID)
	})

	writeJSON(w, http.StatusOK, tasks)
}

// PATCH /tasks/{id}
func (h *httpHandler) handlePatch(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	var body struct {
		Done *bool `json:"done"`
	}

	if err := decodeStrictJSON(r.Body, &body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if body.Done == nil {
		http.Error(w, "done is required", http.StatusBadRequest)
		return
	}

	task, err := h.repo.SetDone(id, *body.Done)

	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.NotFound(w, r)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// Строгая проверка JSON
func decodeStrictJSON(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)

	decoder.DisallowUnknownFields()

	if err := decoder.Decode(v); err != nil {
		return err
	}

	// Проверяем, что после объекта нет других данных.
	var extra any

	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}

		return err
	}

	return nil
}

// Отправка JSON-ответа
func writeJSON(
	w http.ResponseWriter,
	status int,
	v any,
) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(v)
}
