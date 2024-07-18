package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Struct User
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Struct Task
type Task struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Task     string `json:"task"`
	Date     string `json:"date"`
}

// Struct UserRepo
type UserRepo struct {
	DB *sql.DB
}

// Struct TaskRepo
type TaskRepo struct {
	DB *sql.DB
}

// Hàm tạo UserRepo mới
func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{DB: db}
}

// Hàm tạo TaskRepo mới
func NewTaskRepo(db *sql.DB) *TaskRepo {
	return &TaskRepo{DB: db}
}

// Hàm tạo user mới
func (r *UserRepo) CreateUser(user *User) error {
	_, err := r.DB.Exec(`INSERT INTO users (name, email) VALUES ($1, $2)`, user.Name, user.Email)
	return err
}

// Hàm thêm task vào database
func (r *TaskRepo) AddTask(username, task, date string) error {
	var taskCount int
	err := r.DB.QueryRow(`SELECT COUNT(*) FROM tasks WHERE username=$1 AND date=$2`, username, date).Scan(&taskCount)
	if err != nil {
		return err
	}

	var maxTasks int
	err = r.DB.QueryRow(`SELECT max_tasks FROM user_limits WHERE username=$1`, username).Scan(&maxTasks)
	if err != nil {
		return err
	}

	if taskCount >= maxTasks {
		return fmt.Errorf("task limit reached for user %s", username)
	}

	_, err = r.DB.Exec(`INSERT INTO tasks (username, task, date) VALUES ($1, $2, $3)`, username, task, date)
	return err
}

// Struct UserTaskHandler
type UserTaskHandler struct {
	UserRepo *UserRepo
	TaskRepo *TaskRepo
}

// Hàm tạo UserTaskHandler mới
func NewUserTaskHandler(userRepo *UserRepo, taskRepo *TaskRepo) *UserTaskHandler {
	return &UserTaskHandler{UserRepo: userRepo, TaskRepo: taskRepo}
}

// Handler tạo user
func (h *UserTaskHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if user.Name == "" || user.Email == "" {
		http.Error(w, "Name and email are required", http.StatusBadRequest)
		return
	}

	err := h.UserRepo.CreateUser(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "User created successfully"})
}

// Handler thêm task
func (h *UserTaskHandler) AddTask(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Username string `json:"username"`
		Task     string `json:"task"`
	}

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Task == "" {
		http.Error(w, "Username and task are required", http.StatusBadRequest)
		return
	}

	today := time.Now().Format("2006-01-02")
	err := h.TaskRepo.AddTask(req.Username, req.Task, today)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Task added successfully"})
}

// Middleware thiết lập Content-Type là JSON
func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func main() {
	log.Println("Starting server...")
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}
	defer db.Close()

	userRepo := NewUserRepo(db)
	taskRepo := NewTaskRepo(db)
	userTaskHandler := NewUserTaskHandler(userRepo, taskRepo)

	router := mux.NewRouter()

	// Endpoint để tạo user
	router.HandleFunc("/users", userTaskHandler.CreateUser).Methods("POST")

	// Endpoint để thêm task
	router.HandleFunc("/tasks", userTaskHandler.AddTask).Methods("POST")

	router.Use(jsonContentTypeMiddleware)

	log.Println("Listening on port 8080")
	log.Fatal(http.ListenAndServe(":8000", jsonContentTypeMiddleware(router)))
}
