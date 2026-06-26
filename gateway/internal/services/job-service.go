package services

import "relay/gateway/internal/models"

func CreateJob(tool string) models.Job{
	
	job:= models.Job{
		ID: "job-1",
		Tool: tool,
		Status: "QUEUED",
	}

	return job
}