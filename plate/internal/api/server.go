package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aamoghS/sideprojects/plate/internal/accounts"
	"github.com/aamoghS/sideprojects/plate/internal/control"
	"github.com/aamoghS/sideprojects/plate/internal/panel"
	"github.com/aamoghS/sideprojects/plate/internal/plans"
	"github.com/aamoghS/sideprojects/plate/internal/vm"
)

type Server struct {
	Plane    *control.Plane
	Accounts *accounts.Store
	Panel    *panel.Server
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.Panel.Register(mux)

	mux.HandleFunc("GET /v1/plans", s.handlePlans)
	mux.HandleFunc("GET /v1/ippool", s.auth(s.handleIPPool))
	mux.HandleFunc("GET /v1/ip-pool", s.auth(s.handleIPPool))
	mux.HandleFunc("GET /v1/accounts", s.auth(s.handleListAccounts))
	mux.HandleFunc("POST /v1/accounts", s.handleCreateAccount)
	mux.HandleFunc("POST /v1/accounts/{id}/tokens", s.handleCreateToken)
	mux.HandleFunc("GET /v1/billing", s.auth(s.handleBilling))
	mux.HandleFunc("GET /v1/vms", s.auth(s.handleList))
	mux.HandleFunc("POST /v1/vms", s.auth(s.handleCreate))
	mux.HandleFunc("GET /v1/vms/{id}", s.auth(s.handleGet))
	mux.HandleFunc("POST /v1/vms/{id}/start", s.auth(s.handleStart))
	mux.HandleFunc("POST /v1/vms/{id}/stop", s.auth(s.handleStop))
	mux.HandleFunc("DELETE /v1/vms/{id}", s.auth(s.handleDelete))
	mux.HandleFunc("PUT /v1/vms/{id}/firewall", s.auth(s.handleFirewall))
	mux.HandleFunc("PUT /v1/vms/{id}/hostname", s.auth(s.handleHostname))
	mux.HandleFunc("GET /v1/vms/{id}/snapshots", s.auth(s.handleListSnapshots))
	mux.HandleFunc("POST /v1/vms/{id}/snapshots", s.auth(s.handleCreateSnapshot))
	mux.HandleFunc("POST /v1/vms/{id}/snapshots/{snapId}/restore", s.auth(s.handleRestoreSnapshot))

	return mux
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.Accounts == nil {
			next(w, r)
			return
		}
		has, err := s.Accounts.HasTokens()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if !has {
			next(w, r)
			return
		}
		token := bearerToken(r)
		acc, err := s.Accounts.Authenticate(token)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, err)
			return
		}
		r.Header.Set("X-Plate-Account-ID", acc.ID)
		next(w, r)
	}
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return strings.TrimSpace(r.Header.Get("X-Plate-Token"))
}

func (s *Server) handlePlans(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, plans.List())
}

func (s *Server) handleIPPool(w http.ResponseWriter, r *http.Request) {
	st, err := s.Plane.IPPoolStatus()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, st)
}

func (s *Server) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	accs, err := s.Accounts.ListAccounts()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, accs)
}

func (s *Server) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeErr(w, http.StatusBadRequest, errNameRequired())
		return
	}
	acc, err := s.Accounts.CreateAccount(strings.TrimSpace(req.Name))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, acc)
}

func errNameRequired() error {
	return &jsonErr{"name is required"}
}

type jsonErr struct{ msg string }

func (e *jsonErr) Error() string { return e.msg }

func (s *Server) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	has, err := s.Accounts.HasTokens()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if has {
		if _, err := s.Accounts.Authenticate(bearerToken(r)); err != nil {
			writeErr(w, http.StatusUnauthorized, err)
			return
		}
	}
	id := r.PathValue("id")
	var req struct {
		Label string `json:"label"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	plain, tok, err := s.Accounts.CreateToken(id, req.Label)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, map[string]any{"token": plain, "id": tok.ID, "account_id": tok.AccountID})
}

func (s *Server) handleBilling(w http.ResponseWriter, r *http.Request) {
	accountID := r.URL.Query().Get("account_id")
	if accountID == "" {
		accountID = r.Header.Get("X-Plate-Account-ID")
	}
	recs, err := s.Accounts.ListUsage(accountID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, recs)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	items, err := s.Plane.List(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, items)
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req vm.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if req.AccountID == "" {
		req.AccountID = r.Header.Get("X-Plate-Account-ID")
	}
	inst, err := s.Plane.Create(r.Context(), req)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, inst)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inst, err := s.Plane.Get(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, inst)
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inst, err := s.Plane.Start(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, inst)
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	inst, err := s.Plane.Stop(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, inst)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.Plane.Delete(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeErr(w, http.StatusNotFound, err)
			return
		}
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleFirewall(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Rules []vm.FirewallRule `json:"rules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	inst, err := s.Plane.UpdateFirewall(r.Context(), id, req.Rules)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, inst)
}

func (s *Server) handleHostname(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Hostname string `json:"hostname"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	inst, err := s.Plane.UpdateHostname(r.Context(), id, req.Hostname)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, inst)
}

func (s *Server) handleListSnapshots(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	snaps, err := s.Plane.ListSnapshots(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, snaps)
}

func (s *Server) handleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	snap, err := s.Plane.CreateSnapshot(r.Context(), id, req.Name)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, snap)
}

func (s *Server) handleRestoreSnapshot(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	snapID := r.PathValue("snapId")
	inst, err := s.Plane.RestoreSnapshot(r.Context(), id, snapID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, inst)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
