package ipv1

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestRegistryRegisterLookup(t *testing.T) {
	dir := t.TempDir()
	r, err := Open(filepath.Join(dir, "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Register("3", "127.0.0.1:9000"); err != nil {
		t.Fatal(err)
	}
	ep, err := r.Lookup("3")
	if err != nil {
		t.Fatal(err)
	}
	if ep != "127.0.0.1:9000" {
		t.Fatalf("lookup = %q", ep)
	}
	if _, err := r.Lookup("99"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing addr: %v", err)
	}
}

func TestRegistryAllocate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	a1, err := r.Allocate("127.0.0.1:9001")
	if err != nil {
		t.Fatal(err)
	}
	if a1 != "1" {
		t.Fatalf("first allocate = %q, want 1", a1)
	}
	a2, err := r.Allocate("127.0.0.1:9002")
	if err != nil {
		t.Fatal(err)
	}
	if a2 != "2" {
		t.Fatalf("second allocate = %q, want 2", a2)
	}

	r2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ep, err := r2.Lookup("1")
	if err != nil {
		t.Fatal(err)
	}
	if ep != "127.0.0.1:9001" {
		t.Fatalf("persisted endpoint = %q", ep)
	}
	a3, err := r2.Allocate("127.0.0.1:9003")
	if err != nil {
		t.Fatal(err)
	}
	if a3 != "3" {
		t.Fatalf("resume counter = %q, want 3", a3)
	}
}

func TestAllocateSkipsTaken(t *testing.T) {
	dir := t.TempDir()
	r, err := Open(filepath.Join(dir, "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Register("2", "127.0.0.1:9000"); err != nil {
		t.Fatal(err)
	}
	addr, err := r.Allocate("127.0.0.1:9001")
	if err != nil {
		t.Fatal(err)
	}
	if addr != "1" {
		t.Fatalf("allocate = %q, want 1", addr)
	}
}
