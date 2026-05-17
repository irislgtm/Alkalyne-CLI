package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/alkalyne/alkalyne/internal/models"
)

func AddContact(d *sql.DB, c *models.Contact) error {
	_, err := d.Exec(
		`INSERT INTO contacts (peer_id, nickname, added_at, last_seen, status, unread)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(peer_id) DO UPDATE SET nickname=excluded.nickname`,
		c.PeerID, c.Nickname, c.AddedAt.UnixNano(), c.LastSeen.UnixNano(),
		c.Status, c.Unread,
	)
	if err != nil {
		return fmt.Errorf("db: add contact: %w", err)
	}
	return nil
}

func ListContacts(d *sql.DB) ([]*models.Contact, error) {
	rows, err := d.Query(
		`SELECT peer_id, nickname, added_at, last_seen, status, unread
		 FROM contacts ORDER BY last_seen DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("db: list contacts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var contacts []*models.Contact
	for rows.Next() {
		c := &models.Contact{}
		var addedNs, lastNs int64
		if err := rows.Scan(&c.PeerID, &c.Nickname, &addedNs, &lastNs, &c.Status, &c.Unread); err != nil {
			return nil, fmt.Errorf("db: scan contact: %w", err)
		}
		c.AddedAt = time.Unix(0, addedNs)
		c.LastSeen = time.Unix(0, lastNs)
		contacts = append(contacts, c)
	}
	if contacts == nil {
		contacts = []*models.Contact{}
	}
	return contacts, rows.Err()
}

func GetContact(d *sql.DB, peerID string) (*models.Contact, error) {
	c := &models.Contact{}
	var addedNs, lastNs int64
	err := d.QueryRow(
		`SELECT peer_id, nickname, added_at, last_seen, status, unread
		 FROM contacts WHERE peer_id = ?`, peerID,
	).Scan(&c.PeerID, &c.Nickname, &addedNs, &lastNs, &c.Status, &c.Unread)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("db: get contact: %w", err)
	}
	c.AddedAt = time.Unix(0, addedNs)
	c.LastSeen = time.Unix(0, lastNs)
	return c, nil
}

func UpdateContactStatus(d *sql.DB, peerID string, status models.ContactStatus, unread int) error {
	_, err := d.Exec(
		`UPDATE contacts SET status = ?, last_seen = ?, unread = ? WHERE peer_id = ?`,
		status, time.Now().UnixNano(), unread, peerID,
	)
	if err != nil {
		return fmt.Errorf("db: update contact: %w", err)
	}
	return nil
}

func IncrementUnread(d *sql.DB, peerID string) error {
	_, err := d.Exec(`UPDATE contacts SET unread = unread + 1 WHERE peer_id = ?`, peerID)
	if err != nil {
		return fmt.Errorf("db: inc unread: %w", err)
	}
	return nil
}
