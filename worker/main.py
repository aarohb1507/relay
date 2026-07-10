import db
import redis_client

while True:

    jobs = redis_client.client.xreadgroup(
        groupname="relay-workers",
        consumername="worker-1",
        streams={"relay-stream": ">"},
        count=1,
        block=0,
    )

    for stream, messages in jobs:

        for message_id, data in messages:

            print("Received:", message_id)
            print(data)