package db

import "log"

func CreateV2Tables() {
	query := `
	CREATE EXTENSION IF NOT EXISTS pgcrypto;

	CREATE TABLE IF NOT EXISTS workflows (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
		current_step INT NOT NULL DEFAULT 1,
		max_tokens INT NOT NULL DEFAULT 10000,
		task_payload JSONB NOT NULL,
		secret_ref VARCHAR(256),
		idempotency_key VARCHAR(128) UNIQUE NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	-- Keep startup initialization compatible with databases created before these columns existed.
	ALTER TABLE workflows
		ADD COLUMN IF NOT EXISTS max_tokens INT NOT NULL DEFAULT 10000;

	ALTER TABLE workflows
		ADD COLUMN IF NOT EXISTS secret_ref VARCHAR(256);

	CREATE TABLE IF NOT EXISTS wal_events (
		id BIGSERIAL PRIMARY KEY,
		workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
		step_number INT NOT NULL,
		idempotency_key VARCHAR(128) NOT NULL,
		event_type VARCHAR(64) NOT NULL,
		tokens_used INT NOT NULL DEFAULT 0,
		payload JSONB NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		-- One action may legitimately produce both INTENT and SUCCESS events.
		UNIQUE (idempotency_key, event_type)
	);

	CREATE TABLE IF NOT EXISTS compensating_actions (
		id BIGSERIAL PRIMARY KEY,
		workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
		step_number INT NOT NULL,
		undo_action VARCHAR(64) NOT NULL,
		undo_payload JSONB NOT NULL,
		status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
		retry_count INT NOT NULL DEFAULT 0,
		max_retries INT NOT NULL DEFAULT 3,
		last_error TEXT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		,updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	ALTER TABLE wal_events
		ADD COLUMN IF NOT EXISTS tokens_used INT NOT NULL DEFAULT 0;

	ALTER TABLE compensating_actions
		ADD COLUMN IF NOT EXISTS retry_count INT NOT NULL DEFAULT 0;

	ALTER TABLE compensating_actions
		ADD COLUMN IF NOT EXISTS max_retries INT NOT NULL DEFAULT 3;

	ALTER TABLE compensating_actions
		ADD COLUMN IF NOT EXISTS last_error TEXT;

	ALTER TABLE compensating_actions
		ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

	CREATE INDEX IF NOT EXISTS idx_wal_workflow_id
		ON wal_events(workflow_id);

	CREATE INDEX IF NOT EXISTS idx_wal_workflow_step
		ON wal_events(workflow_id, step_number);

	CREATE INDEX IF NOT EXISTS idx_wal_workflow_event
		ON wal_events(workflow_id, event_type);

	CREATE INDEX IF NOT EXISTS idx_compensating_workflow_id
		ON compensating_actions(workflow_id, step_number DESC);
	`

	if _, err := DB.Exec(query); err != nil {
		log.Fatal(err)
	}

	log.Println("Relay v2 tables ready")
}
