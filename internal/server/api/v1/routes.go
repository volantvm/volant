// Copyright (c) 2025 HYPR. PTE. LTD.

package v1

import (
	"net/http"

	"github.com/gorilla/mux"
)

// RegisterRoutes sets up all API v1 routes
func RegisterRoutes(r *mux.Router, handlers *Handlers) {
	api := r.PathPrefix("/api/v1").Subrouter()

	// NEW endpoints (preferred) - Using "stack" terminology
	api.HandleFunc("/stacks", handlers.ListStacks).Methods("GET")
	api.HandleFunc("/stacks", handlers.CreateStack).Methods("POST")
	api.HandleFunc("/stacks/{name}", handlers.GetStack).Methods("GET")
	api.HandleFunc("/stacks/{name}", handlers.UpdateStack).Methods("PUT", "PATCH")
	api.HandleFunc("/stacks/{name}", handlers.DeleteStack).Methods("DELETE")
	api.HandleFunc("/stacks/{name}/start", handlers.StartStack).Methods("POST")
	api.HandleFunc("/stacks/{name}/stop", handlers.StopStack).Methods("POST")
	api.HandleFunc("/stacks/{name}/restart", handlers.RestartStack).Methods("POST")
	api.HandleFunc("/stacks/{name}/logs", handlers.GetStackLogs).Methods("GET")
	api.HandleFunc("/stacks/{name}/status", handlers.GetStackStatus).Methods("GET")

	// LEGACY endpoints (deprecated, keep for backward compatibility)
	api.HandleFunc("/deployments", handlers.ListStacks).Methods("GET")
	api.HandleFunc("/deployments", handlers.CreateStack).Methods("POST")
	api.HandleFunc("/deployments/{name}", handlers.GetStack).Methods("GET")
	api.HandleFunc("/deployments/{name}", handlers.UpdateStack).Methods("PUT", "PATCH")
	api.HandleFunc("/deployments/{name}", handlers.DeleteStack).Methods("DELETE")

	// VM endpoints
	api.HandleFunc("/vms", handlers.ListVMs).Methods("GET")
	api.HandleFunc("/vms", handlers.CreateVM).Methods("POST")
	api.HandleFunc("/vms/{id}", handlers.GetVM).Methods("GET")
	api.HandleFunc("/vms/{id}", handlers.DeleteVM).Methods("DELETE")
	api.HandleFunc("/vms/{id}/start", handlers.StartVM).Methods("POST")
	api.HandleFunc("/vms/{id}/stop", handlers.StopVM).Methods("POST")
	api.HandleFunc("/vms/{id}/restart", handlers.RestartVM).Methods("POST")
	api.HandleFunc("/vms/{id}/console", handlers.GetVMConsole).Methods("GET")

	// Image endpoints
	api.HandleFunc("/images", handlers.ListImages).Methods("GET")
	api.HandleFunc("/images", handlers.ImportImage).Methods("POST")
	api.HandleFunc("/images/{name}", handlers.GetImage).Methods("GET")
	api.HandleFunc("/images/{name}", handlers.DeleteImage).Methods("DELETE")
	api.HandleFunc("/images/{name}/tag", handlers.TagImage).Methods("POST")

	// Volume endpoints
	api.HandleFunc("/volumes", handlers.ListVolumes).Methods("GET")
	api.HandleFunc("/volumes", handlers.CreateVolume).Methods("POST")
	api.HandleFunc("/volumes/{name}", handlers.GetVolume).Methods("GET")
	api.HandleFunc("/volumes/{name}", handlers.DeleteVolume).Methods("DELETE")

	// System endpoints
	api.HandleFunc("/system/info", handlers.GetSystemInfo).Methods("GET")
	api.HandleFunc("/system/version", handlers.GetVersion).Methods("GET")
	api.HandleFunc("/system/health", handlers.GetHealth).Methods("GET")

	// WebSocket endpoint for real-time updates
	api.HandleFunc("/ws", handlers.WebSocketHandler)

	// Add deprecation header middleware for legacy endpoints
	api.Use(deprecationMiddleware)
}

// deprecationMiddleware adds deprecation headers to legacy endpoints
func deprecationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if matched, _ := mux.CurrentRoute(r).GetPathTemplate(); matched != "" {
			if contains(matched, "deployment") {
				w.Header().Set("X-Deprecated", "true")
				w.Header().Set("X-Deprecation-Notice", "The /deployments endpoints are deprecated. Please use /stacks instead.")
			}
		}
		next.ServeHTTP(w, r)
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && s[len(s)-len(substr):] == substr || (len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}