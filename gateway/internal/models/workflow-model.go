package models

import "time"

type WorkflowTask struct {
	Goal   string         `json:"goal"`
	Params map[string]any `json:"params"`
}

type CreateWorkflowRequest struct {
	WorkflowID     string       `json:"workflow_id"`
	Task           WorkflowTask `json:"task"`
	IdempotencyKey string       `json:"idempotency_key"`
	MaxTokens      int          `json:"max_tokens"`
	SecretRef      string       `json:"secret_ref"`
}

type Workflow struct {
	ID             string       `json:"workflow_id"`
	Status         string       `json:"status"`
	CurrentStep    int          `json:"current_step"`
	MaxTokens      int          `json:"max_tokens"`
	Task           WorkflowTask `json:"task"`
	SecretRef      string       `json:"secret_ref,omitempty"`
	IdempotencyKey string       `json:"idempotency_key"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}
