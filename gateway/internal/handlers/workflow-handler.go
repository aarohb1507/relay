package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"relay/gateway/internal/models"
	"relay/gateway/internal/services"
)

func WorkflowExecuteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request models.CreateWorkflowRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid workflow request", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.IdempotencyKey) == "" {
		http.Error(w, "idempotency_key is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.Task.Goal) == "" && strings.TrimSpace(request.Task.Prompt) == "" {
		http.Error(w, "task.goal or task.prompt is required", http.StatusBadRequest)
		return
	}

	workflow, err := services.CreateWorkflow(request)
	if err != nil {
		http.Error(w, "failed to create workflow", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(workflow)
}

func WorkflowHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/v1/workflows/")
	if id == "" || id == r.URL.Path {
		http.Error(w, "workflow id is required", http.StatusBadRequest)
		return
	}

	workflow, found, err := services.GetWorkflow(id)
	if err != nil {
		http.Error(w, "failed to get workflow", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workflow)
}
