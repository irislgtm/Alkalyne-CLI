package mailbox

import (
	"encoding/json"
	"testing"
)

func TestEncodeDecodeRequest(t *testing.T) {
	t.Parallel()
	req := &Request{
		Op:        OpStore,
		TargetPID: "QmTest123",
		MessageID: "msg-001",
		Payload:   []byte("encrypted-data"),
	}

	data, err := EncodeRequest(req)
	if err != nil {
		t.Fatalf("encode request: %v", err)
	}

	decoded, err := DecodeRequest(data)
	if err != nil {
		t.Fatalf("decode request: %v", err)
	}

	if decoded.Op != req.Op {
		t.Errorf("op = %q, want %q", decoded.Op, req.Op)
	}
	if decoded.TargetPID != req.TargetPID {
		t.Errorf("target_pid = %q, want %q", decoded.TargetPID, req.TargetPID)
	}
	if decoded.MessageID != req.MessageID {
		t.Errorf("message_id = %q, want %q", decoded.MessageID, req.MessageID)
	}
	if string(decoded.Payload) != string(req.Payload) {
		t.Errorf("payload = %q, want %q", decoded.Payload, req.Payload)
	}
}

func TestDecodeRequestInvalidJSON(t *testing.T) {
	t.Parallel()
	_, err := DecodeRequest([]byte("not-json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestEncodeDecodeResponse(t *testing.T) {
	t.Parallel()
	resp := &Response{
		OK:         true,
		Messages:   [][]byte{[]byte("msg1"), []byte("msg2")},
		MessageIDs: []string{"0", "1"},
	}

	data, err := EncodeResponse(resp)
	if err != nil {
		t.Fatalf("encode response: %v", err)
	}

	decoded, err := DecodeResponse(data)
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !decoded.OK {
		t.Error("ok = false, want true")
	}
	if len(decoded.Messages) != 2 {
		t.Errorf("messages len = %d, want 2", len(decoded.Messages))
	}
	if len(decoded.MessageIDs) != 2 {
		t.Errorf("message_ids len = %d, want 2", len(decoded.MessageIDs))
	}
}

func TestDecodeResponseInvalidJSON(t *testing.T) {
	t.Parallel()
	_, err := DecodeResponse([]byte("not-json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestErrorResponse(t *testing.T) {
	t.Parallel()
	resp := &Response{
		OK:    false,
		Error: "something went wrong",
	}

	data, err := EncodeResponse(resp)
	if err != nil {
		t.Fatalf("encode response: %v", err)
	}

	decoded, err := DecodeResponse(data)
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if decoded.OK {
		t.Error("ok = true, want false")
	}
	if decoded.Error != "something went wrong" {
		t.Errorf("error = %q, want %q", decoded.Error, "something went wrong")
	}
}

func TestStoreFetch(t *testing.T) {
	t.Parallel()
	store := NewStore()

	store.Store("peer1", []byte("encrypted-msg-1"))
	store.Store("peer1", []byte("encrypted-msg-2"))
	store.Store("peer2", []byte("encrypted-msg-3"))

	msgs1 := store.Fetch("peer1")
	if len(msgs1) != 2 {
		t.Fatalf("peer1 msgs = %d, want 2", len(msgs1))
	}

	msgs1After := store.Fetch("peer1")
	if len(msgs1After) != 0 {
		t.Fatalf("peer1 msgs after fetch = %d, want 0", len(msgs1After))
	}

	msgs2 := store.Fetch("peer2")
	if len(msgs2) != 1 {
		t.Fatalf("peer2 msgs = %d, want 1", len(msgs2))
	}
}

func TestListPending(t *testing.T) {
	t.Parallel()
	store := NewStore()

	store.Store("peer1", []byte("msg1"))
	store.Store("peer1", []byte("msg2"))

	ids := store.ListPending("peer1")
	if len(ids) != 2 {
		t.Fatalf("pending ids = %d, want 2", len(ids))
	}

	empty := store.ListPending("peer2")
	if len(empty) != 0 {
		t.Fatalf("peer2 pending = %d, want 0", len(empty))
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()
	store := NewStore()

	store.Store("peer1", []byte("msg1"))
	store.Store("peer1", []byte("msg2"))
	store.Store("peer1", []byte("msg3"))

	ok := store.Delete("peer1", "1")
	if !ok {
		t.Fatal("delete should succeed")
	}

	ids := store.ListPending("peer1")
	if len(ids) != 2 {
		t.Fatalf("pending after delete = %d, want 2", len(ids))
	}

	ok = store.Delete("peer1", "999")
	if ok {
		t.Fatal("delete non-existent should fail")
	}
}

func TestRequestJSONRoundtrip(t *testing.T) {
	t.Parallel()
	req := &Request{
		Op:        OpStore,
		TargetPID: "QmPeer",
		Payload:   []byte{0x01, 0x02, 0x03},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Op != req.Op {
		t.Errorf("op mismatch")
	}
	if decoded.TargetPID != req.TargetPID {
		t.Errorf("target_pid mismatch")
	}
}
