package worker

import (
	"context"
	"fmt"

	"github.com/aamoghS/sideprojects/movie/internal/orchestrator/task"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
)

// Worker represents a node in our orchestrator that can run tasks.
type Worker struct {
	Name      string
	Queue     chan task.Task
	Db        map[uuid.UUID]*task.Task
	TaskCount int
}

// NewWorker initializes a new Worker.
func NewWorker(name string) *Worker {
	return &Worker{
		Name:  name,
		Queue: make(chan task.Task, 10),
		Db:    make(map[uuid.UUID]*task.Task),
	}
}

// RunTask starts a task using Docker.
func (w *Worker) RunTask(t task.Task) error {
	w.Db[t.ID] = &t
	w.TaskCount++

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}

	ctx := context.Background()

	// Pull the image (simplified)
	_, err = cli.ImagePull(ctx, t.Image, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	// Create the container
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: t.Image,
	}, nil, nil, nil, t.Name)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Start the container
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	t.State = task.Running
	w.Db[t.ID] = &t

	fmt.Printf("Worker %s successfully started task %s (container %s)\n", w.Name, t.ID, resp.ID)
	return nil
}
