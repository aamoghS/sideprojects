package accounts

import "testing"

func TestAccountTokenAuth(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	acc, err := st.CreateAccount("test")
	if err != nil {
		t.Fatal(err)
	}
	plain, _, err := st.CreateToken(acc.ID, "cli")
	if err != nil {
		t.Fatal(err)
	}
	got, err := st.Authenticate(plain)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != acc.ID {
		t.Fatalf("account id = %q", got.ID)
	}
	if _, err := st.Authenticate("bad"); err == nil {
		t.Fatal("expected invalid token")
	}
}

func TestUsageLedger(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	acc, _ := st.CreateAccount("bill")
	if err := st.RecordUsage(UsageRecord{AccountID: acc.ID, VMID: "v1", VMName: "web", Plan: "small"}); err != nil {
		t.Fatal(err)
	}
	recs, err := st.ListUsage(acc.ID)
	if err != nil || len(recs) != 1 {
		t.Fatalf("len = %d err = %v", len(recs), err)
	}
}
