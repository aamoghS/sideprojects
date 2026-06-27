package main

import (
	"fmt"
	"time"

	"movie/internal/orchestrator/manager"
	"movie/internal/orchestrator/task"
	"movie/internal/orchestrator/worker"

	"github.com/google/uuid"
)

func main() {
	fmt.Println("Starting Custom Orchestrator...")

	// Create manager
	m := manager.NewManager()

	// Create a worker
	w1 := worker.NewWorker("worker-1")
	m.AddWorker(w1.Name, w1)

	// Start worker loop
	go func() {
		for t := range w1.Queue {
			fmt.Printf("Worker %s received task %s\n", w1.Name, t.ID)
			err := w1.RunTask(t)
			if err != nil {
				fmt.Printf("Worker %s failed to run task %s: %v\n", w1.Name, t.ID, err)
			}
		}
	}()

	// Create a dummy task
	t1 := task.Task{
		ID:    uuid.New(),
		Name:  "my-postgres",
		Image: "postgres:13",
		State: task.Pending,
	}

	// Manager schedules the task
	workerName := m.SelectWorker()
	m.SendWork(workerName, t1)

	// Wait a bit to observe output
	time.Sleep(5 * time.Second)
	fmt.Println("Orchestrator shutting down.")
}
