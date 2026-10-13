package health

import (
	"encoding/json"
	"net/http"
	"runtime"
)

// RegisterRoutes mounts health endpoints on the given mux:
//
//	GET /health          — shallow check (service metadata)
//	GET /health/live     — liveness probe, always 200
//	GET /health/ready    — readiness probe (runs all checkers)
//	GET /health/detailed — same as /ready but always returns full detail
func RegisterRoutes(mux *http.ServeMux, serviceName, version string, svc *Service) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			KeyStatus:      "ok",
			"service_name": serviceName,
			"version":      version,
			"go_version":   runtime.Version(),
		})
	})

	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{KeyStatus: "ok"})
	})

	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		result, err := svc.CheckAll(r.Context())
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
				KeyStatus: StatusUnhealthy,
				"error":   err.Error(),
			})
			return
		}
		if s, _ := result[KeyStatus].(string); s != StatusHealthy {
			writeJSON(w, http.StatusServiceUnavailable, result)
			return
		}
		writeJSON(w, http.StatusOK, result)
	})

	mux.HandleFunc("GET /health/detailed", func(w http.ResponseWriter, r *http.Request) {
		result, err := svc.CheckAll(r.Context())
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
				KeyStatus: StatusUnhealthy,
				"error":   err.Error(),
			})
			return
		}
		if s, _ := result[KeyStatus].(string); s != StatusHealthy {
			writeJSON(w, http.StatusServiceUnavailable, result)
			return
		}
		writeJSON(w, http.StatusOK, result)
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
