package models

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestNewMessage(t *testing.T) {
	m := NewMessage("peerA", "peerB", "conv1", "hello")
	if m.SenderPeerID != "peerA" {
		t.Errorf("expected peerA, got %s", m.SenderPeerID)
	}
	if m.RecipientPeerID != "peerB" {
		t.Errorf("expected peerB, got %s", m.RecipientPeerID)
	}
	if m.ConversationID != "conv1" {
		t.Errorf("expected conv1, got %s", m.ConversationID)
	}
	if m.Text != "hello" {
		t.Errorf("expected hello, got %s", m.Text)
	}
	if m.LocalStatus != MessageSent {
		t.Errorf("expected sent, got %s", m.LocalStatus)
	}
	if m.TimestampNS == 0 {
		t.Error("expected non-zero timestamp")
	}
	if m.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestMessageStatusValues(t *testing.T) {
	statuses := []MessageStatus{MessageSending, MessageSent, MessageDelivered, MessageRead, MessageFailed, MessageMailboxed}
	if len(statuses) != 6 {
		t.Errorf("expected 6 status values, got %d", len(statuses))
	}
}

func TestSignAndVerify(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	m := NewMessage("peerA", "peerB", "conv1", "hello")
	sig, err := m.Sign(priv)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if len(sig) == 0 {
		t.Fatal("expected non-empty signature")
	}

	if !m.Verify(priv.Public().(ed25519.PublicKey)) {
		t.Fatal("Verify returned false for correct signature")
	}
}

func TestVerifyFailsWithWrongKey(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	_, wrongPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	m := NewMessage("peerA", "peerB", "conv1", "hello")
	_, _ = m.Sign(priv)

	wrongPub := wrongPriv.Public().(ed25519.PublicKey)
	if m.Verify(wrongPub) {
		t.Fatal("Verify should fail with wrong public key")
	}
}

func TestNewMessageUniqueIDs(t *testing.T) {
	m1 := NewMessage("a", "b", "c", "hello")
	m2 := NewMessage("a", "b", "c", "hello")
	if m1.ID == m2.ID {
		t.Fatal("messages should have unique IDs")
	}
}
