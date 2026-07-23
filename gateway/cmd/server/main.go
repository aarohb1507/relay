package main

import (
	"fmt"
	"net/http"
	"relay/gateway/internal/db"
	"relay/gateway/internal/handlers"
	"relay/gateway/internal/redis"
)

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Welcome to Relay Aaroh!")
}

func main() {

	db.Connect()
	redis.Connect()

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/health", handlers.HealthHandler)
	http.HandleFunc("/jobs", handlers.JobHandler)
	http.HandleFunc("/events", handlers.EventsHandler)

	fmt.Println("Server running on port.")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Server error: ", err)
	}

}
