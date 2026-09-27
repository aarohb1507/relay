# RELAY V2: FAULT-TOLERANT AI EXECUTION RUNTIME & STATE CONTROL PLANE
## Technical System Specification & Architecture Blueprint
**Author:** Aaroh Bhardwaj  
**Version:** 2.0 (Production Architecture Spec)  
**Target Execution Date:** December 31, 2026  

---

## 1. Executive Overview & Core Problem

### 1.1 The Enterprise Problem
Enterprise adoption of autonomous AI agents is blocked not by model capability, but by **execution instability and state corruption**. When non-deterministic LLM agents interact with stateful external APIs (Stripe, PostgreSQL, cloud providers, third-party SaaS), they introduce critical failure modes:
1. **State Corruption & Double Execution:** A worker process crashing (`kill -9`, OOM, network partition) right after calling a paid or state-mutating API leaves the system unable to know if the side-effect occurred, leading to duplicate charges or orphaned resources.
2. **Runaway Cost Loops:** LLMs stuck in infinite reasoning or tool-call loops can burn thousands of dollars in token credits within minutes.
3. **Cascading Side-Effect Failures:** When a multi-step workflow fails at Step $N$, previously completed side-effects at Steps $1 \dots N-1$ remain uncleared, leaving downstream systems in an inconsistent state.

### 1.2 The Solution: Relay v2
Relay v2 is an open-source, fault-tolerant execution runtime and state control plane that acts as an **Operating System Kernel for AI Agents**. It sits between non-deterministic LLM reasoning engines and stateful external networks, enforcing **effectively-once execution**, **event-log-backed state recovery**, **deterministic SAGA rollbacks**, and **hard token budget circuit breakers**.

---

## 2. High-Level Architecture & Component Stack

```
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                                 RELAY V2 SYSTEM ARCHITECTURE                           │
├────────────────────────────────────────────────────────────────────────────────────────┤
│ 1. CLIENT APPLICATION / TRIGGER LAYER                                                  │
│    • Sends prompt, tool schemas, secret reference ID (`secret_ref`), and token budget. │
│    • Listens to real-time execution telemetry over Server-Sent Events (SSE).           │
├────────────────────────────────────────────────────────────────────────────────────────┤
│ 2. GO GATEWAY & SUPERVISOR CONTROL PLANE                                               │
│    • Ingestion API (`POST /v1/workflows/execute`): Writes snapshot to PostgreSQL.      │
│    • Dispatches single master task ID to Redis Streams (`XADD agent_tasks`).           │
│    • Supervisor Ticker (3s interval): Scans `XPENDING` idle tasks (>15s) and checks    │
│      Redis lease expiration (`workflow_lease:<id>`). Issues `XCLAIM` on dead workers.  │
├────────────────────────────────────────────────────────────────────────────────────────┤
│ 3. PYTHON EXECUTION WORKER FLEET                                                       │
│    • Consumes master task via Redis `XREADGROUP` and sets 10s heartbeating lease.      │
│    • Drives the multi-step LLM reasoning loop sequentially.                             │
│    • Token Safety Guard: Pre-commit check of cumulative WAL tokens before LLM calls.   │
│    • WAL Guard: Writes `INTENT` to PostgreSQL BEFORE calling external APIs.            │
├────────────────────────────────────────────────────────────────────────────────────────┤
│ 4. POSTGRESQL EVENT-LOG-BACKED STATE ENGINE                                            │
│    • `workflows`: Mutable state snapshot table.                                        │
│    • `wal_events`: Immutable append-only WAL with composite unique constraints.        │
│    • `compensating_actions`: Inverse SAGA undo log registered progressively.            │
├────────────────────────────────────────────────────────────────────────────────────────┤
│ 5. DETERMINISTIC SAGA ROLLBACK ENGINE                                                  │
│    • Zero LLM involvement during rollbacks.                                            │
│    • Executes registered undo handlers in strict reverse order (`ORDER BY step_number  │
│      DESC`) on terminal business failures (e.g., Card Declined, 400 Bad Request).       │
└────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Key Design Refinements & Execution Guarantees

### 3.1 Effectively-Once Execution & Provider Status Reconciliation
* **Mathematical Reality:** Due to the Two Generals Problem over non-transactional HTTP boundaries, pure "exactly-once" execution is impossible. Relay v2 guarantees **effectively-once execution** backed by stable idempotency keys.
* **Reconciliation Protocol:** If a worker crashes after writing `INTENT` but before writing `SUCCESS`, the replacement worker (reclaimed via `XCLAIM`) **does not blindly re-execute**. It executes a **Provider Status Lookup** using the `idempotency_key` (e.g., querying Stripe or Cloud Provider status APIs). If the provider confirms execution, the worker writes `SUCCESS` to WAL and advances; otherwise, it executes the tool safely.

### 3.2 Split-Brain Prevention (`XCLAIM` + Heartbeat Leases)
* To prevent reclaiming tasks from slow workers (e.g., long-running LLM tool calls), supervisor reclaim requires **two concurrent conditions**:
  1. Redis `XPENDING` idle time $> 15\text{ seconds}$.
  2. Redis lease key `workflow_lease:<workflow_id>` is expired in Redis (`SET workflow_lease:<id> worker_id EX 10 NX`).
* While active, workers refresh this lease key every 3 seconds. If a worker dies, the lease expires within 10 seconds, allowing safe `XCLAIM` takeover.

### 3.3 Multi-Step Queuing Mechanics
* **Single Master Task Entry:** Only **1 Master Task ID** is pushed into Redis Streams (`agent_tasks`) per workflow.
* **Sequential Loop Execution:** A single assigned worker runs the multi-step LLM loop sequentially.
* **Crash Resumption:** If a worker dies at Step 3, Worker 2 reclaims the master task, inspects PostgreSQL `wal_events`, sees Steps 1 and 2 are already marked `SUCCESS`, and **skips directly to Step 3** without re-queuing or double-executing prior steps.

---

## 4. Production-Ready Database Schema (`schema.sql`)

```sql
-- 1. Master Workflows Table (Event-Log-Backed Snapshot)
CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING', 
    -- Statuses: 'PENDING', 'RUNNING', 'SUCCESS', 'ROLLBACK_INITIATED', 'ROLLED_BACK', 'ROLLBACK_FAILED', 'FAILED'
    current_step INT NOT NULL DEFAULT 1,
    max_tokens INT NOT NULL DEFAULT 10000,
    task_payload JSONB NOT NULL,
    secret_ref VARCHAR(256), -- Reference to external Secret Vault (NO raw API keys in DB)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Append-Only Write-Ahead Log (WAL)
CREATE TABLE IF NOT EXISTS wal_events (
    id BIGSERIAL PRIMARY KEY,
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    step_number INT NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    event_type VARCHAR(64) NOT NULL, 
    -- Event Types: 'INTENT', 'SUCCESS', 'FAILED', 'UNDO_INTENT', 'UNDO_SUCCESS', 'UNDO_FAILED', 'TOKEN_ACCUMULATED'
    tokens_used INT DEFAULT 0,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Composite unique constraint permits INTENT, SUCCESS, and UNDO events per idempotency key
    CONSTRAINT uk_idempotency_event UNIQUE (idempotency_key, event_type)
);

-- 3. Compensating Actions Registry (SAGA Undo Log)
CREATE TABLE IF NOT EXISTS compensating_actions (
    id BIGSERIAL PRIMARY KEY,
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    step_number INT NOT NULL,
    undo_action VARCHAR(64) NOT NULL,
    undo_payload JSONB NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING', 
    -- Statuses: 'PENDING', 'IN_PROGRESS', 'COMPLETED', 'FAILED'
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 3,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Performance & Recovery Indexes
CREATE INDEX IF NOT EXISTS idx_wal_workflow_id ON wal_events(workflow_id);
CREATE INDEX IF NOT EXISTS idx_wal_idempotency ON wal_events(idempotency_key);
CREATE INDEX IF NOT EXISTS idx_compensating_workflow ON compensating_actions(workflow_id, step_number DESC);
```

---

## 5. State Machine & SAGA Rollback Protocol

### 5.1 Workflow State Transitions
```
                ┌───────────┐
                │  PENDING  │
                └─────┬─────┘
                      │ Task Picked Up via XREADGROUP
                      ▼
                ┌───────────┐
     ┌─────────►│  RUNNING  ├──────────┐
     │          └─────┬─────┘          │
     │                │                │
Token Budget OK /     │ Step Error     │ All Steps Complete
Step Succeeded        │ (Terminal)     │
     │                ▼                ▼
     │      ┌──────────────────┐ ┌───────────┐
     └──────┤ROLLBACK_INITIATED│ │  SUCCESS  │
            └─────────┬────────┘ └───────────┘
                      │
           ┌──────────┴──────────┐
           ▼                     ▼
  All Undos Succeeded     Retries Exhausted
           │                     │
           ▼                     ▼
    ┌────────────┐     ┌─────────────────┐
    │ROLLED_BACK │     │ ROLLBACK_FAILED │ (Alerts Operator DLQ)
    └────────────┘     └─────────────────┘
```

### 5.2 Deterministic Rollback Protocol (Zero LLM Involvement)
When a step encounters a non-retriable terminal business error (e.g., HTTP 400, Payment Declined):
1. **Transition Status:** Update `workflows.status = 'ROLLBACK_INITIATED'`.
2. **Fetch Reverse Undo Plan:** Execute:
   ```sql
   SELECT id, step_number, undo_action, undo_payload, retry_count, max_retries 
   FROM compensating_actions 
   WHERE workflow_id = :workflow_id 
   ORDER BY step_number DESC;
   ```
3. **Execute Reverse Steps:**
   * Check WAL for existing `UNDO_SUCCESS` for step. Skip if completed.
   * Write `UNDO_INTENT` to `wal_events`.
   * Invoke deterministic Python/Go undo handler function with `undo_payload`.
   * On success, write `UNDO_SUCCESS` and update compensating action to `COMPLETED`.
   * On failure, increment `retry_count` with exponential backoff ($2^{\text{retry}}$ seconds). If `max_retries` is hit, transition workflow to `ROLLBACK_FAILED` and push alert to Operator Dead-Letter Queue (DLQ).

---

## 6. Token Safety Guard (Circuit Breaker)

Before sending any prompt to an LLM provider, the worker executes a lightweight WAL pre-check:

```python
def verify_token_budget(db_conn, workflow_id: str, max_tokens: int):
    result = db_conn.query(
        "SELECT COALESCE(SUM(tokens_used), 0) AS total FROM wal_events WHERE workflow_id = %s",
        (workflow_id,)
    )
    current_tokens = result[0]['total']
    if current_tokens >= max_tokens:
        raise TokenBudgetExceededError(
            f"Workflow {workflow_id} exceeded token budget: {current_tokens}/{max_tokens}"
        )
```

---

## 7. Execution Roadmap & Proof-of-Work Target

```
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                        EXECUTION MATRIX                   
├────────────────────────────────────────────────────────────────────────────────────────┤
│ PHASE 1: CORE ENGINE IMPLEMENTATION (WEEKS 1–4)                                        │
│ • Execute `schema.sql` migration on PostgreSQL instance.                               │
│ • Build Python WAL worker daemon (`INTENT` pre-commit, token guard, tool execution).  │
│ • Build Go supervisor ticker (`XCLAIM` + lease expiry validation).                      │
│ • Build SAGA reverse rollback handler (`ORDER BY step_number DESC`).                  │
├────────────────────────────────────────────────────────────────────────────────────────┤
│ PHASE 2: CHAOS TEST SUITE & DEMO (WEEKS 5–8)                                           │
│ • Implement `tests/chaos_test.py` firing `docker kill -9` mid-transaction.              │
│ • Record 2-minute terminal Loom benchmark demo proving 0% state loss.                  │
│ • Publish open-source repository & technical breakdown.                                │
├────────────────────────────────────────────────────────────────────────────────────────┤
│ PHASE 3: DIRECT FOUNDER OUTREACH (WEEKS 9–12)                                          │
│ • Direct outreach to 20 CTOs of Series Seed/A/B AI Infra startups in Bangalore/Remote. │
│ • Present live chaos demo to secure Founding Engineer / AI Platform Engineer role      │
│   (₹20L–₹35LPA+ base cash + founding equity).                                          │
└────────────────────────────────────────────────────────────────────────────────────────┘
```

---

*This specification serves as the immutable architectural contract for Relay v2.*
