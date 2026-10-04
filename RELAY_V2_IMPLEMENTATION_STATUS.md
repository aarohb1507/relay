# Relay v2 Implementation Status

This file tracks what is shipped in Relay v2. The v2 specification and implementation plan remain the source of design decisions.

## Shipped

### Gateway submission bridge

- `POST /v1/workflows/execute` accepts the workflow task, tools, token budget, and `secret_ref`.
- `idempotency_key` is required and prevents duplicate workflow creation.
- `workflows` stores the durable workflow snapshot in PostgreSQL.
- `GET /v1/workflows/{id}` reads the workflow snapshot.
- A newly created workflow publishes one `workflow_id` to the Redis `agent_tasks` stream.
- Existing workflow submissions do not publish a second Redis task.
- Raw provider credentials are not stored; callers provide a secret reference.

### v2-only cleanup

- Legacy job routes, job tables, job repositories, and the old worker were removed.
- SSE and Redis Pub/Sub use `workflow_id` terminology.

### First worker bridge

- A Python worker creates or reuses the `relay-workers` consumer group.
- The worker consumes pending and new messages from `agent_tasks`.
- It loads the workflow from PostgreSQL.
- It writes `INTENT` before executing the fake provider.
- It stores action input and output in the WAL `payload` JSONB.
- It writes `SUCCESS`, updates the workflow snapshot, publishes telemetry, and ACKs Redis last.
- A persisted `SUCCESS` prevents the fake action from running again.

## Not Yet Shipped

- PostgreSQL-to-Redis outbox recovery.
- Worker leases and heartbeats.
- `XPENDING` and `XCLAIM` recovery.
- Provider status reconciliation after an uncertain external call.
- Forward retry classification and backoff.
- Deterministic compensation execution.
- LLM/ReAct orchestration.
- Token budget accounting from real LLM responses.
- Chaos tests and integration tests.

## Current Proof Target

```text
Submit workflow
  -> PostgreSQL workflow row
  -> Redis agent_tasks message
  -> Python worker writes INTENT
  -> fake provider returns deterministic output
  -> WAL stores input and output
  -> workflow reaches SUCCESS
  -> Redis message is ACKed last
```

The next proof should kill the worker after `INTENT` and demonstrate that the pending message can be processed again without creating a second `SUCCESS` or losing the workflow result.
