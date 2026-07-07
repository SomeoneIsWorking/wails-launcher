package main

import (
	"context"
	"encoding/json"
	"net/http"

	"wails-launcher/pkg/process"
)

const httpListenAddr = "127.0.0.1:9901"

// serviceStatusResponse is the JSON shape returned by the HTTP API.
// It omits the full log buffer that ServiceInfo carries.
type serviceStatusResponse struct {
	ID     string               `json:"id"`
	Name   string               `json:"name"`
	Status process.ServiceStatus `json:"status"`
	URL    *string              `json:"url,omitempty"`
	Type   string               `json:"type"`
}

// startHTTPServer registers routes and starts listening in the background.
// The server is shut down when ctx is cancelled (wails OnShutdown).
func (a *App) startHTTPServer(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/services", a.handleListServices)
	mux.HandleFunc("GET /api/services/{id}", a.handleGetService)
	mux.HandleFunc("GET /api/services/{id}/logs", a.handleGetServiceLogs)
	mux.HandleFunc("POST /api/services/{id}/start", a.handleStartService)
	mux.HandleFunc("POST /api/services/{id}/stop", a.handleStopService)
	mux.HandleFunc("POST /api/services/{id}/restart", a.handleRestartService)

	srv := &http.Server{Addr: httpListenAddr, Handler: mux}
	a.httpServer = srv

	go srv.ListenAndServe() //nolint:errcheck

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background()) //nolint:errcheck
	}()
}

// writeJSON writes v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// writeError writes {"error": msg} with the given status code.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// toStatusResponse converts a ServiceInfo into the HTTP response shape.
func toStatusResponse(id string, info ServiceInfo) serviceStatusResponse {
	return serviceStatusResponse{
		ID:     id,
		Name:   info.Name,
		Status: info.Status,
		URL:    info.URL,
		Type:   info.Type,
	}
}

// GET /api/services
func (a *App) handleListServices(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]serviceStatusResponse, 0, len(a.services))
	for id, srv := range a.services {
		result = append(result, toStatusResponse(id, srv.GetInfo()))
	}
	writeJSON(w, http.StatusOK, result)
}

// GET /api/services/{id}
func (a *App) handleGetService(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	a.mu.RLock()
	srv, exists := a.services[id]
	a.mu.RUnlock()
	if !exists {
		writeError(w, http.StatusNotFound, "service not found")
		return
	}
	writeJSON(w, http.StatusOK, toStatusResponse(id, srv.GetInfo()))
}

// GET /api/services/{id}/logs
// Returns the captured log buffer for a service (omitted from the status
// responses). Useful for debugging startup/port/TLS issues from the CLI.
func (a *App) handleGetServiceLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	a.mu.RLock()
	srv, exists := a.services[id]
	a.mu.RUnlock()
	if !exists {
		writeError(w, http.StatusNotFound, "service not found")
		return
	}
	info := srv.GetInfo()
	writeJSON(w, http.StatusOK, map[string]any{
		"id":   id,
		"name": info.Name,
		"logs": info.Logs,
	})
}

// POST /api/services/{id}/start[?no-build=true]
func (a *App) handleStartService(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var err error
	if r.URL.Query().Get("no-build") == "true" {
		err = a.StartServiceWithoutBuild(id)
	} else {
		err = a.StartService(id)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "starting"})
}

// POST /api/services/{id}/stop
func (a *App) handleStopService(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.StopService(id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

// POST /api/services/{id}/restart[?no-build=true]
func (a *App) handleRestartService(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.StopService(id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var err error
	if r.URL.Query().Get("no-build") == "true" {
		err = a.StartServiceWithoutBuild(id)
	} else {
		err = a.StartService(id)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "starting"})
}
