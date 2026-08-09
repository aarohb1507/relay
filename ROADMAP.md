# Relay Roadmap

> Current development status and implementation roadmap for Relay.

---

# ✅ Completed

## Foundation
- [x] Go Gateway HTTP server
- [x] Health endpoint
- [x] Job creation endpoint
- [x] PostgreSQL integration
- [x] Redis Streams integration
- [x] Python Worker
- [x] Consumer Group
- [x] Pending Entry List (PEL) handling
- [x] Job state updates (QUEUED → RUNNING → COMPLETED)
- [x] Store tool results in PostgreSQL
- [x] GET Job endpoint
- [x] Return completed results from database

---

## Reliability

- [x] Durable job queue using Redis Streams
- [x] ACK after successful processing
- [x] PostgreSQL as the source of truth

---

## Real-Time

- [x] SSE endpoint
- [x] Event handler
- [x] Client registration (jobID → ResponseWriter)
- [x] Event sending service (`events.Send()`)
- [x] JSON event serialization
- [x] Automatic cleanup when writing to a dead client fails
- [x] Worker publishes RUNNING event
- [x] Worker publishes COMPLETED event
- [x] Gateway Pub/Sub subscriber
- [x] Start subscriber as background goroutine
- [x] End-to-end real-time event flow
- [x] Close connection after COMPLETED
- [x] Client disconnect cleanup
- [x] Proper connection lifecycle

---

# 📌 Next Production Layers

## Layer 2 — Worker Reliability

- [ ] Retry mechanism
- [ ] Dead Letter Queue (DLQ)
- [ ] Retry events
- [ ] Failure events
- [ ] Worker crash recovery

---

## Layer 3 — Observability

- [ ] Structured logging
- [ ] Request tracing
- [ ] Metrics
- [ ] Job execution timings
- [ ] Queue statistics

---

## Layer 4 — Scalability

- [ ] Multiple workers
- [ ] Multiple gateway instances
- [ ] Horizontal scaling
- [ ] Worker identification
- [ ] Concurrency improvements

---

## Layer 5 — Production Hardening

- [ ] Context propagation
- [ ] Graceful shutdown
- [ ] Configuration management
- [ ] Timeouts
- [ ] Validation
- [ ] Error handling improvements

---

## Layer 6 — API Improvements

- [ ] Better event schema
- [ ] Progress events
- [ ] Error events
- [ ] Retry events
- [ ] Job cancellation
- [ ] Event history

---

## Layer 7 — Deployment

- [ ] Docker polish
- [ ] Kubernetes manifests
- [ ] Production deployment
- [ ] Monitoring stack

---

# Engineering Principles

- PostgreSQL is the source of truth.
- Redis Streams are used for durable job execution.
- Redis Pub/Sub is used only for transient notifications.
- Keep package responsibilities clean.
- Prefer simple, production-style Go.
- Build complete vertical features before moving on.
- Optimize for shipping working production features over writing unnecessary boilerplate.
