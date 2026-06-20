package handlers 

import (
	"fmt",
	"encoding/json"
	"net/http"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {

	response:= map[string]string{
		"service": "relay-gateway",
		"health": "healthy"
	}
	w.Header().Set("Content-Type", "Application/json")
	
	json.NewEncoder(w).Encode(response)
	
}