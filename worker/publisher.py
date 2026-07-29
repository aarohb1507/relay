import json
import redis_client

def publish_event(job_id, status, result=None):
    event = {
        "job_id": job_id,
        "status": status,
        "result": result,
    }

    redis_client.client.publish(
        "relay-events",
        json.dumps(event),
    )