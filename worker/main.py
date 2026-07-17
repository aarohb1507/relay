import time
import db
import redis_client

def process_jobs(jobs):

        for stream, messages in jobs:

            for message_id, data in messages:

                print("Received:", message_id)
                print(data)

                db.update_job_status(data["job_id"], "RUNNING")

                print("Executing Tool...")

                time.sleep(2)

                db.update_job_status(data["job_id"], "COMPLETED")

                redis_client.client.xack(
                    "relay-stream",
                    "relay-workers",
                    message_id,
                )

                print("ACK Sent")

while True:

    pending = redis_client.client.xreadgroup(
        groupname="relay-workers",
        consumername="worker-1",
        streams={"relay-stream": "0"},
        count=1,
    )

    if pending:
        process_jobs(pending)
        continue

    new_jobs = redis_client.client.xreadgroup(
        groupname="relay-workers",
        consumername="worker-1",
        streams={"relay-stream": ">"},
        count=1,
        block=0,
    )

    process_jobs(new_jobs)


    