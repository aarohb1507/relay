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

func GetJob(id string) (models.Job, bool){
	
	for _, job := range jobs {

		if job.ID == id {

			return job, true

		}
	}

	return models.Job{}, false
}

func UpdateJobStatus(id string, status string) (models.job, bool) {
	
	for i, job := range jobs {

		if jobs[i].ID == id {

			jobs[i].Status = status

			return jobs[i], true
		}
	}

	return models.Job{}, false
}