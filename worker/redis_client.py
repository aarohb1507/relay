import os

import redis

STREAM_NAME = os.getenv("RELAY_STREAM", "agent_tasks")
GROUP_NAME = os.getenv("RELAY_CONSUMER_GROUP", "relay-workers")
CONSUMER_NAME = os.getenv("RELAY_CONSUMER_NAME", "worker-1")

client = redis.Redis(
    host=os.getenv("RELAY_REDIS_HOST", "localhost"),
    port=int(os.getenv("RELAY_REDIS_PORT", "6379")),
    decode_responses=True,
)


def ensure_consumer_group():
    try:
        client.xgroup_create(
            name=STREAM_NAME,
            groupname=GROUP_NAME,
            id="0",
            mkstream=True,
        )
    except redis.exceptions.ResponseError as error:
        if "BUSYGROUP" not in str(error):
            raise


def read_pending():
    return client.xreadgroup(
        groupname=GROUP_NAME,
        consumername=CONSUMER_NAME,
        streams={STREAM_NAME: "0"},
        count=1,
    )


def read_new():
    return client.xreadgroup(
        groupname=GROUP_NAME,
        consumername=CONSUMER_NAME,
        streams={STREAM_NAME: ">"},
        count=1,
        block=0,
    )


def acknowledge(message_id):
    client.xack(STREAM_NAME, GROUP_NAME, message_id)


def publish_event(workflow_id, status, result=None):
    event = {"workflow_id": workflow_id, "status": status}
    if result is not None:
        event["result"] = result
    client.publish("relay-events", json_encode(event))


def json_encode(value):
    import json

    return json.dumps(value, separators=(",", ":"))
