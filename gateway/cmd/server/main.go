package main

import (
	"fmt"
	"net/http"
    "relay/gateway/internal/handlers"
    "relay/gateway/internal/db"
)


func rootHandler(w http.ResponseWriter, r *http.Request){
	fmt.Fprintln(w, "Welcome to Relay Aaroh!")
}

func main() {
    db.Connect()
    http.HandleFunc("/", rootHandler)
    http.HandleFunc("/health", handlers.HealthHandler)
    http.HandleFunc("/jobs", handlers.JobHandler)

	fmt.Println("Server running on port.")
    
    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        fmt.Println("Server error: ", err)
    }
    
}