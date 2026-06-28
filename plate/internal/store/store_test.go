package store

import (
	"testing"

	"github.com/aamoghS/sideprojects/plate/internal/vm"
)

func TestStorePutGet(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	inst := vm.Instance{ID: "abc123", Name: "web-1", Plan: "small", Status: vm.StatusRunning}
	if err := st.Put(inst); err != nil {
		t.Fatal(err)
	}

	got, err := st.Get("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "web-1" {
		t.Fatalf("name = %q", got.Name)
	}

	list, err := st.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("list len = %d, err = %v", len(list), err)
	}

	if err := st.Delete("abc123"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Get("abc123"); err == nil {
		t.Fatal("expected not found after delete")
	}
}
