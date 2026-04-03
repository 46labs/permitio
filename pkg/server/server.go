package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/46labs/permitio/pkg/config"
	"github.com/46labs/permitio/pkg/store"
)

type Server struct {
	cfg   *config.Config
	store *store.Store
}

func New(cfg *config.Config) *Server {
	st := store.New()
	st.Seed(cfg)
	st.Materialize()

	return &Server{
		cfg:   cfg,
		store: st,
	}
}

// NewWithStore creates a server with an existing store (for testing)
func NewWithStore(cfg *config.Config, st *store.Store) *Server {
	return &Server{
		cfg:   cfg,
		store: st,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// API key scope (SDK calls this first to resolve project/env context)
	mux.HandleFunc("/v2/api-key/scope", s.handleAPIKeyScope)

	// Schema endpoints: /v2/schema/{proj}/{env}/...
	mux.HandleFunc("/v2/schema/", s.routeSchema)

	// Facts endpoints: /v2/facts/{proj}/{env}/...
	mux.HandleFunc("/v2/facts/", s.routeFacts)

	// Health
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// PDP check endpoints
	mux.HandleFunc("/allowed", s.handleCheckImpl)
	mux.HandleFunc("/allowed/bulk", s.handleBulkCheckImpl)
	mux.HandleFunc("/allowed/all-tenants", s.handleAllTenantsCheckImpl)
	mux.HandleFunc("/user-permissions", s.handleUserPermissionsImpl)

	return corsMiddleware(logMiddleware(mux))
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	log.Printf("Starting permit.io mock on %s", addr)
	log.Printf("PDP API: POST /allowed")
	log.Printf("Management API: /v2/schema/... and /v2/facts/...")
	return http.ListenAndServe(addr, s.Handler())
}

// routeSchema handles all /v2/schema/{proj}/{env}/... requests
func (s *Server) routeSchema(w http.ResponseWriter, r *http.Request) {
	// Strip /v2/schema/{proj}/{env}/ prefix — segments: [proj, env, resource_type, ...]
	segs := extractPathSegments(r.URL.Path, "/v2/schema")
	if len(segs) < 3 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	// segs[0] = proj, segs[1] = env, segs[2:] = resource path
	rest := segs[2:]
	s.handleSchemaRoute(w, r, rest)
}

// routeFacts handles all /v2/facts/{proj}/{env}/... requests
func (s *Server) routeFacts(w http.ResponseWriter, r *http.Request) {
	segs := extractPathSegments(r.URL.Path, "/v2/facts")
	if len(segs) < 3 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	rest := segs[2:]
	s.handleFactsRoute(w, r, rest)
}

func (s *Server) handleSchemaRoute(w http.ResponseWriter, r *http.Request, rest []string) {
	if len(rest) == 0 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	switch rest[0] {
	case "resources":
		s.handleResources(w, r, rest[1:])
	case "roles":
		s.handleRoles(w, r, rest[1:])
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (s *Server) handleFactsRoute(w http.ResponseWriter, r *http.Request, rest []string) {
	if len(rest) == 0 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	switch rest[0] {
	case "tenants":
		s.handleTenants(w, r, rest[1:])
	case "users":
		s.handleUsers(w, r, rest[1:])
	case "resource_instances":
		s.handleResourceInstances(w, r, rest[1:])
	case "relationship_tuples":
		s.handleRelationshipTuples(w, r, rest[1:])
	case "role_assignments":
		s.handleRoleAssignments(w, r, rest[1:])
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}
