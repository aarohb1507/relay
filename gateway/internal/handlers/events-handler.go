package handlers

import (
	"fmt"
	"net/http"
	"relay/gateway/internal/events"
)

func EventsHandler(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("id")
	events.Clients[id] = w

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "data: connected\n\n")
	flusher.Flush()

	select {}
}
