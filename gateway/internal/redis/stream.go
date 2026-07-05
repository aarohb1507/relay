package redis

import "relay/gateway/internal/models"

func PublishJob(job models.Job) error {

	_, err := Client.XAdd(Ctx, &goredis.XAddArgs{
		Stream: "relay-stream",
		Values: map[string]interface{}{
			
			"job_id": job.ID,
			"tool":   job.Tool,
		},
	}).Result()

	return err
}