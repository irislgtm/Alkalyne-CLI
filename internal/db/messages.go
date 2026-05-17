package db

import (
	"database/sql"
	"fmt"

	"github.com/alkalyne/alkalyne/internal/models"
)

func StoreMessage(d *sql.DB, msg *models.Message) error {
	_, err := d.Exec(
		`INSERT INTO messages (id, conversation_id, sender_peer_id, text, timestamp_ns, local_status, delivered_via)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		msg.ID, msg.ConversationID, msg.SenderPeerID, msg.Text, msg.TimestampNS,
		msg.LocalStatus, msg.DeliveredVia,
	)
	if err != nil {
		return fmt.Errorf("db: store message: %w", err)
	}
	return nil
}

func ConversationMessages(d *sql.DB, convID string, limit, offset int) ([]*models.Message, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := d.Query(
		`SELECT id, conversation_id, sender_peer_id, text, timestamp_ns, local_status, COALESCE(delivered_via, '')
		 FROM messages WHERE conversation_id = ?
		 ORDER BY timestamp_ns DESC LIMIT ? OFFSET ?`,
		convID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("db: conv messages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var msgs []*models.Message
	for rows.Next() {
		m := &models.Message{}
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderPeerID, &m.Text,
			&m.TimestampNS, &m.LocalStatus, &m.DeliveredVia); err != nil {
			return nil, fmt.Errorf("db: scan message: %w", err)
		}
		msgs = append(msgs, m)
	}
	if msgs == nil {
		msgs = []*models.Message{}
	}
	return msgs, rows.Err()
}

func UpdateMessageStatus(d *sql.DB, id string, status models.MessageStatus) error {
	_, err := d.Exec(`UPDATE messages SET local_status = ? WHERE id = ?`, status, id)
	if err != nil {
		return fmt.Errorf("db: update msg status: %w", err)
	}
	return nil
}
