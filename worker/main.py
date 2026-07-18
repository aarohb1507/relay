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

                result = {
                    "temperature": 28,
                    "city": "Bangalore",
                }

                db.complete_job(
                    data["job_id"],
                    result,
                )

                redis_client.client.xack(
                    "relay-stream",
                    "relay-workers",
                    message_id,
                )

                print("ACK Sent")

print("Worker starting loop...")

while True:

    print("Checking pending...")

    pending = redis_client.client.xreadgroup(
        groupname="relay-workers",
        consumername="worker-1",
        streams={"relay-stream": "0"},
        count=1,
    )
    print("Pending:", pending)

    if pending and pending[0][1]:
        process_jobs(pending)
        continue

    print("Waiting for new jobs...")

    new_jobs = redis_client.client.xreadgroup(
        groupname="relay-workers",
        consumername="worker-1",
        streams={"relay-stream": ">"},
        count=1,
        block=0,
    )

    print("New Jobs:", new_jobs)

    process_jobs(new_jobs)


    