package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Connect() {

	connStr := "host=localhost port=5432 user=relay password=relay123 dbname=relay sslmode=disable"

	var err error

	DB, err = sql.Open("postgres", connStr)

	if err != nil {
		log.Fatal(err)
	}

	err = DB.Ping()

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Connected to PostgreSQL")

	CreateJobsTable()
}