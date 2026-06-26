package handlers

import (
	"encoding/json"
	"net/http"
	"relay/gateway/internal/models"
	"relay/gateway/internal/services"
)


func JobHandler(w http.ResponseWriter, r *http.Request) {

	var job models.Job

	err:= json.NewDecoder(r.Body).Decode(&job)

	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	

	if job.Tool == "" {
		http.Error(w, "Tool Empty", http.StatusBadRequest)
		return
	}
	createdJob := services.CreateJob(job.Tool)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdJob)

}