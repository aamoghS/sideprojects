package vm

import "time"

type Status string

const (
	StatusCreating Status = "creating"
	StatusStopped  Status = "stopped"
	StatusRunning  Status = "running"
	StatusError    Status = "error"
)

type FirewallRule struct {
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
	Label    string `json:"label,omitempty"`
}

type Snapshot struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Health struct {
	OK        bool      `json:"ok"`
	Detail    string    `json:"detail,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

type Instance struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Plan       string         `json:"plan"`
	Image      string         `json:"image"`
	Status     Status         `json:"status"`
	Provider   string         `json:"provider"`
	BackendID  string         `json:"backend_id,omitempty"`
	IPv4       string         `json:"ipv4,omitempty"`
	PublicIPv4 string         `json:"public_ipv4,omitempty"`
	IPv1Addr   string         `json:"ipv1_addr,omitempty"`
	SSHPort    int            `json:"ssh_port,omitempty"`
	Hostname   string         `json:"hostname,omitempty"`
	Firewall   []FirewallRule `json:"firewall,omitempty"`
	SSHKeys    []string       `json:"ssh_keys,omitempty"`
	Snapshots  []Snapshot     `json:"snapshots,omitempty"`
	Health     *Health        `json:"health,omitempty"`
	AccountID  string         `json:"account_id,omitempty"`
	StartedAt  *time.Time     `json:"started_at,omitempty"`
	Error      string         `json:"error,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type CreateRequest struct {
	Name      string         `json:"name"`
	Plan      string         `json:"plan"`
	Image     string         `json:"image,omitempty"`
	SSHKeys   []string       `json:"ssh_keys,omitempty"`
	Hostname  string         `json:"hostname,omitempty"`
	Firewall  []FirewallRule `json:"firewall,omitempty"`
	AccountID string         `json:"account_id,omitempty"`
}

func DefaultFirewall() []FirewallRule {
	return []FirewallRule{
		{Protocol: "tcp", Port: 22, Label: "ssh"},
		{Protocol: "tcp", Port: 80, Label: "http"},
		{Protocol: "tcp", Port: 443, Label: "https"},
	}
}
