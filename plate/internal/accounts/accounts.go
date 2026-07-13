package accounts

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Account struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Token struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	Hash      string    `json:"hash"`
	Label     string    `json:"label,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type UsageRecord struct {
	ID        string     `json:"id"`
	AccountID string     `json:"account_id"`
	VMID      string     `json:"vm_id"`
	VMName    string     `json:"vm_name"`
	Plan      string     `json:"plan"`
	CreatedAt time.Time  `json:"created_at"`
	StoppedAt *time.Time `json:"stopped_at,omitempty"`
	Hours     float64    `json:"hours"`
}

type Store struct {
	dir string
	mu  sync.Mutex
}

func Open(dir string) (*Store, error) {
	if dir == "" {
		dir = ".plate"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

func (s *Store) CreateAccount(name string) (Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts, err := s.readAccounts()
	if err != nil {
		return Account{}, err
	}
	acc := Account{
		ID:        newID(),
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	accounts[acc.ID] = acc
	return acc, s.writeAccounts(accounts)
}

func (s *Store) ListAccounts() ([]Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts, err := s.readAccounts()
	if err != nil {
		return nil, err
	}
	out := make([]Account, 0, len(accounts))
	for _, a := range accounts {
		out = append(out, a)
	}
	return out, nil
}

func (s *Store) GetAccount(id string) (Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts, err := s.readAccounts()
	if err != nil {
		return Account{}, err
	}
	a, ok := accounts[id]
	if !ok {
		return Account{}, fmt.Errorf("account %q not found", id)
	}
	return a, nil
}

func (s *Store) CreateToken(accountID, label string) (plain string, tok Token, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts, err := s.readAccounts()
	if err != nil {
		return "", Token{}, err
	}
	if _, ok := accounts[accountID]; !ok {
		return "", Token{}, fmt.Errorf("account %q not found", accountID)
	}
	tokens, err := s.readTokens()
	if err != nil {
		return "", Token{}, err
	}
	plain, err = randomToken()
	if err != nil {
		return "", Token{}, err
	}
	tok = Token{
		ID:        newID(),
		AccountID: accountID,
		Hash:      hashToken(plain),
		Label:     label,
		CreatedAt: time.Now().UTC(),
	}
	tokens[tok.ID] = tok
	return plain, tok, s.writeTokens(tokens)
}

func (s *Store) Authenticate(plain string) (Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if plain == "" {
		return Account{}, fmt.Errorf("missing token")
	}
	tokens, err := s.readTokens()
	if err != nil {
		return Account{}, err
	}
	h := hashToken(plain)
	for _, t := range tokens {
		if t.Hash == h {
			accounts, err := s.readAccounts()
			if err != nil {
				return Account{}, err
			}
			a, ok := accounts[t.AccountID]
			if !ok {
				return Account{}, fmt.Errorf("account not found")
			}
			return a, nil
		}
	}
	return Account{}, fmt.Errorf("invalid token")
}

func (s *Store) HasTokens() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tokens, err := s.readTokens()
	if err != nil {
		return false, err
	}
	return len(tokens) > 0, nil
}

func (s *Store) RecordUsage(rec UsageRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ledger, err := s.readLedger()
	if err != nil {
		return err
	}
	if rec.ID == "" {
		rec.ID = newID()
	}
	ledger[rec.ID] = rec
	return s.writeLedger(ledger)
}

func (s *Store) UpdateUsageHours(vmID string, hours float64, stoppedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ledger, err := s.readLedger()
	if err != nil {
		return err
	}
	for id, rec := range ledger {
		if rec.VMID == vmID && rec.StoppedAt == nil {
			rec.Hours = hours
			t := stoppedAt.UTC()
			rec.StoppedAt = &t
			ledger[id] = rec
			return s.writeLedger(ledger)
		}
	}
	return nil
}

func (s *Store) ListUsage(accountID string) ([]UsageRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ledger, err := s.readLedger()
	if err != nil {
		return nil, err
	}
	out := make([]UsageRecord, 0)
	for _, rec := range ledger {
		if accountID == "" || rec.AccountID == accountID {
			out = append(out, rec)
		}
	}
	return out, nil
}

func (s *Store) accountsPath() string  { return filepath.Join(s.dir, "accounts.json") }
func (s *Store) tokensPath() string    { return filepath.Join(s.dir, "tokens.json") }
func (s *Store) ledgerPath() string    { return filepath.Join(s.dir, "billing.json") }

func (s *Store) readAccounts() (map[string]Account, error) {
	return readJSON[Account](s.accountsPath())
}

func (s *Store) writeAccounts(m map[string]Account) error {
	return writeJSON(s.accountsPath(), m)
}

func (s *Store) readTokens() (map[string]Token, error) {
	return readJSON[Token](s.tokensPath())
}

func (s *Store) writeTokens(m map[string]Token) error {
	return writeJSON(s.tokensPath(), m)
}

func (s *Store) readLedger() (map[string]UsageRecord, error) {
	return readJSON[UsageRecord](s.ledgerPath())
}

func (s *Store) writeLedger(m map[string]UsageRecord) error {
	return writeJSON(s.ledgerPath(), m)
}

func readJSON[T any](path string) (map[string]T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]T{}, nil
		}
		return nil, err
	}
	var m map[string]T
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]T{}
	}
	return m, nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func newID() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func randomToken() (string, error) {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "plt_" + hex.EncodeToString(b[:]), nil
}

func hashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
