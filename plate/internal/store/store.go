package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"plate/internal/vm"
)

type Store struct {
	path string
	mu   sync.Mutex
}

func Open(dir string) (*Store, error) {
	if dir == "" {
		dir = ".plate"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{path: filepath.Join(dir, "vms.json")}, nil
}

func (s *Store) List() ([]vm.Instance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.read()
	if err != nil {
		return nil, err
	}
	out := make([]vm.Instance, 0, len(all))
	for _, v := range all {
		out = append(out, v)
	}
	return out, nil
}

func (s *Store) Get(id string) (vm.Instance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.read()
	if err != nil {
		return vm.Instance{}, err
	}
	v, ok := all[id]
	if !ok {
		return vm.Instance{}, fmt.Errorf("vm %q not found", id)
	}
	return v, nil
}

func (s *Store) Put(v vm.Instance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.read()
	if err != nil {
		return err
	}
	all[v.ID] = v
	return s.write(all)
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.read()
	if err != nil {
		return err
	}
	delete(all, id)
	return s.write(all)
}

func (s *Store) read() (map[string]vm.Instance, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]vm.Instance{}, nil
		}
		return nil, err
	}
	var all map[string]vm.Instance
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, err
	}
	if all == nil {
		all = map[string]vm.Instance{}
	}
	return all, nil
}

func (s *Store) write(all map[string]vm.Instance) error {
	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}
