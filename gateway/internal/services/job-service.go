package services

import (

	"relay/gateway/internal/models"
	"relay/gateway/internal/repository"
	"relay/gateway/internal/redis"

	"fmt"
	"log"

)

var jobCounter = 0

func CreateJob(tool string) (models.Job, error){

	jobCounter++

	job := models.Job {

		ID: fmt.Sprintf("job-%d", jobCounter),
		Tool: tool,
		Status: "QUEUED",	

	}
	
	err := repository.CreateJob(job)

	if err != nil {
		log.Printf("failed to insert job %s: %v", job.ID, err)
		return models.Job{}, err
	}

	err = redis.PublishJob(job)

	if err != nil {
		log.Printf("failed to publish job on queue %s: %v", job.ID, err)
		return models.Job{}, err
	}


	return job, nil
}

func GetJob(id string) (models.Job, bool){
	
	return repository.GetJob(id)

}


func UpdateJobStatus(id string, status string) (models.Job, bool) {
	
	return repository.UpdateJobStatus(id, status)

}