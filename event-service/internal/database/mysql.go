package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func NewMySQLConnection(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("could not ping database: %w", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("could not create tables: %w", err)
	}

	log.Println("Connected to MySQL database")
	return db, nil
}

func createTables(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS events (
		id VARCHAR(36) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		date DATETIME NOT NULL,
		location VARCHAR(255) NOT NULL,
		ticket_stock INT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`

	_, err := db.Exec(query)
	return err
}
