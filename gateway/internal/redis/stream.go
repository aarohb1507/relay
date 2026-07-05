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

func ReadJobs() error {

	streams, err := Client.XRead(Ctx, &goredis.XReadArgs{
		Streams: []string{"relay-streams", "0"},
		Count: 10,
	}).Result()

	if err != nil {
		return err
	}

	for _, stream := range streams {

		log.Println("Stream:", stream.Stream)

		for _, message := range stream.Messages {
			log.Println("ID:", message.ID)
			log.Println("Values: ", message.Values)
		}
	}
	return nil
}