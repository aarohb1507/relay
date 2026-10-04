package events

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"relay/gateway/internal/redis"
)

type Client struct {
	Writer http.ResponseWriter
	Done   chan struct{}
}

var (
	Clients = make(map[string]*Client)
	mu      sync.Mutex
)

func Register(workflowID string, w http.ResponseWriter) *Client {
	mu.Lock()
	defer mu.Unlock()

	client := &Client{
		Writer: w,
		Done:   make(chan struct{}),
	}
	Clients[workflowID] = client
	return client
}

func Remove(workflowID string) {
	mu.Lock()
	defer mu.Unlock()

	delete(Clients, workflowID)
}

func Send(event redis.Event) {
	mu.Lock()
	client, ok := Clients[event.WorkflowID]
	mu.Unlock()

	if !ok {
		return
	}

	flusher, ok := client.Writer.(http.Flusher)
	if !ok {
		return
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	_, err = fmt.Fprintf(client.Writer, "data: %s\n\n", data)
	if err != nil {
		Remove(event.WorkflowID)
		return
	}

	flusher.Flush()

	if event.Status == "SUCCESS" {
		select {
		case <-client.Done:
		default:
			close(client.Done)
		}
		Remove(event.WorkflowID)
	}
}
