package main

import (
	"fmt"
	"net/http"
    "encoding/json"
    "relay/gateway/internal/handlers"
)


func rootHandler(w http.ResponseWriter, r *http.Request){
	fmt.Fprintln(w, "Welcome to Relay Aaroh!")
}

func main() {
    http.HandleFunc("/", rootHandler)
    http.HandleFunc("/health", handlers.healthHandler)

	fmt.Println("Server running on port.")
    
    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        fmt.Println("Server error: ", err)
    }
}