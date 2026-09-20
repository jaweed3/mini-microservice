package repository

import "database/sql"

func InitExporterDB(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS exported_orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_id TEXT NOT NULL UNIQUE,
		order_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		amount INTEGER NOT NULL,
		action TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'PENDING',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)
	`

	_, err := db.Exec(query)
	return err
}
