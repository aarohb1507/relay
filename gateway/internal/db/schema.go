package db

import "log"

func CreateJobsTable() {
	
	query := `
	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		tool TEXT NOT NULL,
		status TEXT NOT NULL
	);
	`

	_, err := DB.Exec(query)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Jobs table ready")
}