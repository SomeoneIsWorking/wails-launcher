package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"wails-launcher/pkg/config"
	"wails-launcher/pkg/group"
	"wails-launcher/pkg/process"
	"wails-launcher/pkg/service"
)

// mockApp is the AppInterface implementation used by service.NewService;
// it discards all frontend events.
type mockApp struct{}

func (m *mockApp) EmitToFrontend(_ string, _ string, _ interface{}) {}

// newTestApp builds a minimal App wired up with a fake service, without
// starting the wails runtime or the HTTP server.
func newTestApp() *App {
	cfg := &config.Config{Groups: make(map[string]config.GroupConfig)}
	app := &App{
		services: make(map[string]*service.Service),
		groups:   group.NewManager(cfg.Groups),
		config:   cfg,
	}
	return app
}

// addFakeService injects a service with a known ID directly into the app's
// service map, bypassing the full config/group machinery.
func addFakeService(a *App, id string, cfg config.ServiceConfig) {
	svc := service.NewService(id, cfg, config.ServiceEnv{}, &mockApp{})
	a.mu.Lock()
	a.services[id] = svc
	a.mu.Unlock()
}

// do executes a request against the handler mux built from the app.
func do(a *App, method, path string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/services", a.handleListServices)
	mux.HandleFunc("GET /api/services/{id}", a.handleGetService)
	mux.HandleFunc("POST /api/services/{id}/start", a.handleStartService)
	mux.HandleFunc("POST /api/services/{id}/stop", a.handleStopService)
	mux.HandleFunc("POST /api/services/{id}/restart", a.handleRestartService)

	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// ── GET /api/services ─────────────────────────────────────────────────────

func TestHandleListServices_Empty(t *testing.T) {
	a := newTestApp()
	rec := do(a, http.MethodGet, "/api/services")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body []serviceStatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body) != 0 {
		t.Errorf("expected empty list, got %d item(s)", len(body))
	}
}

func TestHandleListServices_WithService(t *testing.T) {
	a := newTestApp()
	addFakeService(a, "svc1", config.ServiceConfig{Name: "My API", Type: "dotnet"})

	rec := do(a, http.MethodGet, "/api/services")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body []serviceStatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body) != 1 {
		t.Fatalf("expected 1 service, got %d", len(body))
	}
	svc := body[0]
	if svc.ID != "svc1" {
		t.Errorf("id = %q, want svc1", svc.ID)
	}
	if svc.Name != "My API" {
		t.Errorf("name = %q, want 'My API'", svc.Name)
	}
	if svc.Status != process.Stopped {
		t.Errorf("status = %q, want stopped", svc.Status)
	}
	if svc.URL != nil {
		t.Errorf("url should be omitted when nil, got %q", *svc.URL)
	}
	if svc.Type != "dotnet" {
		t.Errorf("type = %q, want dotnet", svc.Type)
	}
}

// ── GET /api/services/{id} ───────────────────────────────────────────────

func TestHandleGetService_Found(t *testing.T) {
	a := newTestApp()
	addFakeService(a, "abc", config.ServiceConfig{Name: "Worker", Type: "npm"})

	rec := do(a, http.MethodGet, "/api/services/abc")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body serviceStatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.ID != "abc" {
		t.Errorf("id = %q, want abc", body.ID)
	}
	if body.Type != "npm" {
		t.Errorf("type = %q, want npm", body.Type)
	}
}

func TestHandleGetService_NotFound(t *testing.T) {
	a := newTestApp()
	rec := do(a, http.MethodGet, "/api/services/nope")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body) //nolint
	if body["error"] == "" {
		t.Errorf("expected non-empty error field, got %v", body)
	}
}

// ── start / stop / restart error paths ───────────────────────────────────

func TestHandleStartService_NotFound(t *testing.T) {
	a := newTestApp()
	rec := do(a, http.MethodPost, "/api/services/ghost/start")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleStopService_NotFound(t *testing.T) {
	a := newTestApp()
	rec := do(a, http.MethodPost, "/api/services/ghost/stop")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleRestartService_NotFound(t *testing.T) {
	a := newTestApp()
	rec := do(a, http.MethodPost, "/api/services/ghost/restart")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// ── Content-Type ──────────────────────────────────────────────────────────

func TestHandlers_ContentTypeJSON(t *testing.T) {
	a := newTestApp()
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/services"},
		{http.MethodGet, "/api/services/x"},
		{http.MethodPost, "/api/services/x/start"},
	} {
		rec := do(a, tc.method, tc.path)
		ct := rec.Header().Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("%s %s: Content-Type = %q, want application/json", tc.method, tc.path, ct)
		}
	}
}
