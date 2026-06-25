package handlers 

import (
	"encoding/json"
	"net/http"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {

	response:= map[string]string{
		"service": "relay-gateway",
		"health": "healthy",
	}
	w.Header().Set("Content-Type", "Application/json")
	
	json.NewEncoder(w).Encode(response)
}