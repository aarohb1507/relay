package handlers

import (
	"encoding/json"
	"net/http"
	"relay/gateway/internal/models"
)


func JobHandler(w http.ResponseWriter, r *http.Request) {
	var job models.Job
	err := json.NewDecoder(r.Body).Decode(&job)

	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	job.ID = "job-1"
	job.Status = "QUEUED"

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)

}