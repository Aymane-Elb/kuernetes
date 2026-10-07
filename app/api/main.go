package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

var db *pgxpool.Pool

func main() {
	db = connectDB()
	defer db.Close()

	fmt.Println("API starting on port 8080...")

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/ready", readyHandler)

	http.HandleFunc("/api/tasks", apiTasksHandler)
	http.HandleFunc("/api/tasks/", apiTaskByIDHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

// ----------------------------------------------------
// DATABASE
// ----------------------------------------------------

func connectDB() *pgxpool.Pool {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")

	databaseURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		dbUser,
		dbPassword,
		dbHost,
		dbPort,
		dbName,
	)

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		log.Fatalf(
			"unable to parse database config: %v",
			err,
		)
	}

	pool, err := pgxpool.NewWithConfig(
		context.Background(),
		config,
	)
	if err != nil {
		log.Fatalf(
			"unable to create database pool: %v",
			err,
		)
	}

	err = pool.Ping(context.Background())
	if err != nil {
		log.Fatalf(
			"unable to connect to database: %v",
			err,
		)
	}

	log.Println("Connected to PostgreSQL")

	return pool
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
// HEALTH
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
// READY
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

	err := db.Ping(r.Context())
	if err != nil {
		writeJSON(
			w,
			http.StatusServiceUnavailable,
			HealthResponse{
				Status: "not ready",
			},
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
		getTasks(w, r)

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
		getTask(w, r, id)

	case http.MethodPut:
		updateTask(w, r, id)

	case http.MethodDelete:
		deleteTask(w, r, id)

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
	r *http.Request,
) {
	rows, err := db.Query(
		r.Context(),
		`
		SELECT id, title, completed
		FROM tasks
		ORDER BY created_at DESC
		`,
	)

	if err != nil {
		log.Println("getTasks query error:", err)

		writeError(
			w,
			http.StatusInternalServerError,
			"unable to fetch tasks",
		)
		return
	}

	defer rows.Close()

	result := []Task{}

	for rows.Next() {
		var task Task

		err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Completed,
		)

		if err != nil {
			log.Println("getTasks scan error:", err)

			writeError(
				w,
				http.StatusInternalServerError,
				"unable to read task",
			)
			return
		}

		result = append(result, task)
	}

	if err := rows.Err(); err != nil {
		log.Println("getTasks rows error:", err)

		writeError(
			w,
			http.StatusInternalServerError,
			"unable to fetch tasks",
		)
		return
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
	r *http.Request,
	id string,
) {
	var task Task

	err := db.QueryRow(
		r.Context(),
		`
		SELECT id, title, completed
		FROM tasks
		WHERE id = $1
		`,
		id,
	).Scan(
		&task.ID,
		&task.Title,
		&task.Completed,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			writeError(
				w,
				http.StatusNotFound,
				"task not found",
			)
			return
		}

		log.Println("getTask query error:", err)

		writeError(
			w,
			http.StatusInternalServerError,
			"unable to fetch task",
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

	_, err = db.Exec(
		r.Context(),
		`
		INSERT INTO tasks (
			id,
			title,
			completed
		)
		VALUES ($1, $2, $3)
		`,
		task.ID,
		task.Title,
		task.Completed,
	)

	if err != nil {
		log.Println("createTask insert error:", err)

		writeError(
			w,
			http.StatusInternalServerError,
			"unable to create task",
		)
		return
	}

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

	commandTag, err := db.Exec(
		r.Context(),
		`
		UPDATE tasks
		SET
			title = $1,
			completed = $2
		WHERE id = $3
		`,
		request.Title,
		request.Completed,
		id,
	)

	if err != nil {
		log.Println("updateTask query error:", err)

		writeError(
			w,
			http.StatusInternalServerError,
			"unable to update task",
		)
		return
	}

	if commandTag.RowsAffected() == 0 {
		writeError(
			w,
			http.StatusNotFound,
			"task not found",
		)
		return
	}

	task := Task{
		ID:        id,
		Title:     request.Title,
		Completed: request.Completed,
	}

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
	r *http.Request,
	id string,
) {
	commandTag, err := db.Exec(
		r.Context(),
		`
		DELETE FROM tasks
		WHERE id = $1
		`,
		id,
	)

	if err != nil {
		log.Println("deleteTask query error:", err)

		writeError(
			w,
			http.StatusInternalServerError,
			"unable to delete task",
		)
		return
	}

	if commandTag.RowsAffected() == 0 {
		writeError(
			w,
			http.StatusNotFound,
			"task not found",
		)
		return
	}

	w.WriteHeader(
		http.StatusNoContent,
	)
}
