package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql" // Side effects
	"github.com/robfig/cron/v3"
	"log"
	"os"
)

var ( // Get database info from ENV
	username = os.Getenv("DB_USER")
	password = os.Getenv("DB_PASS")
	hostname = os.Getenv("DB_HOST")
	database = os.Getenv("DB_NAME")
)

type LoginAttempt struct { // Struct to store login attempts from syslog
	Username  string `json:"username"`
	Timestamp string `json:"timestamp"`
	Success   bool   `json:"success"`
}

func databaseURI() string { // Create connection string
	return fmt.Sprintf("%s:%s@tcp(%s)/%s", username, password, hostname, database)
}

func insert(db *sql.DB, loginAttempts []LoginAttempt) { // Routine
	log.Println("[Job 1] Run")

	for _, attempt := range loginAttempts { // Insert login attempts from slice of structs
		q := "INSERT INTO logs (username, timestamp) VALUES (?, ?)" // Create prepared statement
		_, err := db.Query(q, attempt.Username, attempt.Timestamp)  // Insert with values from struct
		if err != nil {                                             // Log errors to console
			log.Println(err.Error())
		}
	}
} // End insert

func main() {
	loginAttempts := []LoginAttempt{ // Test slice of structs to simulate logins
		{Username: "err4471", Timestamp: "2022-10-05", Success: false},
		{Username: "err4471", Timestamp: "2022-10-05", Success: true},
		{Username: "err4471", Timestamp: "2022-10-05", Success: true},
	}

	db, err := sql.Open("mysql", databaseURI()) // Open database connection

	if err != nil { // Log failed connections
		panic("Unable to establish database connection.")
	}
	defer func(db *sql.DB) { // Handle disconnection at end of session
		err := db.Close()
		if err != nil { // Handle disconnection errors
			log.Fatal("Unable to gracefully disconnect from database.")
		}
	}(db)

	c := cron.New() // Instantiate cron

	_, err = c.AddFunc("@every 5s", func() { // Schedule insert
		go insert(db, loginAttempts) // Add insert routine
	})
	if err != nil { // Handle errors scheduling the jobs
		log.Fatal("Failed to add jobs.")
	}

	c.Start() // Start scheduled jobs

	_, err = fmt.Scanln() // Run jobs until key press
	if err != nil {
		return
	}
} // End main
