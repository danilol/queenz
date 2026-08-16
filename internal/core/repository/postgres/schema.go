package postgres

import (
	"context"
	"fmt"
)

// EnsureSchema runs idempotent DDL queries to create the necessary relational tables
// for the Core and Ingestion domains if they do not already exist.
func EnsureSchema(ctx context.Context, db DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS franchises (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			country VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS seasons (
			id VARCHAR(255) PRIMARY KEY,
			franchise_id VARCHAR(255) REFERENCES franchises(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			number INT NOT NULL,
			air_date TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS episodes (
			id VARCHAR(255) PRIMARY KEY,
			season_id VARCHAR(255) REFERENCES seasons(id) ON DELETE CASCADE,
			title VARCHAR(255) NOT NULL,
			number INT NOT NULL,
			air_date TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS persons (
			id VARCHAR(255) PRIMARY KEY,
			drag_name VARCHAR(255) NOT NULL,
			real_name VARCHAR(255) NOT NULL,
			birth_place VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS ingestion_jobs (
			id VARCHAR(255) PRIMARY KEY,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			progress TEXT NOT NULL DEFAULT '',
			error_msg TEXT,
			retries INT NOT NULL DEFAULT 0,
			max_retries INT NOT NULL DEFAULT 3,
			locked_until TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);`,
	}

	for _, query := range queries {
		if _, err := db.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to execute schema query: %w", err)
		}
	}

	return nil
}
