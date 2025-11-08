// Copyright (c) 2025 HYPR. PTE. LTD.

package v1

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/volantvm/volant/internal/server/db"
)

// ListStacks handles GET /api/v1/stacks
func (h *Handlers) ListStacks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters for filtering
	filters := db.StackFilters{
		Name:   r.URL.Query().Get("name"),
		Status: r.URL.Query().Get("status"),
	}

	stacks, err := h.db.ListStacks(ctx, filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Default to empty array instead of null
	if stacks == nil {
		stacks = []*db.Stack{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stacks); err != nil {
		log.Printf("Error encoding stacks response: %v", err)
	}
}

// GetStack handles GET /api/v1/stacks/{name}
func (h *Handlers) GetStack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	name := vars["name"]

	stack, err := h.db.GetStack(ctx, name)
	if err != nil {
		if err.Error() == "stack not found: "+name {
			http.Error(w, "Stack not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stack); err != nil {
		log.Printf("Error encoding stack response: %v", err)
	}
}

// CreateStack handles POST /api/v1/stacks
func (h *Handlers) CreateStack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var stack db.Stack
	if err := json.NewDecoder(r.Body).Decode(&stack); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if stack.Name == "" {
		http.Error(w, "Stack name is required", http.StatusBadRequest)
		return
	}

	// Set default status if not provided
	if stack.Status == "" {
		stack.Status = db.StackStatusPending
	}

	if err := h.db.CreateStack(ctx, &stack); err != nil {
		if err.Error() == "UNIQUE constraint failed: stacks.name" {
			http.Error(w, "Stack with this name already exists", http.StatusConflict)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(stack); err != nil {
		log.Printf("Error encoding created stack response: %v", err)
	}
}

// UpdateStack handles PUT/PATCH /api/v1/stacks/{name}
func (h *Handlers) UpdateStack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	name := vars["name"]

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.db.UpdateStack(ctx, name, updates); err != nil {
		if err.Error() == "stack not found: "+name {
			http.Error(w, "Stack not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return updated stack
	stack, err := h.db.GetStack(ctx, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stack); err != nil {
		log.Printf("Error encoding updated stack response: %v", err)
	}
}

// DeleteStack handles DELETE /api/v1/stacks/{name}
func (h *Handlers) DeleteStack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	name := vars["name"]

	if err := h.db.DeleteStack(ctx, name); err != nil {
		if err.Error() == "stack not found: "+name {
			http.Error(w, "Stack not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// StartStack handles POST /api/v1/stacks/{name}/start
func (h *Handlers) StartStack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	name := vars["name"]

	// Update status to running
	updates := map[string]interface{}{
		"status": db.StackStatusRunning,
	}

	if err := h.db.UpdateStack(ctx, name, updates); err != nil {
		if err.Error() == "stack not found: "+name {
			http.Error(w, "Stack not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// TODO: Actually start the VMs in the stack

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Stack started successfully",
		"status":  db.StackStatusRunning,
	})
}

// StopStack handles POST /api/v1/stacks/{name}/stop
func (h *Handlers) StopStack(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	name := vars["name"]

	// Update status to stopped
	updates := map[string]interface{}{
		"status": db.StackStatusStopped,
	}

	if err := h.db.UpdateStack(ctx, name, updates); err != nil {
		if err.Error() == "stack not found: "+name {
			http.Error(w, "Stack not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// TODO: Actually stop the VMs in the stack

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Stack stopped successfully",
		"status":  db.StackStatusStopped,
	})
}

// RestartStack handles POST /api/v1/stacks/{name}/restart
func (h *Handlers) RestartStack(w http.ResponseWriter, r *http.Request) {
	// First stop, then start
	h.StopStack(w, r)
	if w.Header().Get("Status") == "404" {
		return // Stack not found
	}
	h.StartStack(w, r)
}

// GetStackLogs handles GET /api/v1/stacks/{name}/logs
func (h *Handlers) GetStackLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	// TODO: Implement log aggregation from stack VMs

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"stack": name,
		"logs":  []string{"Log aggregation not yet implemented"},
	})
}

// GetStackStatus handles GET /api/v1/stacks/{name}/status
func (h *Handlers) GetStackStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	name := vars["name"]

	stack, err := h.db.GetStack(ctx, name)
	if err != nil {
		if err.Error() == "stack not found: "+name {
			http.Error(w, "Stack not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Get VMs associated with this stack
	vms, err := h.db.GetStackVMs(ctx, stack.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":    stack.Name,
		"status":  stack.Status,
		"vm_count": len(vms),
		"vms":     vms,
	})
}