package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq" // registers postgres driver with database/sql
)

// DB is global connection pool shared across all packages. It is initialized in Connect() and should be used for all database operations.
var DB *sql.DB

// Connect opens and verifies a connection to the PostgreSQL database using values from .env.
// Called once at startup. Kills the server if connection fails.
func Connect() {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")

	dsn := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s  sslmode=disable",
		dbHost, dbPort, dbName, dbUser, dbPassword,
	)

	var err error

	// sql.Open only validates the DSN, does not connect yet.
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	// Connection pool limits — without these, Go's defaults (unlimited open
	// connections, only 2 idle) cause unpredictable latency under concurrent
	// load, since every burst of requests fights for new connections instead
	// of reusing a stable pool.
	DB.SetMaxOpenConns(25)                 // hard cap on concurrent connections
	DB.SetMaxIdleConns(10)                 // keep some warm connections ready to reuse
	DB.SetConnMaxLifetime(5 * time.Minute) // recycle connections periodically

	// .Ping() verifies that the database is reachable and the connection is valid.
	err = DB.Ping()
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	log.Println("Database connected successfully")
}
