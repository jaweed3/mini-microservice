// Package repository here for seeding database
package repository

import (
	"database/sql"
	"fmt"
)

func InitDBAudit(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS audit_service (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_id TEXT NOT NULL UNIQUE,
		request_id TEXT NOT NULL,
		order_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		amount INTEGER NOT NULL,
		action TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'PENDING',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)
	`)
	if err != nil {
		return err
	}
	return addColumnIfNotExist(db, "audit_service", "status", "TEXT NOT NULL DEFAULT 'PENDING'")
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
