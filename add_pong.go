package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set in environment")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	fmt.Println("✅ Connected to database successfully!")

	// Insert Pong environment
	query := `
		INSERT INTO environments (id, name, description, version, config)
		VALUES
		    (
		        'pong',
		        'Pong',
		        'Classic Pong game with paddles and ball',
		        '1.0.0',
		        '{"fieldWidth": 800, "fieldHeight": 400, "paddleHeight": 80, "paddleWidth": 12, "ballRadius": 8, "maxScore": 11, "timeLimit": 300000}'::jsonb
		    )
		ON CONFLICT (id) DO UPDATE SET
		    name = EXCLUDED.name,
		    description = EXCLUDED.description,
		    version = EXCLUDED.version,
		    config = EXCLUDED.config,
		    updated_at = NOW()
	`

	_, err = db.Exec(query)
	if err != nil {
		log.Fatal("Failed to insert Pong environment:", err)
	}

	fmt.Println("✅ Pong environment added successfully!")

	// Verify insertion
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM environments WHERE id = 'pong'").Scan(&count)
	if err != nil {
		log.Fatal("Failed to verify insertion:", err)
	}

	if count == 1 {
		fmt.Println("✅ Pong environment verified in database!")
	} else {
		fmt.Println("❌ Pong environment not found after insertion")
	}

	// List all environments
	fmt.Println("\n📋 All available environments:")
	rows, err := db.Query("SELECT id, name, description FROM environments ORDER BY name")
	if err != nil {
		log.Fatal("Failed to query environments:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, name, description string
		err := rows.Scan(&id, &name, &description)
		if err != nil {
			log.Fatal("Failed to scan environment:", err)
		}
		fmt.Printf("  - %s: %s (%s)\n", id, name, description)
	}
}
