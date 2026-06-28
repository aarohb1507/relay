package handlers

import (
	"encoding/json"
	"net/http"
	"relay/gateway/internal/models"
	"relay/gateway/internal/services"
)


func JobHandler(w http.ResponseWriter, r *http.Request) {

	switch r.Method {

	case http.MethodPost:
		
			var job models.Job

			err := json.NewDecoder(r.Body).Decode(&job)

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
		
	case http.MethodGet:

			id := r.URL.Query().Get("id")

			job, found := services.GetJob(id)
	
			if !found {
				http.Error(w, "Job not found", http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(job)
	
	

	case http.MethodPath:
		
		id := r.URL.Query().Get("id")

		var request struct {
			Status string `json:"status"`
		}

		err := http.NewDecoder(r.Body).Decode(&request)

		if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
		}

		job, found := services.UpdateJobStatus(id, request.Status)

		if !found {
			http.Error(w, "Job not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(job)
	

	default:
		http.Error(w, "Method not Allowed", http.StatusMethodNotAllowed)	
		
	}

}

