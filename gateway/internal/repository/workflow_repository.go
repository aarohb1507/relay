package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"relay/gateway/internal/db"
	"relay/gateway/internal/models"
)

func CreateOrGetWorkflow(request models.CreateWorkflowRequest) (models.Workflow, bool, error) {
	taskPayload, err := json.Marshal(request.Task)
	if err != nil {
		return models.Workflow{}, false, fmt.Errorf("marshal workflow task: %w", err)
	}

	maxTokens := request.MaxTokens
	if maxTokens == 0 {
		maxTokens = 10000
	}

	tx, err := db.DB.Begin()
	if err != nil {
		return models.Workflow{}, false, fmt.Errorf("begin workflow transaction: %w", err)
	}
	defer tx.Rollback()

	// Return an existing workflow before attempting a new insert.
	var workflow models.Workflow
	err = scanWorkflow(tx.QueryRow(`
		SELECT id, status, current_step, max_tokens, task_payload,
		       secret_ref, idempotency_key, created_at, updated_at
		FROM workflows
		WHERE idempotency_key = $1
	`, request.IdempotencyKey), &workflow)
	if err == nil {
		if err := tx.Commit(); err != nil {
			return models.Workflow{}, false, fmt.Errorf("commit existing workflow lookup: %w", err)
		}
		return workflow, false, nil
	}
	if err != sql.ErrNoRows {
		return models.Workflow{}, false, fmt.Errorf("find workflow by idempotency key: %w", err)
	}

	var row *sql.Row
	if request.WorkflowID == "" {
		row = tx.QueryRow(`
			INSERT INTO workflows (status, max_tokens, task_payload, secret_ref, idempotency_key)
			VALUES ('PENDING', $1, $2, $3, $4)
			ON CONFLICT (idempotency_key) DO NOTHING
			RETURNING id, status, current_step, max_tokens, task_payload,
			          secret_ref, idempotency_key, created_at, updated_at
		`, maxTokens, taskPayload, request.SecretRef, request.IdempotencyKey)
	} else {
		row = tx.QueryRow(`
			INSERT INTO workflows (id, status, max_tokens, task_payload, secret_ref, idempotency_key)
			VALUES ($1, 'PENDING', $2, $3, $4, $5)
			ON CONFLICT (idempotency_key) DO NOTHING
			RETURNING id, status, current_step, max_tokens, task_payload,
			          secret_ref, idempotency_key, created_at, updated_at
		`, request.WorkflowID, maxTokens, taskPayload, request.SecretRef, request.IdempotencyKey)
	}

	// A concurrent request may win the unique-key race; read its workflow instead.
	err = scanWorkflow(row, &workflow)
	if err == sql.ErrNoRows {
		err = scanWorkflow(tx.QueryRow(`
			SELECT id, status, current_step, max_tokens, task_payload,
			       secret_ref, idempotency_key, created_at, updated_at
			FROM workflows
			WHERE idempotency_key = $1
		`, request.IdempotencyKey), &workflow)
		if err == nil {
			if err := tx.Commit(); err != nil {
				return models.Workflow{}, false, fmt.Errorf("commit concurrent workflow lookup: %w", err)
			}
			return workflow, false, nil
		}
	}
	if err != nil {
		return models.Workflow{}, false, fmt.Errorf("create workflow: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return models.Workflow{}, false, fmt.Errorf("commit workflow: %w", err)
	}
	return workflow, true, nil
}

func GetWorkflow(id string) (models.Workflow, bool, error) {
	var workflow models.Workflow
	err := scanWorkflow(db.DB.QueryRow(`
		SELECT id, status, current_step, max_tokens, task_payload,
		       secret_ref, idempotency_key, created_at, updated_at
		FROM workflows
		WHERE id = $1
	`, id), &workflow)
	if err == sql.ErrNoRows {
		return models.Workflow{}, false, nil
	}
	if err != nil {
		return models.Workflow{}, false, fmt.Errorf("get workflow: %w", err)
	}
	return workflow, true, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanWorkflow(row rowScanner, workflow *models.Workflow) error {
	var taskPayload []byte
	err := row.Scan(
		&workflow.ID,
		&workflow.Status,
		&workflow.CurrentStep,
		&workflow.MaxTokens,
		&taskPayload,
		&workflow.SecretRef,
		&workflow.IdempotencyKey,
		&workflow.CreatedAt,
		&workflow.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(taskPayload, &workflow.Task); err != nil {
		return fmt.Errorf("decode workflow task: %w", err)
	}
	return nil
}
