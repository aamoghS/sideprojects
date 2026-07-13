package ippool

import (
	"errors"
	"testing"
)

func TestPoolAssignRelease(t *testing.T) {
	dir := t.TempDir()
	p, err := Open(dir, "10.0.0.1,10.0.0.2")
	if err != nil {
		t.Fatal(err)
	}
	ip1, err := p.Assign("vm1")
	if err != nil {
		t.Fatal(err)
	}
	if ip1 != "10.0.0.1" {
		t.Fatalf("ip1 = %q", ip1)
	}
	ip2, err := p.Assign("vm2")
	if err != nil {
		t.Fatal(err)
	}
	if ip2 != "10.0.0.2" {
		t.Fatalf("ip2 = %q", ip2)
	}
	if _, err := p.Assign("vm3"); !errors.Is(err, ErrPoolExhausted) {
		t.Fatalf("expected pool exhausted, got %v", err)
	}
	p.Release("vm1")
	ip3, err := p.Assign("vm3")
	if err != nil {
		t.Fatal(err)
	}
	if ip3 != "10.0.0.1" {
		t.Fatalf("ip3 = %q", ip3)
	}
}

func TestAssignIdempotent(t *testing.T) {
	dir := t.TempDir()
	p, err := Open(dir, "10.0.0.5")
	if err != nil {
		t.Fatal(err)
	}
	ip1, err := p.Assign("vm1")
	if err != nil {
		t.Fatal(err)
	}
	ip2, err := p.Assign("vm1")
	if err != nil {
		t.Fatal(err)
	}
	if ip1 != ip2 {
		t.Fatalf("idempotent assign: %q != %q", ip1, ip2)
	}
}

func TestNoCollision(t *testing.T) {
	dir := t.TempDir()
	p, err := Open(dir, "10.0.0.1,10.0.0.2,10.0.0.3")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, id := range []string{"a", "b", "c"} {
		ip, err := p.Assign(id)
		if err != nil {
			t.Fatal(err)
		}
		if owner, ok := seen[ip]; ok {
			t.Fatalf("collision: %s and %s both got %s", owner, id, ip)
		}
		seen[ip] = id
	}
}

func TestParseCIDR(t *testing.T) {
	ips, err := parsePoolSpec("10.0.0.0/30")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"10.0.0.1", "10.0.0.2"}
	if len(ips) != len(want) {
		t.Fatalf("got %v, want %v", ips, want)
	}
	for i, ip := range ips {
		if ip != want[i] {
			t.Fatalf("[%d] = %q, want %q", i, ip, want[i])
		}
	}
}

func TestParseMixedSpec(t *testing.T) {
	ips, err := parsePoolSpec("203.0.113.10,10.0.0.0/31")
	if err != nil {
		t.Fatal(err)
	}
	if len(ips) != 3 {
		t.Fatalf("got %d ips: %v", len(ips), ips)
	}
	if ips[0] != "203.0.113.10" {
		t.Fatalf("first ip = %q", ips[0])
	}
}

func TestStatusCounts(t *testing.T) {
	dir := t.TempDir()
	p, err := Open(dir, "10.0.0.1,10.0.0.2,10.0.0.3")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = p.Assign("vm1")
	st, err := p.Status()
	if err != nil {
		t.Fatal(err)
	}
	if st.Total != 3 || st.Used != 1 || st.Free != 2 {
		t.Fatalf("status = %+v", st)
	}
}

func TestInvalidSpec(t *testing.T) {
	if _, err := parsePoolSpec("not-an-ip"); err == nil {
		t.Fatal("expected error for invalid ip")
	}
	if _, err := parsePoolSpec("10.0.0.0/33"); err == nil {
		t.Fatal("expected error for bad cidr")
	}
}

func TestPersistAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	p1, err := Open(dir, "10.0.0.10,10.0.0.11")
	if err != nil {
		t.Fatal(err)
	}
	ip, err := p1.Assign("persist-vm")
	if err != nil {
		t.Fatal(err)
	}
	p2, err := Open(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	st, err := p2.Status()
	if err != nil {
		t.Fatal(err)
	}
	if st.Assigned["persist-vm"] != ip {
		t.Fatalf("assignment lost after reopen: %+v", st.Assigned)
	}
	if len(st.Pool) != 2 {
		t.Fatalf("pool size = %d", len(st.Pool))
	}
}
