package panel

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/aamoghS/sideprojects/plate/internal/control"
	"github.com/aamoghS/sideprojects/plate/internal/plans"
	"github.com/aamoghS/sideprojects/plate/internal/vm"
)

//go:embed templates/*
var files embed.FS

type Server struct {
	Plane *control.Plane
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /panel", s.handlePanel)
	mux.HandleFunc("POST /panel/create", s.handleCreate)
	mux.HandleFunc("POST /panel/{id}/start", s.handleStart)
	mux.HandleFunc("POST /panel/{id}/stop", s.handleStop)
}

func (s *Server) handlePanel(w http.ResponseWriter, r *http.Request) {
	vms, err := s.Plane.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl, err := template.ParseFS(files, "templates/panel.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct {
		VMs   any
		Plans []plans.Plan
	}{
		VMs:   vms,
		Plans: plans.List(),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, data)
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req := struct {
		Name     string
		Plan     string
		Hostname string
		SSHKeys  string
	}{
		Name:     r.FormValue("name"),
		Plan:     r.FormValue("plan"),
		Hostname: r.FormValue("hostname"),
		SSHKeys:  r.FormValue("ssh_keys"),
	}
	var keys []string
	for _, line := range splitLines(req.SSHKeys) {
		if line != "" {
			keys = append(keys, line)
		}
	}
	_, err := s.Plane.Create(r.Context(), vm.CreateRequest{
		Name: req.Name, Plan: req.Plan, Hostname: req.Hostname, SSHKeys: keys,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/panel", http.StatusSeeOther)
}

func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.Plane.Start(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/panel", http.StatusSeeOther)
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.Plane.Stop(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/panel", http.StatusSeeOther)
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			out = append(out, line)
			start = i + 1
		}
	}
	return out
}
