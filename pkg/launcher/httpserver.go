package launcher

import (
	"context"
	"encoding/json"
	"net/http"

	"wails-launcher/pkg/config"
	"wails-launcher/pkg/process"
)

// HTTPListenAddr is where the control API listens. Local only, on purpose.
const HTTPListenAddr = "127.0.0.1:9901"

// serviceStatusResponse is the JSON shape returned by the HTTP API.
// It omits the full log buffer that ServiceInfo carries.
type serviceStatusResponse struct {
	ID     string                `json:"id"`
	Name   string                `json:"name"`
	Status process.ServiceStatus `json:"status"`
	URL    *string               `json:"url,omitempty"`
	Type   string                `json:"type"`
}

// startHTTPServer registers routes and starts listening in the background.
// The server is shut down when ctx is cancelled.
func (a *App) startHTTPServer(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/services", a.handleListServices)
	mux.HandleFunc("GET /api/services/{id}", a.handleGetService)
	mux.HandleFunc("GET /api/services/{id}/logs", a.handleGetServiceLogs)
	mux.HandleFunc("POST /api/services/{id}/start", a.handleStartService)
	mux.HandleFunc("POST /api/services/{id}/stop", a.handleStopService)
	mux.HandleFunc("POST /api/services/{id}/restart", a.handleRestartService)
	mux.HandleFunc("POST /api/services", a.handleAddService)

	srv := &http.Server{Addr: HTTPListenAddr, Handler: mux}
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

// addServiceRequest is the body of POST /api/services. Group is the group's
// name; the service inherits that group's environment, as it would when added
// from the UI.
type addServiceRequest struct {
	Group   string            `json:"group"`
	Name    string            `json:"name"`
	Path    string            `json:"path"`
	Type    string            `json:"type"`
	Profile string            `json:"profile"`
	Env     config.ServiceEnv `json:"env"`
}

// POST /api/services
// Adds a service to an existing group and saves the config. A service with the
// same name already in any group is refused rather than duplicated.
func (a *App) handleAddService(w http.ResponseWriter, r *http.Request) {
	var req addServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if req.Group == "" || req.Name == "" || req.Path == "" || req.Type == "" {
		writeError(w, http.StatusBadRequest, "group, name, path and type are required")
		return
	}
	if req.Env == nil {
		req.Env = config.ServiceEnv{}
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	for id, srv := range a.services {
		if srv.GetInfo().Name == req.Name {
			writeError(w, http.StatusConflict, "a service named "+req.Name+" already exists ("+id+")")
			return
		}
	}
	groupID := ""
	for id, grp := range a.groups.GetGroups() {
		if grp.Name == req.Group {
			groupID = id
			break
		}
	}
	if groupID == "" {
		writeError(w, http.StatusNotFound, "group not found: "+req.Group)
		return
	}

	id := a.AddServiceToGroup(groupID, config.ServiceConfig{
		Name: req.Name, Path: req.Path, Env: req.Env, Type: req.Type, Profile: req.Profile,
	})
	srv, exists := a.services[id]
	if !exists {
		writeError(w, http.StatusInternalServerError, "service was saved but not created")
		return
	}
	writeJSON(w, http.StatusCreated, toStatusResponse(id, srv.GetInfo()))
}
