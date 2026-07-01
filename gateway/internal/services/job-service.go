package services

import (
	"relay/gateway/internal/models"
	"relay/gateway/internal/db"
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
	
	_, err := db.DB.Exec(

		"INSERT INTO jobs (id, tool, status) VALUES ($1, $2, $3)",
		job.ID,
		job.Tool,
		job.Status,

	)

	if err != nil {
		log.Printf("failed to insert job %s: %v", job.ID, err)
		return models.Job{}, err
	}

	return job, nil
}

func GetJob(id string) (models.Job, bool){
	
	var job models.Job

	err := db.DB.QueryRow(
		"SELECT id, tool, status from jobs where id = $1",
		id,
	).Scan(
		&job.ID,
		&job.Tool,
		&job.Status,
	)

	if err != nil {
		log.Printf("failed to fetch job %s: %v", id, err)
		return models.Job{}, false
	}

	return job, true
}


func UpdateJobStatus(id string, status string) (models.Job, bool) {
	
	_, err := db.DB.Exec(
		"UPDATE jobs SET status = $1 WHERE id = $2",
		status,
		id,
	)

	if err != nil {
		log.Printf("failed to update job %s : %v", id, err)
		return models.Job{}, false
	}

	return GetJob(id)
}