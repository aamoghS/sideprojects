package main

import (
	"fmt"

	"movie/internal/orchestrator/manager"
	"movie/internal/orchestrator/worker"
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

	// Start Manager scheduler loop
	go m.Start()

	// Start the API Server to receive Task definitions
	api := manager.Api{
		Port:    5555,
		Manager: m,
	}
	api.Start() // This blocks forever
}
