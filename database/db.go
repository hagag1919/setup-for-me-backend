package database

import (
	"database/sql"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func InitDB() (*sql.DB, error) {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Use the DATABASE_URL from environment variables
	connectionString := os.Getenv("DATABASE_URL")
	if connectionString == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Create tables
	if err := createTables(db); err != nil {
		return nil, err
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	// Users table
	userSchema := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		email VARCHAR(255) NOT NULL UNIQUE,
		password VARCHAR(255) NOT NULL
	);`

	if _, err := db.Exec(userSchema); err != nil {
		return err
	}

	// Apps table
	appSchema := `
	CREATE TABLE IF NOT EXISTS apps (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL,
		name VARCHAR(255),
		winget_id VARCHAR(255),
		download_url TEXT,
		args TEXT,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);`

	if _, err := db.Exec(appSchema); err != nil {
		return err
	}

	return nil
}
