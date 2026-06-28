package manager

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aamoghS/sideprojects/movie/internal/orchestrator/task"

	"github.com/google/uuid"
)

// Api represents the REST API for the Manager.
type Api struct {
	Port    int
	Manager *Manager
}

// Start starts the HTTP server to listen for new tasks.
func (a *Api) Start() {
	http.HandleFunc("/tasks", a.SubmitTaskHandler)
	
	addr := fmt.Sprintf("0.0.0.0:%d", a.Port)
	fmt.Printf("Manager API Server listening on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Printf("API server failed: %v\n", err)
	}
}

// SubmitTaskHandler handles POST requests to create a new task.
func (a *Api) SubmitTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST is allowed", http.StatusMethodNotAllowed)
		return
	}

	var t task.Task
	err := json.NewDecoder(r.Body).Decode(&t)
	if err != nil {
		http.Error(w, "Invalid task JSON payload", http.StatusBadRequest)
		return
	}

	// Assign an ID if not provided
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	t.State = task.Pending

	// Save to DB and schedule
	a.Manager.TaskDb[t.ID] = &t
	a.Manager.Pending <- t

	fmt.Printf("API received task %s (%s)\n", t.Name, t.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Task submitted successfully",
		"task_id": t.ID.String(),
	})
}
