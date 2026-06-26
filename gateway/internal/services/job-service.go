package services

import (
	"relay/gateway/internal/models"
	"fmt"
)

var jobCounter = 0

func CreateJob(tool string) models.Job{

	jobCounter++
	
	job:= models.Job{
		ID: fmt.Sprintf("job-%d", jobCounter),
		Tool: tool,
		Status: "QUEUED",
	}

	return job
}