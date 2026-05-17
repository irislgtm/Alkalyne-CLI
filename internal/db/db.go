package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("db: wal: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("db: foreign_keys: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("db: migrate: %w", err)
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS conversations (
		id          TEXT PRIMARY KEY,
		kind        TEXT NOT NULL DEFAULT 'dm',
		peer_id     TEXT NOT NULL DEFAULT '',
		nickname    TEXT NOT NULL DEFAULT '',
		created_at  INTEGER NOT NULL,
		last_msg_at INTEGER NOT NULL DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS messages (
		id              TEXT PRIMARY KEY,
		conversation_id TEXT NOT NULL,
		sender_peer_id  TEXT NOT NULL,
		text            TEXT NOT NULL,
		timestamp_ns    INTEGER NOT NULL,
		local_status    TEXT NOT NULL DEFAULT 'sending',
		delivered_via   TEXT NOT NULL DEFAULT '',
		FOREIGN KEY (conversation_id) REFERENCES conversations(id)
	);
	CREATE TABLE IF NOT EXISTS contacts (
		peer_id   TEXT PRIMARY KEY,
		nickname  TEXT NOT NULL DEFAULT '',
		added_at  INTEGER NOT NULL,
		last_seen INTEGER NOT NULL DEFAULT 0,
		status    TEXT NOT NULL DEFAULT 'offline',
		unread    INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_messages_conv ON messages(conversation_id, timestamp_ns);
	`
	_, err := db.Exec(schema)
	return err
}
