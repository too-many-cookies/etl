package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql" // Side effects
	"github.com/robfig/cron/v3"
)

type LoginAttempt struct { // Struct to store login attempts from syslog
	Username  string `json:"username"`
	Timestamp string `json:"timestamp"`
	Success   string `json:"success"`
}

func insert(db *sql.DB, ch chan LoginAttempt) { // Routine
	for attempt := range ch { // Insert login attempts from slice of structs
		q := "INSERT INTO logs (username, timestamp, successful) VALUES (?, ?, ?)"  // Create prepared statement
		_, err := db.Query(q, attempt.Username, attempt.Timestamp, attempt.Success) // Insert with values from struct
		if err != nil {                                                             // Log errors to console
			log.Println(err.Error())
		}
	}
} // End insert

func databaseURI() string { // Create connection string from ENV
	return fmt.Sprintf("%s:%s@tcp(%s)/%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_NAME"))
}

func ingest(logfile string, ch chan LoginAttempt) {
	file, err := os.Open(logfile) // Open logfile
	if err != nil {               // Log errors opening
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
	os.Remove(logfile)
}

func generateTimestamp(month string, day string, clock string) string {
	year := time.Now().Year() // The log doesn't store year, so assume it's this one
	m := map[string]string{   // Hacky way of mapping months to avoid the Time package
		"Jan": "01", "Feb": "02", "Mar": "03", "Apr": "04", "May": "05", "Jun": "06",
		"Jul": "07", "Aug": "08", "Sep": "09", "Oct": "10", "Nov": "11", "Dec": "12",
	}
	return fmt.Sprintf("%d-%s-%s %s", year, m[month], day, clock) // Format and return string
} // End generateTimestamp

func main() {
	db, err := sql.Open("mysql", databaseURI()) // Open database connection
	if err != nil {                             // Log failed connections
		log.Fatal("Unable to establish database connection.")
	}
	defer func(db *sql.DB) { // Handle disconnection at end of session
		err := db.Close()
		if err != nil { // Handle disconnection errors
			log.Fatal("Unable to gracefully disconnect from database.")
		}
	}(db)

	logfile := os.Getenv("LOG_PATH") // Get logfile path from ENV

	c := cron.New() // Instantiate cron

	_, err = c.AddFunc("0 4 * * *", func() { // Schedule jobs nightly
		ch := make(chan LoginAttempt)
		go ingest(logfile, ch) // Add ingest routine
		go insert(db, ch)      // Add insert routine
		log.Println("[ETL] Jobs started")
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
