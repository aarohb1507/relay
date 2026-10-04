def execute(task):
    """Deterministic stand-in for the first WAL recovery proof."""
    return {
        "provider": "fake",
        "operation": "process_task",
        "goal": task.get("goal", ""),
        "prompt": task.get("prompt", ""),
        "params": task.get("params", {}),
        "processed": True,
    }
