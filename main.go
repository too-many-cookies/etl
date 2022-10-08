package main

import (
	"bufio"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql" // Side effects
	"github.com/robfig/cron/v3"
	"log"
	"os"
	"strings"
	"time"
)

var ( // Get database info from ENV
	username = os.Getenv("DB_USER")
	password = os.Getenv("DB_PASS")
	hostname = os.Getenv("DB_HOST")
	database = os.Getenv("DB_NAME")
	logfile  = os.Getenv("LOG_NAME")
)

type LoginAttempt struct { // Struct to store login attempts from syslog
	Username  string `json:"username"`
	Timestamp string `json:"timestamp"`
	Success   string `json:"success"`
}

func insert(db *sql.DB, ch chan LoginAttempt) { // Routine
	log.Println("[Job 1] Run")

	for attempt := range ch { // Insert login attempts from slice of structs
		q := "INSERT INTO logs (username, timestamp, successful) VALUES (?, ?, ?)"  // Create prepared statement
		_, err := db.Query(q, attempt.Username, attempt.Timestamp, attempt.Success) // Insert with values from struct
		if err != nil {                                                             // Log errors to console
			log.Println(err.Error())
		}
	}
} // End insert

func databaseURI() string { // Create connection string
	return fmt.Sprintf("%s:%s@tcp(%s)/%s", username, password, hostname, database)
}

func ingest(path string, ch chan LoginAttempt) {
	file, err := os.Open(path) // Open logfile
	if err != nil {            // Log errors opening
		log.Println(err.Error())
	}
	defer func(file *os.File) { // Close syslog when finished
		err := file.Close()
		if err != nil {
		}
	}(file)

	scanner := bufio.NewScanner(file) // Create new scanner to read syslog

	for scanner.Scan() { // Read log line by line
		line := scanner.Text() // Hold each line
		if strings.Contains(line, "Failed") {
			f := strings.Fields(line) // Split each line into fields
			attempt := LoginAttempt{  // Handle failed attempts
				Username:  f[8],                                // Grab username field
				Timestamp: generateTimestamp(f[0], f[1], f[2]), // Pass MM, DD, HH:MM:SS
				Success:   "N",                                 // Failure
			}
			ch <- attempt // Hand to insert goroutine
		} else if strings.Contains(line, "session opened") { // Handle successful attempts
			f := strings.Fields(line)
			attempt := LoginAttempt{
				Username:  strings.Split(f[10], "(")[0], // Drop uuid
				Timestamp: generateTimestamp(f[0], f[1], f[2]),
				Success:   "Y", // Success
			}
			ch <- attempt
		}
	}
	close(ch) // Close channel when no attempts remain
}

func generateTimestamp(month string, day string, clock string) string {
	year := time.Now().Year() // The log doesn't store year, so assume it's this one
	m := map[string]string{   // Hacky way of mapping months to avoid the Time package
		"Jan": "01", "Feb": "02", "Mar": "03", "Apr": "04", "May": "05", "Jun": "06",
		"Jul": "07", "Aug": "08", "Sep": "09", "Oct": "10", "Nov": "11", "Dec": "12",
	}
	return fmt.Sprintf("%d-%s-%s %s", year, m[month], day, clock) // Format and return string
} // End ingest

func main() {
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
		ch := make(chan LoginAttempt)
		go ingest(logfile, ch) // Add ingest routine
		go insert(db, ch)      // Add insert routine
	})
	if err != nil { // Handle errors scheduling the jobs
		log.Fatal("Failed to add jobs.")
	}

	c.Start() // Start scheduled jobs

	_, err = fmt.Scanln() // Run jobs until keypress
	if err != nil {
		return
	}
} // End main
