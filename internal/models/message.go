package models

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type MessageStatus string

const (
	MessageSending   MessageStatus = "sending"
	MessageSent      MessageStatus = "sent"
	MessageDelivered MessageStatus = "delivered"
	MessageRead      MessageStatus = "read"
	MessageFailed    MessageStatus = "failed"
	MessageMailboxed MessageStatus = "mailboxed"
)

type Message struct {
	ID              string        `json:"id"`
	ConversationID  string        `json:"conversation_id"`
	SenderPeerID    string        `json:"sender_peer_id"`
	RecipientPeerID string        `json:"recipient_peer_id"`
	Text            string        `json:"text"`
	TimestampNS     int64         `json:"timestamp_ns"`
	Signature       []byte        `json:"signature"`
	LocalStatus     MessageStatus `json:"local_status"`
	DeliveredVia    string        `json:"delivered_via,omitempty"`
}

func NewMessage(senderPeerID, recipientPeerID, conversationID, text string) *Message {
	return &Message{
		ID:              generateID(),
		SenderPeerID:    senderPeerID,
		RecipientPeerID: recipientPeerID,
		ConversationID:  conversationID,
		Text:            text,
		TimestampNS:     time.Now().UnixNano(),
		LocalStatus:     MessageSent,
	}
}

func (m *Message) Sign(privateKey ed25519.PrivateKey) ([]byte, error) {
	data := m.signingData()
	sig := ed25519.Sign(privateKey, data)
	m.Signature = sig
	return sig, nil
}

func (m *Message) Verify(publicKey ed25519.PublicKey) bool {
	return ed25519.Verify(publicKey, m.signingData(), m.Signature)
}

func (m *Message) VerifyFromPeerID(id peer.ID) bool {
	pubKey, err := id.ExtractPublicKey()
	if err != nil {
		return false
	}
	rawKey, err := pubKey.Raw()
	if err != nil {
		return false
	}
	return m.Verify(ed25519.PublicKey(rawKey))
}

func (m *Message) signingData() []byte {
	n := len(m.ID) + len(m.SenderPeerID) + len(m.RecipientPeerID) + len(m.ConversationID) + len(m.Text) + 8
	b := make([]byte, 0, n)
	b = append(b, []byte(m.ID)...)
	b = append(b, []byte(m.SenderPeerID)...)
	b = append(b, []byte(m.RecipientPeerID)...)
	b = append(b, []byte(m.ConversationID)...)
	b = append(b, []byte(m.Text)...)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(m.TimestampNS))
	b = append(b, buf...)
	return b
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
