package ipv1

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

var ErrNotFound = errors.New("ipv1: address not in registry")

type state struct {
	Addrs map[string]string `json:"addrs"`
	Next  int               `json:"next"`
}

type Registry struct {
	path string
	mu   sync.Mutex
}

func Open(path string) (*Registry, error) {
	if path == "" {
		path = ".ipv1/registry.json"
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	r := &Registry{path: path}
	if err := r.ensure(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Registry) Register(addr, endpoint string) error {
	if err := validateAddr(addr); err != nil {
		return err
	}
	if endpoint == "" {
		return errors.New("ipv1: empty endpoint")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	st, err := r.readLocked()
	if err != nil {
		return err
	}
	st.Addrs[addr] = endpoint
	return r.writeLocked(st)
}

func (r *Registry) Lookup(addr string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	st, err := r.readLocked()
	if err != nil {
		return "", err
	}
	ep, ok := st.Addrs[addr]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrNotFound, addr)
	}
	return ep, nil
}

// Allocate picks the next free numeric address and binds it to endpoint.
func (r *Registry) Allocate(endpoint string) (string, error) {
	if endpoint == "" {
		return "", errors.New("ipv1: empty endpoint")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	st, err := r.readLocked()
	if err != nil {
		return "", err
	}
	for {
		addr := strconv.Itoa(st.Next)
		st.Next++
		if _, used := st.Addrs[addr]; used {
			continue
		}
		st.Addrs[addr] = endpoint
		if err := r.writeLocked(st); err != nil {
			return "", err
		}
		return addr, nil
	}
}

func (r *Registry) List() (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	st, err := r.readLocked()
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(st.Addrs))
	for k, v := range st.Addrs {
		out[k] = v
	}
	return out, nil
}

func (r *Registry) ensure() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	st, err := r.readLocked()
	if err != nil {
		return err
	}
	if st.Next < 1 {
		st.Next = 1
	}
	if st.Addrs == nil {
		st.Addrs = map[string]string{}
	}
	return r.writeLocked(st)
}

func (r *Registry) readLocked() (state, error) {
	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return state{Addrs: map[string]string{}, Next: 1}, nil
		}
		return state{}, err
	}
	var st state
	if err := json.Unmarshal(data, &st); err != nil {
		return state{}, err
	}
	if st.Addrs == nil {
		st.Addrs = map[string]string{}
	}
	if st.Next < 1 {
		st.Next = 1
	}
	return st, nil
}

func (r *Registry) writeLocked(st state) error {
	if st.Addrs == nil {
		st.Addrs = map[string]string{}
	}
	if st.Next < 1 {
		st.Next = 1
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0o644)
}
