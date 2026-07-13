package ippool

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var ErrPoolExhausted = errors.New("ip pool exhausted")

type state struct {
	Spec     string            `json:"spec,omitempty"`
	Pool     []string          `json:"pool"`
	Assigned map[string]string `json:"assigned"`
}

type Status struct {
	Spec     string            `json:"spec,omitempty"`
	Pool     []string          `json:"pool"`
	Assigned map[string]string `json:"assigned"`
	Total    int               `json:"total"`
	Used     int               `json:"used"`
	Free     int               `json:"free"`
}

type Pool struct {
	path string
	mu   sync.Mutex
}

func Open(dir string, envPool string) (*Pool, error) {
	if dir == "" {
		dir = ".plate"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	p := &Pool{path: filepath.Join(dir, "ip-pool.json")}
	if err := p.init(envPool); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Pool) init(envPool string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	st, err := p.readLocked()
	if err != nil {
		return err
	}
	if len(st.Pool) == 0 && strings.TrimSpace(envPool) != "" {
		pool, err := parsePoolSpec(envPool)
		if err != nil {
			return err
		}
		st.Spec = strings.TrimSpace(envPool)
		st.Pool = pool
		return p.writeLocked(st)
	}
	if st.Assigned == nil {
		st.Assigned = map[string]string{}
		return p.writeLocked(st)
	}
	return nil
}

func (p *Pool) Assign(vmID string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	st, err := p.readLocked()
	if err != nil {
		return "", err
	}
	if ip, ok := st.Assigned[vmID]; ok {
		return ip, nil
	}
	used := map[string]bool{}
	for _, ip := range st.Assigned {
		used[ip] = true
	}
	for _, ip := range st.Pool {
		if !used[ip] {
			st.Assigned[vmID] = ip
			if err := p.writeLocked(st); err != nil {
				return "", err
			}
			return ip, nil
		}
	}
	return "", ErrPoolExhausted
}

func (p *Pool) Release(vmID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	st, err := p.readLocked()
	if err != nil {
		return
	}
	delete(st.Assigned, vmID)
	_ = p.writeLocked(st)
}

func (p *Pool) List() ([]string, map[string]string, error) {
	st, err := p.Status()
	if err != nil {
		return nil, nil, err
	}
	return st.Pool, st.Assigned, nil
}

func (p *Pool) Status() (Status, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	st, err := p.readLocked()
	if err != nil {
		return Status{}, err
	}
	pool := append([]string(nil), st.Pool...)
	assigned := map[string]string{}
	for k, v := range st.Assigned {
		assigned[k] = v
	}
	used := len(assigned)
	total := len(pool)
	return Status{
		Spec:     st.Spec,
		Pool:     pool,
		Assigned: assigned,
		Total:    total,
		Used:     used,
		Free:     total - used,
	}, nil
}

func parsePoolSpec(spec string) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var ips []string
		var err error
		if strings.Contains(part, "/") {
			ips, err = expandCIDR(part)
		} else {
			ips, err = parseHost(part)
		}
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			if seen[ip] {
				continue
			}
			seen[ip] = true
			out = append(out, ip)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty ip pool spec")
	}
	return out, nil
}

func parseHost(s string) ([]string, error) {
	ip := net.ParseIP(strings.TrimSpace(s))
	if ip == nil {
		return nil, fmt.Errorf("invalid ip %q", s)
	}
	return []string{ip.String()}, nil
}

func expandCIDR(cidr string) ([]string, error) {
	ip, network, err := net.ParseCIDR(strings.TrimSpace(cidr))
	if err != nil {
		return nil, fmt.Errorf("invalid cidr %q: %w", cidr, err)
	}
	if ip.To4() == nil {
		return nil, fmt.Errorf("ipv6 cidr not supported: %q", cidr)
	}

	ones, bits := network.Mask.Size()
	if ones == bits {
		return []string{ip.String()}, nil
	}

	start := cloneIP(network.IP.To4())
	end := lastIP(network)

	skipEnds := ones <= 30
	var out []string
	for cur := cloneIP(start); ; incIP(cur) {
		if skipEnds && ipEqual(cur, start) {
			if ipEqual(cur, end) {
				break
			}
			continue
		}
		if skipEnds && ipEqual(cur, end) {
			break
		}
		out = append(out, cur.String())
		if ipEqual(cur, end) {
			break
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("cidr %q has no assignable hosts", cidr)
	}
	return out, nil
}

func cloneIP(ip net.IP) net.IP {
	out := make(net.IP, len(ip))
	copy(out, ip)
	return out
}

func lastIP(network *net.IPNet) net.IP {
	ip := cloneIP(network.IP.To4())
	for i := range ip {
		ip[i] |= ^network.Mask[i]
	}
	return ip
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] != 0 {
			return
		}
	}
}

func ipEqual(a, b net.IP) bool {
	return a.Equal(b)
}

func (p *Pool) readLocked() (state, error) {
	data, err := os.ReadFile(p.path)
	if err != nil {
		if os.IsNotExist(err) {
			return state{Assigned: map[string]string{}}, nil
		}
		return state{}, err
	}
	var st state
	if err := json.Unmarshal(data, &st); err != nil {
		return state{}, err
	}
	if st.Assigned == nil {
		st.Assigned = map[string]string{}
	}
	return st, nil
}

func (p *Pool) writeLocked(st state) error {
	if st.Assigned == nil {
		st.Assigned = map[string]string{}
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.path, data, 0o644)
}
