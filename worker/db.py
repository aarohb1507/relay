import json
import os
from contextlib import contextmanager

import psycopg2
from psycopg2.extras import Json


def connect():
    return psycopg2.connect(
        host=os.getenv("RELAY_POSTGRES_HOST", "localhost"),
        port=os.getenv("RELAY_POSTGRES_PORT", "5432"),
        database=os.getenv("RELAY_POSTGRES_DB", "relay"),
        user=os.getenv("RELAY_POSTGRES_USER", "relay"),
        password=os.getenv("RELAY_POSTGRES_PASSWORD", "relay123"),
    )


@contextmanager
def transaction(connection):
    try:
        yield connection
        connection.commit()
    except Exception:
        connection.rollback()
        raise


def load_workflow(connection, workflow_id):
    with connection.cursor() as cursor:
        cursor.execute(
            """
            SELECT id, status, current_step, task_payload, idempotency_key
            FROM workflows
            WHERE id = %s
            """,
            (workflow_id,),
        )
        row = cursor.fetchone()

    if row is None:
        return None

    return {
        "id": str(row[0]),
        "status": row[1],
        "current_step": row[2],
        "task": row[3] if isinstance(row[3], dict) else json.loads(row[3]),
        "idempotency_key": row[4],
    }


def has_success(connection, action_key):
    with connection.cursor() as cursor:
        cursor.execute(
            """
            SELECT 1
            FROM wal_events
            WHERE idempotency_key = %s AND event_type = 'SUCCESS'
            LIMIT 1
            """,
            (action_key,),
        )
        return cursor.fetchone() is not None


def write_intent(connection, workflow_id, step_number, action_key, payload):
    with transaction(connection):
        with connection.cursor() as cursor:
            cursor.execute(
                """
                INSERT INTO wal_events (
                    workflow_id, step_number, idempotency_key, event_type, payload
                )
                VALUES (%s, %s, %s, 'INTENT', %s)
                ON CONFLICT (idempotency_key, event_type) DO NOTHING
                """,
                (workflow_id, step_number, action_key, Json(payload)),
            )
            cursor.execute(
                """
                UPDATE workflows
                SET status = 'RUNNING', updated_at = NOW()
                WHERE id = %s AND status = 'PENDING'
                """,
                (workflow_id,),
            )


def write_success_and_complete(
    connection, workflow_id, step_number, action_key, payload
):
    with transaction(connection):
        with connection.cursor() as cursor:
            cursor.execute(
                """
                INSERT INTO wal_events (
                    workflow_id, step_number, idempotency_key, event_type, payload
                )
                VALUES (%s, %s, %s, 'SUCCESS', %s)
                ON CONFLICT (idempotency_key, event_type) DO NOTHING
                """,
                (workflow_id, step_number, action_key, Json(payload)),
            )
            cursor.execute(
                """
                UPDATE workflows
                SET status = 'SUCCESS',
                    current_step = %s,
                    updated_at = NOW()
                WHERE id = %s
                """,
                (step_number + 1, workflow_id),
            )
