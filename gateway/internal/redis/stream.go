package redis

import (
	goredis "github.com/redis/go-redis/v9"

	"log"

	"relay/gateway/internal/models"
)

func PublishJob(job models.Job) error {

	log.Println("Publishing job to Redis:", job.ID)

	id, err := Client.XAdd(Ctx, &goredis.XAddArgs{
		Stream: "relay-stream",
		Values: map[string]interface{}{
			
			"job_id": job.ID,
			"tool":   job.Tool,
			
		},
	}).Result()

	if err != nil {
    log.Println("Redis publish failed:", err)
    return err
	}	

	log.Println("Published to Redis with ID:", id)

	return nil
}