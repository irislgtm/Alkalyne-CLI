package db

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/alkalyne/alkalyne/internal/models"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return db
}

func TestOpenAndMigrate(t *testing.T) {
	t.Helper()
	db := openTestDB(t)
	defer func() { _ = db.Close() }()

	var tables []string
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		tables = append(tables, name)
	}
	if len(tables) != 3 {
		t.Fatalf("expected 3 tables, got %v", tables)
	}
}

func TestStoreAndQueryMessage(t *testing.T) {
	db := openTestDB(t)
	defer func() { _ = db.Close() }()

	if err := EnsureConversation(db, "conv1", "dm", "peerA", "alice"); err != nil {
		t.Fatal(err)
	}

	msg := &models.Message{
		ID:             "msg1",
		ConversationID: "conv1",
		SenderPeerID:   "peerA",
		Text:           "hello world",
		TimestampNS:    time.Now().UnixNano(),
		LocalStatus:    models.MessageSent,
		DeliveredVia:   "direct",
	}

	if err := StoreMessage(db, msg); err != nil {
		t.Fatal(err)
	}

	msgs, err := ConversationMessages(db, "conv1", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Text != "hello world" {
		t.Fatalf("expected 'hello world', got %q", msgs[0].Text)
	}

	if err := UpdateMessageStatus(db, "msg1", models.MessageDelivered); err != nil {
		t.Fatal(err)
	}
	msgs, _ = ConversationMessages(db, "conv1", 10, 0)
	if msgs[0].LocalStatus != models.MessageDelivered {
		t.Fatalf("expected delivered status, got %q", msgs[0].LocalStatus)
	}
}

func TestConversationMessagesEmpty(t *testing.T) {
	db := openTestDB(t)
	defer func() { _ = db.Close() }()

	msgs, err := ConversationMessages(db, "nonexistent", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected empty, got %d", len(msgs))
	}
}

func TestConversationMessagesPagination(t *testing.T) {
	db := openTestDB(t)
	defer func() { _ = db.Close() }()

	if err := EnsureConversation(db, "conv_paginate", "dm", "peerA", "alice"); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		msg := &models.Message{
			ID:             fmt.Sprintf("msg%d", i),
			ConversationID: "conv_paginate",
			SenderPeerID:   "peerA",
			Text:           fmt.Sprintf("message %d", i),
			TimestampNS:    int64(i),
			LocalStatus:    models.MessageSent,
		}
		if err := StoreMessage(db, msg); err != nil {
			t.Fatal(err)
		}
	}

	msgs, err := ConversationMessages(db, "conv_paginate", 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
}

func TestAddAndListContacts(t *testing.T) {
	db := openTestDB(t)
	defer func() { _ = db.Close() }()

	c1 := models.NewContact("peer1", "alice")
	c2 := models.NewContact("peer2", "bob")

	if err := AddContact(db, c1); err != nil {
		t.Fatal(err)
	}
	if err := AddContact(db, c2); err != nil {
		t.Fatal(err)
	}

	contacts, err := ListContacts(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(contacts) != 2 {
		t.Fatalf("expected 2 contacts, got %d", len(contacts))
	}
}

func TestGetContact(t *testing.T) {
	db := openTestDB(t)
	defer func() { _ = db.Close() }()

	got, err := GetContact(db, "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("expected nil for missing contact")
	}

	c := models.NewContact("peer1", "alice")
	if err := AddContact(db, c); err != nil {
		t.Fatal(err)
	}

	got, err = GetContact(db, "peer1")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected contact, got nil")
	}
	if got.Nickname != "alice" {
		t.Fatalf("expected 'alice', got %q", got.Nickname)
	}
}

func TestUpdateContactStatus(t *testing.T) {
	db := openTestDB(t)
	defer func() { _ = db.Close() }()

	c := models.NewContact("peer1", "alice")
	if err := AddContact(db, c); err != nil {
		t.Fatal(err)
	}

	if err := UpdateContactStatus(db, "peer1", models.ContactOnline, 0); err != nil {
		t.Fatal(err)
	}

	got, _ := GetContact(db, "peer1")
	if got.Status != models.ContactOnline {
		t.Fatalf("expected online, got %q", got.Status)
	}

	if err := IncrementUnread(db, "peer1"); err != nil {
		t.Fatal(err)
	}
	got, _ = GetContact(db, "peer1")
	if got.Unread != 1 {
		t.Fatalf("expected 1 unread, got %d", got.Unread)
	}
}

func TestEnsureConversation(t *testing.T) {
	db := openTestDB(t)
	defer func() { _ = db.Close() }()

	if err := EnsureConversation(db, "conv1", "dm", "peer1", "alice"); err != nil {
		t.Fatal(err)
	}

	convs, err := ListConversations(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(convs) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(convs))
	}
	if convs[0].Kind != "dm" {
		t.Fatalf("expected dm kind, got %q", convs[0].Kind)
	}
}

func TestListConversationsEmpty(t *testing.T) {
	db := openTestDB(t)
	defer func() { _ = db.Close() }()

	convs, err := ListConversations(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(convs) != 0 {
		t.Fatalf("expected empty, got %d", len(convs))
	}
}
