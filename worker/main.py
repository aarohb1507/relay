import logging

import db
import fake_provider
import redis_client

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("relay-worker")


def process_message(connection, message_id, values):
    workflow_id = values.get("workflow_id")
    if not workflow_id:
        raise ValueError("agent task is missing workflow_id")

    workflow = db.load_workflow(connection, workflow_id)
    if workflow is None:
        raise ValueError(f"workflow {workflow_id} was not found")

    step_number = workflow["current_step"]
    action_key = f"{workflow_id}:step:{step_number}:fake"

    if db.has_success(connection, action_key):
        redis_client.acknowledge(message_id)
        return

    input_payload = {
        "task": workflow["task"],
        "step_number": step_number,
    }
    db.write_intent(
        connection,
        workflow_id,
        step_number,
        action_key,
        input_payload,
    )
    redis_client.publish_event(workflow_id, "RUNNING")

    output_payload = fake_provider.execute(workflow["task"])
    result_payload = {
        "input": input_payload,
        "output": output_payload,
    }
    db.write_success_and_complete(
        connection,
        workflow_id,
        step_number,
        action_key,
        result_payload,
    )
    redis_client.publish_event(workflow_id, "SUCCESS", output_payload)
    redis_client.acknowledge(message_id)
    logger.info("completed workflow %s", workflow_id)


def process_batch(connection, messages):
    for _, stream_messages in messages:
        for message_id, values in stream_messages:
            try:
                process_message(connection, message_id, values)
            except Exception:
                logger.exception("workflow task failed; leaving it pending")


def run():
    connection = db.connect()
    redis_client.ensure_consumer_group()
    logger.info("Relay v2 worker listening on %s", redis_client.STREAM_NAME)

    try:
        while True:
            pending = redis_client.read_pending()
            if pending and pending[0][1]:
                process_batch(connection, pending)
                continue

            process_batch(connection, redis_client.read_new())
    finally:
        connection.close()


if __name__ == "__main__":
    run()
