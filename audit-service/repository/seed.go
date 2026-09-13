// Package repository here for seeding database
package repository

import "database/sql"

func InitDBAudit(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS audit_service (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_id TEXT NOT NULL UNIQUE,
		order_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		amount INTEGER NOT NULL,
		action TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)
	`

	_, err := db.Exec(query)
	return err
}
