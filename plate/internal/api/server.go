package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"plate/internal/control"
	"plate/internal/plans"
	"plate/internal/vm"
)

type Server struct {
	Plane *control.Plane
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/plans", s.handlePlans)
	mux.HandleFunc("GET /v1/vms", s.handleList)
	mux.HandleFunc("POST /v1/vms", s.handleCreate)
	mux.HandleFunc("GET /v1/vms/{id}", s.handleGet)
	mux.HandleFunc("POST /v1/vms/{id}/start", s.handleStart)
	mux.HandleFunc("POST /v1/vms/{id}/stop", s.handleStop)
	mux.HandleFunc("DELETE /v1/vms/{id}", s.handleDelete)
	return mux
}

func (s *Server) handlePlans(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, plans.List())
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

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
