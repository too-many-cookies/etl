package main

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestDatabaseURI(t *testing.T) {
	os.Setenv("DB_USER", "user")
	defer os.Unsetenv("DB_USER")

	os.Setenv("DB_PASS", "password")
	defer os.Unsetenv("DB_PASS")

	os.Setenv("DB_HOST", "10.10.10.10")
	defer os.Unsetenv("DB_HOST")

	os.Setenv("DB_NAME", "test_db")
	defer os.Unsetenv("DB_NAME")

	want := "user:password@tcp(10.10.10.10)/test_db"
	msg := databaseURI()
	if msg != want {
		t.Fatal()
	}
}

func TestGenerateTimestamp(t *testing.T) {
	want := fmt.Sprintf("%d-05-04 01:30:00", time.Now().Year())
	msg := generateTimestamp("May", "04", "01:30:00") // Check result is of current year
	if msg != want {
		t.Fatal()
	}
}
