package services

import (
	"relay/gateway/internal/models"
	"fmt"
)

var jobs []models.Job

var jobCounter = 0

func CreateJob(tool string) models.Job{

	jobCounter++
	
	job:= models.Job{
		ID: fmt.Sprintf("job-%d", jobCounter),
		Tool: tool,
		Status: "QUEUED",
	}

	jobs = append(jobs, job)

	return job
}