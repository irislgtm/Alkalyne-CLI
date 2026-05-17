package db

import (
	"database/sql"
	"fmt"
	"time"
)

type Conversation struct {
	ID        string
	Kind      string
	PeerID    string
	Nickname  string
	CreatedAt int64
	LastMsgAt int64
}

func EnsureConversation(d *sql.DB, id, kind, peerID, nickname string) error {
	_, err := d.Exec(
		`INSERT INTO conversations (id, kind, peer_id, nickname, created_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET last_msg_at = MAX(last_msg_at, excluded.created_at)`,
		id, kind, peerID, nickname, time.Now().UnixNano(),
	)
	if err != nil {
		return fmt.Errorf("db: ensure conversation: %w", err)
	}
	return nil
}

func ListConversations(d *sql.DB) ([]Conversation, error) {
	rows, err := d.Query(
		`SELECT id, kind, peer_id, nickname, created_at, last_msg_at
		 FROM conversations ORDER BY last_msg_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("db: list conversations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var convs []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.Kind, &c.PeerID, &c.Nickname, &c.CreatedAt, &c.LastMsgAt); err != nil {
			return nil, fmt.Errorf("db: scan conversation: %w", err)
		}
		convs = append(convs, c)
	}
	if convs == nil {
		convs = []Conversation{}
	}
	return convs, rows.Err()
}
