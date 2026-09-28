package db

import "log"

func CreateJobsTable() {

	query := `
	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		tool TEXT NOT NULL,
		status TEXT NOT NULL,
		result JSONB
	);

	ALTER TABLE jobs
		ADD COLUMN IF NOT EXISTS result JSONB;
	`

	_, err := DB.Exec(query)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Jobs table ready")
}

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
		payload JSONB NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		UNIQUE (idempotency_key, event_type)
	);

	CREATE TABLE IF NOT EXISTS compensating_actions (
		id BIGSERIAL PRIMARY KEY,
		workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
		step_number INT NOT NULL,
		undo_action VARCHAR(64) NOT NULL,
		undo_payload JSONB NOT NULL,
		status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_wal_workflow_id
		ON wal_events(workflow_id);

	CREATE INDEX IF NOT EXISTS idx_compensating_workflow_id
		ON compensating_actions(workflow_id, step_number DESC);
	`

	if _, err := DB.Exec(query); err != nil {
		log.Fatal(err)
	}

	log.Println("Relay v2 tables ready")
}
