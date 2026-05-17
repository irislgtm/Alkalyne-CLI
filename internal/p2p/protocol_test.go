package p2p

import (
	"testing"
)

func TestEncodeDecodeRoundtrip(t *testing.T) {
	original := &ChatMessage{
		ID:          "msg1",
		SenderID:    "peerA",
		SenderName:  "alice",
		Text:        "hello world",
		TimestampNS: 1234567890,
	}

	data, err := EncodeMessage(original)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("empty encoded data")
	}

	decoded, err := DecodeMessage(data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.ID != original.ID {
		t.Fatalf("id: expected %q, got %q", original.ID, decoded.ID)
	}
	if decoded.Text != original.Text {
		t.Fatalf("text: expected %q, got %q", original.Text, decoded.Text)
	}
	if decoded.SenderID != original.SenderID {
		t.Fatalf("sender: expected %q, got %q", original.SenderID, decoded.SenderID)
	}
}

func TestEncodeDecodeEmptyText(t *testing.T) {
	msg := &ChatMessage{
		ID:       "empty",
		SenderID: "peerB",
		Text:     "",
	}

	data, err := EncodeMessage(msg)
	if err != nil {
		t.Fatal(err)
	}

	decoded, err := DecodeMessage(data)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Text != "" {
		t.Fatalf("expected empty text, got %q", decoded.Text)
	}
}

func TestDecodeInvalidData(t *testing.T) {
	_, err := DecodeMessage([]byte("{invalid json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestEncodeDecodeMinimal(t *testing.T) {
	msg := &ChatMessage{
		ID:       "minimal",
		SenderID: "peerC",
		Text:     "hi",
	}

	data, err := EncodeMessage(msg)
	if err != nil {
		t.Fatal(err)
	}

	decoded, err := DecodeMessage(data)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.SenderName != "" {
		t.Fatalf("expected empty sender name, got %q", decoded.SenderName)
	}
	if decoded.TimestampNS != 0 {
		t.Fatalf("expected timestamp 0, got %d", decoded.TimestampNS)
	}
}
