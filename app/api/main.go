package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type Task struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

type CreateTaskRequest struct {
	Title string `json:"title"`
}

type UpdateTaskRequest struct {
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

var (
	tasks = make(map[string]Task)
	mu    sync.RWMutex
)

func main() {
	fmt.Println("API starting on port 8080...")

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ready", readyHandler)

	// Collection:
	// GET  /api/tasks
	// POST /api/tasks
	http.HandleFunc("/api/tasks", apiTasksHandler)

	// Individual task:
	// GET    /api/tasks/{id}
	// PUT    /api/tasks/{id}
	// DELETE /api/tasks/{id}
	http.HandleFunc("/api/tasks/", apiTaskByIDHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

// ----------------------------------------------------
// CORS
// ----------------------------------------------------

func enableCors(w http.ResponseWriter) {
	w.Header().Set(
		"Access-Control-Allow-Origin",
		"*",
	)

	w.Header().Set(
		"Access-Control-Allow-Headers",
		"Content-Type",
	)

	w.Header().Set(
		"Access-Control-Allow-Methods",
		"GET, POST, PUT, DELETE, OPTIONS",
	)
}

// ----------------------------------------------------
// JSON helpers
// ----------------------------------------------------

func writeJSON(
	w http.ResponseWriter,
	status int,
	data interface{},
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(status)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Println(
				"failed to encode response:",
				err,
			)
		}
	}
}

func writeError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	writeJSON(
		w,
		status,
		ErrorResponse{
			Error: message,
		},
	)
}

// ----------------------------------------------------
// Health
// ----------------------------------------------------

func healthHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	enableCors(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodGet {
		writeError(
			w,
			http.StatusMethodNotAllowed,
			"method not allowed",
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		HealthResponse{
			Status: "healthy",
		},
	)
}

// ----------------------------------------------------
// Ready
// ----------------------------------------------------

func readyHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	enableCors(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodGet {
		writeError(
			w,
			http.StatusMethodNotAllowed,
			"method not allowed",
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		HealthResponse{
			Status: "ready",
		},
	)
}

// ----------------------------------------------------
// /api/tasks
// ----------------------------------------------------

func apiTasksHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	enableCors(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch r.Method {

	case http.MethodGet:
		getTasks(w)

	case http.MethodPost:
		createTask(w, r)

	default:
		writeError(
			w,
			http.StatusMethodNotAllowed,
			"method not allowed",
		)
	}
}

// ----------------------------------------------------
// /api/tasks/{id}
// ----------------------------------------------------

func apiTaskByIDHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	enableCors(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	id := strings.TrimPrefix(
		r.URL.Path,
		"/api/tasks/",
	)

	id = strings.TrimSpace(id)

	if id == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"task id is required",
		)
		return
	}

	switch r.Method {

	case http.MethodGet:
		getTask(w, id)

	case http.MethodPut:
		updateTask(w, r, id)

	case http.MethodDelete:
		deleteTask(w, id)

	default:
		writeError(
			w,
			http.StatusMethodNotAllowed,
			"method not allowed",
		)
	}
}

// ----------------------------------------------------
// GET /api/tasks
// ----------------------------------------------------

func getTasks(
	w http.ResponseWriter,
) {
	mu.RLock()
	defer mu.RUnlock()

	result := make(
		[]Task,
		0,
		len(tasks),
	)

	for _, task := range tasks {
		result = append(
			result,
			task,
		)
	}

	writeJSON(
		w,
		http.StatusOK,
		result,
	)
}

// ----------------------------------------------------
// GET /api/tasks/{id}
// ----------------------------------------------------

func getTask(
	w http.ResponseWriter,
	id string,
) {
	mu.RLock()
	defer mu.RUnlock()

	task, exists := tasks[id]

	if !exists {
		writeError(
			w,
			http.StatusNotFound,
			"task not found",
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		task,
	)
}

// ----------------------------------------------------
// POST /api/tasks
// ----------------------------------------------------

func createTask(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request CreateTaskRequest

	err := json.NewDecoder(
		r.Body,
	).Decode(&request)

	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	request.Title = strings.TrimSpace(
		request.Title,
	)

	if request.Title == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"title is required",
		)
		return
	}

	task := Task{
		ID:        uuid.NewString(),
		Title:     request.Title,
		Completed: false,
	}

	mu.Lock()
	tasks[task.ID] = task
	mu.Unlock()

	writeJSON(
		w,
		http.StatusCreated,
		task,
	)
}

// ----------------------------------------------------
// PUT /api/tasks/{id}
// ----------------------------------------------------

func updateTask(
	w http.ResponseWriter,
	r *http.Request,
	id string,
) {
	var request UpdateTaskRequest

	err := json.NewDecoder(
		r.Body,
	).Decode(&request)

	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	request.Title = strings.TrimSpace(
		request.Title,
	)

	if request.Title == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"title is required",
		)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	task, exists := tasks[id]

	if !exists {
		writeError(
			w,
			http.StatusNotFound,
			"task not found",
		)
		return
	}

	task.Title = request.Title
	task.Completed = request.Completed

	tasks[id] = task

	writeJSON(
		w,
		http.StatusOK,
		task,
	)
}

// ----------------------------------------------------
// DELETE /api/tasks/{id}
// ----------------------------------------------------

func deleteTask(
	w http.ResponseWriter,
	id string,
) {
	mu.Lock()
	defer mu.Unlock()

	_, exists := tasks[id]

	if !exists {
		writeError(
			w,
			http.StatusNotFound,
			"task not found",
		)
		return
	}

	delete(tasks, id)

	w.WriteHeader(
		http.StatusNoContent,
	)
}
