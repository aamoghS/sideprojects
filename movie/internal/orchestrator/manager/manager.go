package manager

import (
	"fmt"

	"movie/internal/orchestrator/task"
	"movie/internal/orchestrator/worker"

	"github.com/google/uuid"
)

// Manager represents the control plane of our orchestrator.
type Manager struct {
	Pending       chan task.Task
	TaskDb        map[uuid.UUID]*task.Task
	EventDb       map[uuid.UUID]*task.TaskEvent
	Workers       []string
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string
	WorkerNodes   map[string]*worker.Worker
}

// NewManager initializes a new Manager.
func NewManager() *Manager {
	return &Manager{
		Pending:       make(chan task.Task, 100),
		TaskDb:        make(map[uuid.UUID]*task.Task),
		EventDb:       make(map[uuid.UUID]*task.TaskEvent),
		WorkerTaskMap: make(map[string][]uuid.UUID),
		TaskWorkerMap: make(map[uuid.UUID]string),
		WorkerNodes:   make(map[string]*worker.Worker),
	}
}

// SelectWorker picks a worker in a round-robin or simplistic manner.
func (m *Manager) SelectWorker() string {
	// Simple naive scheduling: return the first worker
	if len(m.Workers) > 0 {
		return m.Workers[0]
	}
	return ""
}

// SendWork dispatches a task to the selected worker.
func (m *Manager) SendWork(workerName string, t task.Task) {
	workerNode, ok := m.WorkerNodes[workerName]
	if !ok {
		fmt.Printf("Worker %s not found\n", workerName)
		return
	}

	fmt.Printf("Manager sending task %s to worker %s\n", t.ID, workerName)
	workerNode.Queue <- t
}

// AddWorker registers a new worker.
func (m *Manager) AddWorker(name string, w *worker.Worker) {
	m.Workers = append(m.Workers, name)
	m.WorkerNodes[name] = w
}
