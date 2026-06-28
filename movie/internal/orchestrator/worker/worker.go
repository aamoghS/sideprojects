package worker

import (
	"fmt"
	"os/exec"

	"github.com/aamoghS/sideprojects/movie/internal/orchestrator/task"
)

// Worker represents a node in our orchestrator that can run tasks.
type Worker struct {
	Name      string
	Queue     chan task.Task
	Db        map[string]*task.Task
	TaskCount int
}

// NewWorker initializes a new Worker.
func NewWorker(name string) *Worker {
	return &Worker{
		Name:  name,
		Queue: make(chan task.Task, 10),
		Db:    make(map[string]*task.Task),
	}
}

// RunTask starts a task as a local process.
func (w *Worker) RunTask(t task.Task) error {
	w.Db[t.ID] = &t
	w.TaskCount++

	// In our in-house orchestrator, 'Image' represents the binary or command to run.
	cmd := exec.Command(t.Image)
	
	// Start the process
	if err := cmd.Start(); err != nil {
		t.State = task.Failed
		w.Db[t.ID] = &t
		return fmt.Errorf("failed to start process: %w", err)
	}

	t.State = task.Running
	w.Db[t.ID] = &t
	
	fmt.Printf("Worker %s successfully started task %s (PID %d)\n", w.Name, t.ID, cmd.Process.Pid)

	// Wait for process to finish asynchronously
	go func() {
		err := cmd.Wait()
		if err != nil {
			fmt.Printf("Task %s failed: %v\n", t.ID, err)
			t.State = task.Failed
		} else {
			fmt.Printf("Task %s completed successfully\n", t.ID)
			t.State = task.Completed
		}
		w.Db[t.ID] = &t
	}()

	return nil
}
