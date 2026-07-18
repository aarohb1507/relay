import psycopg2
import json

conn = psycopg2.connect(
    host="localhost",
    port=5432,
    database="relay",
    user="relay",
    password="relay123",
)

cursor = conn.cursor()

print("Connected to PostgreSQL")


def update_job_status(job_id, status):

    cursor.execute(
        """
        UPDATE jobs
        SET status = %s
        WHERE id = %s
        """,
        (status, job_id),
    )

    conn.commit()

def complete_job(job_id, result):

    cursor.execute(
        """
        UPDATE jobs
        SET
            status = %s,
            result = %s
        WHERE id = %s
        """,
        (
            "COMPLETED",
            json.dumps(result),
            job_id,
        ),
    )

    conn.commit()