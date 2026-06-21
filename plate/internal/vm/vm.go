package vm

import "time"

type Status string

const (
	StatusCreating Status = "creating"
	StatusStopped  Status = "stopped"
	StatusRunning  Status = "running"
	StatusError    Status = "error"
)

type Instance struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Plan      string    `json:"plan"`
	Image     string    `json:"image"`
	Status    Status    `json:"status"`
	Provider  string    `json:"provider"`
	BackendID string    `json:"backend_id,omitempty"`
	IPv4      string    `json:"ipv4,omitempty"`
	SSHPort   int       `json:"ssh_port,omitempty"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Name  string `json:"name"`
	Plan  string `json:"plan"`
	Image string `json:"image,omitempty"`
}
