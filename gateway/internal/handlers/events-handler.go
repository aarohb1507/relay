package handlers

import (
	"fmt"
	"net/http"
	"relay/gateway/internal/events"
)

func EventsHandler(w http.ResponseWriter, r *http.Request) {
	workflowID := r.URL.Query().Get("workflow_id")
	if workflowID == "" {
		http.Error(w, "workflow_id is required", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	client := events.Register(workflowID, w)

	fmt.Fprintf(w, "data: connected\n\n")
	flusher.Flush()

	select {
	case <-client.Done:
	case <-r.Context().Done():
	}

	events.Remove(workflowID)
}
