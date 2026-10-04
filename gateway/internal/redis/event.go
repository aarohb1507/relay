package redis

type Event struct {
	WorkflowID string         `json:"workflow_id"`
	Status     string         `json:"status"`
	Result     map[string]any `json:"result,omitempty"`
}
