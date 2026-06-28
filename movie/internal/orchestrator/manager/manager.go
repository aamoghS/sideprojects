package manager

import (
	"fmt"

	"github.com/aamoghS/sideprojects/movie/internal/orchestrator/task"
	"github.com/aamoghS/sideprojects/movie/internal/orchestrator/worker"
)

// Manager represents the control plane of our orchestrator.
type Manager struct {
	Pending       chan task.Task
	TaskDb        map[string]*task.Task
	EventDb       map[string]*task.TaskEvent
	Workers       []string
	WorkerTaskMap map[string][]string
	TaskWorkerMap map[string]string
	WorkerNodes   map[string]*worker.Worker
}

// NewManager initializes a new Manager.
func NewManager() *Manager {
	return &Manager{
		Pending:       make(chan task.Task, 100),
		TaskDb:        make(map[string]*task.Task),
		EventDb:       make(map[string]*task.TaskEvent),
		WorkerTaskMap: make(map[string][]string),
		TaskWorkerMap: make(map[string]string),
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

// Start runs the scheduler loop, constantly looking for pending tasks to assign.
func (m *Manager) Start() {
	fmt.Println("Manager starting scheduler loop...")
	for t := range m.Pending {
		workerName := m.SelectWorker()
		if workerName != "" {
			m.SendWork(workerName, t)
		} else {
			fmt.Println("No available workers for task", t.ID)
		}
	}
}
