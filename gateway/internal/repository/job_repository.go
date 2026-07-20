package repository

import (

	"log"
	
	"relay/gateway/internal/db"
	"relay/gateway/internal/models"

	"encoding/json"

)

func CreateJob(job models.Job) error{

	_, err := db.DB.Exec(
		"INSERT INTO jobs (id, tool, status) VALUES ($1, $2, $3)",
		job.ID,
		job.Tool,
		job.Status,
	)

	if err != nil {
		log.Printf("failed to insert job %s: %v", job.ID, err)
		return err
	}

	return nil
}

func GetJob(id string) (models.Job, bool){

	var job models.Job
	var result []byte 

	err := db.DB.QueryRow(
		"SELECT id, tool, status, result from jobs where id = $1",
		id,
	).Scan(
		&job.ID,
		&job.Tool,
		&job.Status,
		&result,
	)

	if err != nil {
		log.Printf("Job Not Found %s: %v", id, err)
		return models.Job{}, false
	}

	if result != nil {
		json.Unmarshal(result, &job.Result)
	}

	return job, true
}

func UpdateJobStatus(id string, status string) (models.Job, bool){

	_, err := db.DB.Exec(
		"UPDATE jobs SET status = $1 WHERE id = $2",
		status,
		id,
	)

	if err != nil {
		log.Printf("failed to update job status %s: %v", id, err)
		return models.Job{}, false
	}

	return GetJob(id)

}