package models

type Job struct {
	ID	string `json:"jobId"`
	Tool   string `json:"tool"`
	Status string `json:"status"`
	Result map[string]interface{} `json:"result`
}
