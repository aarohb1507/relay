# Relay - Agent Instructions

## Project
Relay is a reliability layer for AI tool execution.

Architecture:
Gateway (Go)
→ Redis Streams
→ Worker (Python)
→ PostgreSQL
→ Redis Pub/Sub
→ SSE

PostgreSQL is the source of truth.

Redis Streams are used for durable job execution.

Redis Pub/Sub is used for transient notifications.

## Coding Principles

- Prefer simple, production-style Go.
- Make minimal changes.
- Follow existing package boundaries.
- Don't rewrite working code.
- Keep responsibilities separated.
- If a feature requires multiple files, implement the complete feature.

## Workflow

The user learns by building.

When implementing features:

1. Brief architecture explanation.
2. Implement the complete feature.
3. Help debug.
4. Move immediately to the next production layer.

Avoid excessive theoretical explanations.

If repository context is missing, inspect the codebase before making assumptions.