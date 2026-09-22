# Relay v2: Fault-Tolerant Agent Execution Runtime

## Project

Relay v2, also referred to as RDATC, is a crash-resilient execution runtime for non-deterministic AI agent workflows and stateful tool calls.

The runtime is designed for operations such as payments, database mutations, and cloud provisioning where a worker crash or partial workflow failure must not silently lose state or repeat an external side effect.

## Goals

- Execute multi-step AI-driven workflows through decoupled Go and Python services.
- Preserve workflow progress and tool execution history in PostgreSQL.
- Recover tasks abandoned by crashed workers through Redis Streams consumer-group recovery.
- Prevent duplicate external side effects with a PostgreSQL write-ahead log and idempotency keys.
- Compensate completed side effects in reverse order when a workflow reaches a terminal business failure.
- Keep workers stateless: a new worker must reconstruct execution context from PostgreSQL and the task payload.
- Expose workflow state and execution telemetry through HTTP and SSE.

## Non-goals for the first implementation

- Building a production LLM provider integration before the execution contracts are stable.
- Guaranteeing exactly-once physical execution against an external system that does not support idempotency.
- Treating Redis Pub/Sub as durable state.
- Adding deployment orchestration before the runtime behavior is covered by tests.

## Architecture

```text
Client / Dashboard
        |
        | POST /v1/workflows/execute
        | GET  /v1/workflows/events?id=<workflow_id>
        v
Go Gateway and Supervisor
        |
        | XADD agent_tasks
        v
Redis Streams: agent_tasks
        |
        | XREADGROUP
        v
Python Execution Worker Fleet
        |
        | PostgreSQL transactions
        v
PostgreSQL
  - workflows: current workflow snapshot
  - wal_events: append-only execution and rollback log
  - compensating_actions: Saga undo registry
```

### Go gateway and supervisor

The Go service owns:

- Workflow submission and validation.
- Publishing tasks to the `agent_tasks` Redis Stream.
- Workflow snapshot reads.
- SSE client registration and event broadcasting.
- A background recovery supervisor that finds abandoned consumer-group messages with `XPENDING` and reassigns them with `XCLAIM`.

### Python execution workers

Workers own:

- Consuming workflow tasks with `XREADGROUP`.
- Running the agent decision loop and tool calls.
- Enforcing the pre-commit WAL invariant before every stateful external call.
- Recording successful, failed, and resumed execution.
- Registering compensating actions.
- Executing Saga rollback actions on terminal business failures.
- Acknowledging Redis messages only after successful processing or durable terminal handling.

### PostgreSQL

PostgreSQL is the source of truth. Redis provides durable task delivery and transient coordination, but workflow state must always be recoverable from PostgreSQL.

## PostgreSQL schema

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    current_step INT NOT NULL DEFAULT 1,
    task_payload JSONB NOT NULL,
    credentials JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS wal_events (
    id BIGSERIAL PRIMARY KEY,
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    step_number INT NOT NULL,
    idempotency_key VARCHAR(128) UNIQUE NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
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
```

### Workflow statuses

- `PENDING`: accepted but not started.
- `RUNNING`: a worker is actively processing the workflow.
- `SUCCESS`: all workflow steps completed.
- `ROLLBACK_INITIATED`: a terminal failure started compensation.
- `ROLLED_BACK`: compensation completed.
- `FAILED`: execution or compensation reached an unrecoverable failure.

### WAL event types

- `INTENT`: durable record written before a physical tool call.
- `SUCCESS`: tool call completed and its result was persisted.
- `FAILED`: terminal business failure was persisted.
- `UNDO_INTENT`: durable record written before a compensation call.
- `UNDO_SUCCESS`: compensation completed successfully.

## Execution invariants

### Pre-commit WAL invariant

A worker must never call an external HTTP endpoint or stateful tool before writing an `INTENT` row to `wal_events`.

For every tool call:

1. Check whether a `SUCCESS` event already exists for the idempotency key.
2. If it exists, return the persisted result without executing the tool again.
3. Insert `INTENT` with `ON CONFLICT DO NOTHING`.
4. Execute the physical tool call, passing the same idempotency key to the external API.
5. Append `SUCCESS` with the result.
6. Update the workflow snapshot and current step transactionally.

An `INTENT` without a corresponding `SUCCESS` means the worker may have crashed during the external call. Recovery must use the same idempotency key and rely on the external provider's idempotency behavior where available.

### Error classification

Retriable errors include network failures, timeouts, rate limits, and HTTP 502/503 responses. The worker must leave the task unacknowledged so it can be retried, subject to bounded retry policy.

Terminal errors include invalid requests, authentication failures, and business declines such as a declined card. The worker must append `FAILED`, initiate Saga rollback, and only acknowledge the Redis message after the terminal state is durably recorded.

### Stateless worker requirement

Workers must not depend on process memory for workflow progress, retry state, or rollback state. On startup or after a crash, a worker must reconstruct the workflow from the task payload and PostgreSQL WAL.

## Redis task and crash recovery

Use the following Redis names in v2:

- Stream: `agent_tasks`
- Consumer group: `agent_group`
- Recovery consumer: `supervisor_recovery_node`

Workers consume with `XREADGROUP`. The Go supervisor runs every three seconds and reclaims messages idle for at least five seconds:

```go
pending, err := client.XPendingExt(ctx, &redis.XPendingExtArgs{
    Stream: "agent_tasks",
    Group:  "agent_group",
    Idle:   5 * time.Second,
    Start:  "-",
    End:    "+",
    Count:  10,
}).Result()

if err == nil {
    var messageIDs []string
    for _, entry := range pending {
        messageIDs = append(messageIDs, entry.ID)
    }

    if len(messageIDs) > 0 {
        _, _ = client.XClaim(ctx, &redis.XClaimArgs{
            Stream:   "agent_tasks",
            Group:    "agent_group",
            Consumer: "supervisor_recovery_node",
            MinIdle:  5 * time.Second,
            Messages: messageIDs,
        }).Result()
    }
}
```

Reclaiming a message does not by itself guarantee safe physical execution. WAL idempotency is the protection against duplicate tool effects after a crash.

## Saga compensation

When a terminal business failure occurs:

1. Set the workflow status to `ROLLBACK_INITIATED`.
2. Read registered compensating actions ordered by `step_number DESC`.
3. For each action, derive a stable key such as `undo_<workflow_id>_<step_number>`.
4. Skip the action if `UNDO_SUCCESS` already exists for that key.
5. Append `UNDO_INTENT` before invoking the undo handler.
6. Execute the compensating action.
7. Append `UNDO_SUCCESS` after it completes.
8. Mark the action `COMPLETED`.
9. Mark the workflow `ROLLED_BACK` after all actions succeed.
10. Mark the workflow `FAILED` if compensation itself cannot complete and requires operator intervention.

Rollback must be reverse ordered and independently crash-resilient. A worker restart must resume from the WAL rather than repeating completed undo operations.

## HTTP contracts

### Execute workflow

`POST /v1/workflows/execute`

```json
{
  "workflow_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
  "task": {
    "goal": "Process Vendor Invoice #9910",
    "params": {
      "vendor_id": "v_881",
      "amount_cents": 15000,
      "currency": "USD"
    }
  },
  "idempotency_key": "idem_inv_9910_exec"
}
```

The gateway must validate the request, persist the workflow snapshot, publish the Redis task, and return the workflow identifier and initial status.

### Workflow events

`GET /v1/workflows/events?id=<workflow_id>`

SSE messages use JSON payloads such as:

```text
data: {"type":"WAL_EVENT","step":1,"status":"INTENT","action":"stripe_charge_hold"}

data: {"type":"WAL_EVENT","step":1,"status":"SUCCESS","action":"stripe_charge_hold","tx":"ch_3M1"}

data: {"type":"WORKER_CRASH","node":"worker_2","event":"RECLAIMED_VIA_XCLAIM"}

data: {"type":"WAL_EVENT","step":2,"status":"RESUMED_FROM_WAL","action":"issue_license"}
```

Events are telemetry. PostgreSQL remains authoritative when an SSE client disconnects.

## Delivery order

### Phase 1: contracts and persistence

- Add the v2 PostgreSQL schema and migrations.
- Introduce workflow identifiers and request models.
- Add repository methods for workflow snapshots and WAL events.
- Add transaction-safe state transitions.

### Phase 2: worker execution core

- Replace the hardcoded job simulation with a workflow task loop.
- Implement WAL idempotency checks and intent/success/failure events.
- Add explicit retriable and terminal error types.
- Add bounded retry metadata and durable retry events.

### Phase 3: crash recovery

- Add the Go `XPENDING` / `XCLAIM` supervisor.
- Emit recovery telemetry.
- Verify recovery after worker termination and after a crash between `INTENT` and `SUCCESS`.

### Phase 4: Saga rollback

- Add compensating-action registration.
- Implement reverse-order rollback with its own WAL events.
- Resume interrupted rollback safely.

### Phase 5: production hardening

- Add structured logs, metrics, tracing, timeouts, graceful shutdown, configuration, and authentication/authorization boundaries.
- Add integration tests with PostgreSQL and Redis.
- Add Docker and deployment manifests only after runtime behavior is stable.

## Required tests

- Duplicate workflow submission with the same idempotency key does not create a second workflow.
- A persisted `SUCCESS` WAL event prevents a second physical tool call.
- A worker crash after `INTENT` is recoverable with the same idempotency key.
- A retriable error leaves the message available for retry.
- A terminal error writes `FAILED` and starts rollback.
- Rollback executes steps in reverse order.
- Completed undo actions are not repeated after worker restart.
- The supervisor reclaims messages idle beyond the recovery threshold.
- Workflow state and WAL remain correct when SSE clients disconnect.

## Current implementation status

Relay v1 already provides the basic Go gateway, PostgreSQL job persistence, Redis Streams delivery, Python worker, and SSE event flow. Relay v2 begins by replacing the generic job model with the workflow/WAL contracts above while preserving the existing reliability principles:

- PostgreSQL is the source of truth.
- Redis Streams provide durable execution delivery.
- Redis Pub/Sub provides transient notifications only.
- Workers acknowledge tasks only after durable handling.
- Public changes should be introduced as complete, testable vertical slices.
