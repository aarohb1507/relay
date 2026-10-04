# Relay v2 Implementation Plan

## Product Intent

Relay is a durable execution runtime and state control plane for autonomous AI agents that perform risky, state-changing actions.

It sits between non-deterministic agent decisions and deterministic external systems such as payment providers, cloud APIs, databases, and SaaS platforms.

The practical problem is simple:

> If a worker crashes after an external action but before recording the result, the system must recover without blindly repeating the action.

## Product Positioning

Relay is similar to a durable execution engine for AI agents, with a specific focus on safely governing LLM-selected tool calls.

The client agent decides what it wants to accomplish. Relay controls whether and how the requested action is executed, recorded, retried, recovered, or compensated.

### Pitch

Autonomous agents can reason, but they are unreliable when their actions affect money, infrastructure, or business data. Relay makes those actions durable and recoverable by recording intent before side effects, using stable idempotency keys, surviving worker crashes, reconciling uncertain outcomes, and executing deterministic compensation when supported.

## Central Invariant

For every supported state-changing action:

> Relay records durable intent with a stable idempotency key before calling the external system, and acknowledges the Redis task only after the outcome and local workflow state are durably recorded.

This provides effectively-once handling when the external provider supports idempotency or status reconciliation. Relay does not claim impossible absolute exactly-once execution across arbitrary external APIs.

## Chosen Architecture

Relay uses one master Redis task per workflow.

```text
Client
  -> Go Gateway
  -> PostgreSQL workflow snapshot
  -> Redis Streams master task
  -> Python Worker
  -> WAL-protected tool calls
  -> PostgreSQL result
  -> SSE telemetry
```

The worker owns the sequential execution loop. Redis carries the workflow task; PostgreSQL and the WAL carry durable state and execution history.

The first implementation will not pre-create a separate Redis message for every step. The LLM may dynamically choose the next tool based on previous results, so each action is recorded as it is created.

## Storage Responsibilities

### `workflows`

Stores the current snapshot of the complete workflow:

- workflow ID;
- status;
- current action number;
- original task payload;
- secret reference, never raw credentials;
- token budget;
- timestamps.

### `wal_events`

Stores the append-only history of important actions:

- `INTENT` before an external call;
- `SUCCESS` after a confirmed result;
- `FAILED` for terminal action failures;
- `TOKEN_ACCUMULATED` for LLM usage;
- `UNDO_INTENT` before compensation;
- `UNDO_SUCCESS` after compensation.

The WAL is recorded against the workflow and action identity. A single action can have both `INTENT` and `SUCCESS` events.

### `compensating_actions`

Stores deterministic undo instructions registered after successful forward actions. Rollback never asks the LLM what to undo.

## Honest Guarantees

Relay can provide:

- durable workflow state;
- crash-resumable task handling;
- stable action identity;
- effectively-once behavior for supported providers;
- durable audit history;
- token and execution limits;
- deterministic compensation where an undo operation exists;
- live progress telemetry.

Relay cannot guarantee:

- absolute exactly-once execution against every external API;
- that every side effect has a valid undo operation;
- 99.99% availability from application code alone;
- automatic recovery when a provider gives no idempotency or status mechanism;
- safe execution of arbitrary unvalidated LLM output.

## Implementation Order

### Phase 0: Recover and stabilize v1

Before adding new behavior:

- trace the existing `/jobs` flow end to end;
- fix the missing legacy `jobs.result` column;
- verify PostgreSQL updates from the Python worker;
- verify Redis Stream delivery;
- verify Redis Pub/Sub to SSE delivery;
- add basic repeatable tests or verification commands;
- document what v1 does and what it does not do.

The existing `/jobs` path remains temporarily available while v2 is introduced.

### Phase 1: Durable workflow submission

- Add workflow request and response models.
- Add the workflow table and migration path.
- Add workflow-level idempotency.
- Add `POST /v1/workflows/execute`.
- Persist the workflow before publishing work.
- Publish one master task to `agent_tasks`.
- Add workflow state retrieval.
- Ensure duplicate submissions do not enqueue duplicate master tasks.

### Phase 2: One deterministic tool through the WAL

Start with a fake provider, not an LLM or Stripe.

- Load the workflow from PostgreSQL.
- Check whether the action already has `SUCCESS`.
- Write `INTENT` atomically.
- Execute the deterministic fake tool.
- Write `SUCCESS` or `FAILED`.
- Update workflow state.
- Publish telemetry.
- ACK Redis last.

### Phase 3: Failure and worker recovery

Test crashes at each boundary:

- before `INTENT`;
- after `INTENT`;
- after the fake provider succeeds;
- after `SUCCESS` is recorded;
- before PostgreSQL state update;
- before Redis ACK.

Then add:

- worker identity;
- Redis leases;
- heartbeats;
- `XPENDING` inspection;
- `XCLAIM` recovery;
- split-brain protection;
- recovery telemetry.

### Phase 4: ReAct execution loop

- Add an LLM provider interface.
- Store enough prompt, response, tool-call, result, and token context for recovery.
- Rebuild the context for each LLM call.
- Validate structured tool calls against an allowlist and schemas.
- Check token budgets before LLM calls.
- Treat LLM output as a proposal, never as authority.

The loop is:

```text
reason -> propose tool -> validate -> WAL INTENT -> execute -> WAL SUCCESS -> observe -> reason again
```

### Phase 5: Provider adapters

Each external integration must explicitly define:

```text
execute()
check_status()
compensate()
```

Relay must document provider-specific behavior instead of assuming that every API behaves like Stripe.

### Phase 6: Deterministic compensation

- Register an undo action after successful forward work.
- Enter `ROLLBACK_INITIATED` on terminal failure.
- Execute undo actions in reverse order.
- Write `UNDO_INTENT` and `UNDO_SUCCESS` through the WAL.
- Retry failed compensation with bounded backoff.
- Enter `ROLLBACK_FAILED` when operator intervention is required.
- Expose rollback progress through SSE and durable state.

Rollback is a separate control path, but it can initially run inside the Python worker. It must not use the LLM to decide how to undo work.

## Action Persistence and Retry Rules

Each action result is persisted in `wal_events.payload` as JSONB. The payload may contain the action input, provider response, normalized output, error details, and any data needed by the next workflow step or by crash recovery.

Forward execution and rollback use separate retry paths:

```text
Attempt forward action
  -> success: write SUCCESS with input and output to the WAL
  -> transient failure: retry the same action with the same idempotency key
  -> terminal failure or exhausted forward retries: begin rollback

Attempt compensation
  -> success: write UNDO_SUCCESS to the WAL
  -> transient failure: retry the compensation
  -> exhausted compensation retries: mark ROLLBACK_FAILED and alert an operator
```

Forward retries must be classified by the provider or tool adapter. A replacement worker must reconcile an uncertain result before repeating an external action. Compensation also requires a stable idempotency key because a worker can crash during an undo call.

`workflows` stores the current snapshot, `wal_events` stores append-only action history and results, and `compensating_actions` stores the current undo state and retry counters. PostgreSQL remains the source of truth; a Redis dead-letter stream is only an operator notification mechanism.

### Phase 7: Production hardening

- configuration management;
- secret references and secret-manager integration;
- structured logging;
- metrics;
- tracing;
- timeouts;
- graceful shutdown;
- authentication and authorization;
- integration tests;
- chaos tests;
- Docker deployment;
- Kubernetes only after runtime behavior is proven.

## Required Proof Tests

- Duplicate workflow submission returns the original workflow and creates one master task.
- A persisted `SUCCESS` prevents a second tool execution.
- A crash after `INTENT` can be reconciled or safely retried.
- A crash after external success does not blindly duplicate the action.
- PostgreSQL state is durable before Redis ACK.
- A worker restart resumes the workflow from durable state.
- Lease heartbeats prevent reclaiming a healthy slow worker.
- A dead worker can be reclaimed by another worker.
- Token limits stop further LLM calls.
- Terminal failure executes compensation in reverse order.
- Completed compensation is not repeated.
- Rollback failure becomes visible to an operator.
- SSE disconnects do not change PostgreSQL truth.

## First Demonstration

The first credible demo is:

```text
1. Submit one workflow.
2. Queue one master task.
3. Worker writes INTENT.
4. Fake provider performs one deterministic action.
5. Kill the worker before it finishes recording the result.
6. Start a replacement worker.
7. Replacement reads PostgreSQL and resumes safely.
8. Workflow reaches SUCCESS.
9. Client receives progress and completion over SSE.
```

The demo should show the failure, recovery, database state, and absence of duplicate execution.

## Current Scope Boundary

The first implementation does not include:

- real Stripe calls;
- arbitrary external tools;
- production LLM orchestration;
- Saga rollback execution;
- Kubernetes;
- multi-region availability;
- compliance certification;
- a 99.99% uptime claim.

Those become meaningful only after the first durable execution proof works.
