// Package repository repository/seed.go
package repository

import (
	"database/sql"
	"fmt"
	"log"
)

func InitDB(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		amount INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'PENDING'
	)
	`)
	if err != nil {
		return err
	}

	if err := addColumnIfNotExist(
		db, "orders", "status", "TEXT NOT NULL DEFAULT 'PENDING'"); err != nil {
		return err
	}
	return createOutbox(db)
}

func createOutbox(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS outbox (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_id TEXT NOT NULL UNIQUE,
		subject TEXT NOT NULL,
		payload TEXT NOT NULL,
		published INTEGER NOT NULL DEFAULT 0
	)
	`)
	if err != nil {
		return fmt.Errorf("error when creating table outbox: %s", err)
	}
	log.Print("successfully created outbox table")
	return nil
}

func addColumnIfNotExist(db *sql.DB, table, column, definition string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     string
			notnull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
	return err
}
