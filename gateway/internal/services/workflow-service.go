package services

import (
	"fmt"

	"relay/gateway/internal/models"
	"relay/gateway/internal/redis"
	"relay/gateway/internal/repository"
)

func CreateWorkflow(request models.CreateWorkflowRequest) (models.Workflow, error) {
	workflow, created, err := repository.CreateOrGetWorkflow(request)
	if err != nil {
		return models.Workflow{}, err
	}
	if !created {
		return workflow, nil
	}

	if err := redis.PublishWorkflow(workflow); err != nil {
		return models.Workflow{}, fmt.Errorf("publish workflow task: %w", err)
	}
	return workflow, nil
}

func GetWorkflow(id string) (models.Workflow, bool, error) {
	return repository.GetWorkflow(id)
}
