package task

import (
	"time"
)

type State int

const (
	Pending State = iota
	Scheduled
	Running
	Completed
	Failed
)

// Task represents a unit of work (a container) to be scheduled and run.
type Task struct {
	ID            string
	Name          string
	State         State
	Image         string
	Memory        int64
	Disk          int64
	ExposedPorts  map[string]struct{}
	PortBindings  map[string]string
	RestartPolicy string
	StartTime     time.Time
	FinishTime    time.Time
}

// TaskEvent represents a state change for a task.
type TaskEvent struct {
	ID        string
	State     State
	Timestamp time.Time
	Task      Task
}
