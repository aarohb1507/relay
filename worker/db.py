import psycopg2

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