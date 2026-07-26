package events

import (
	"encoding/json"
	"fmt"
	"net/http"

	"relay/gateway/internal/redis"
)

var Clients = make(map[string]http.ResponseWriter)

func Register(jobID string, w http.ResponseWriter) {

	Clients[jobID] = w

}

func Remove(jobID string) {

	delete(Clients, jobID)

}

func Send(event redis.Event) {

	w, ok := Clients[event.JobID]
	if !ok {
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	if err != nil {
		Remove(event.JobID)
		return
	}

	flusher.Flush()
}
