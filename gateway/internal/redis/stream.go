package redis

import (
	goredis "github.com/redis/go-redis/v9"

	"relay/gateway/internal/models"
)

func PublishWorkflow(workflow models.Workflow) error {
	_, err := Client.XAdd(Ctx, &goredis.XAddArgs{
		Stream: "agent_tasks",
		Values: map[string]interface{}{
			"workflow_id": workflow.ID,
		},
	}).Result()
	return err
}
