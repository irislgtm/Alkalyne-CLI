package models

import "time"

type ContactStatus string

const (
	ContactOnline  ContactStatus = "online"
	ContactOffline ContactStatus = "offline"
	ContactPending ContactStatus = "pending"
)

type Contact struct {
	PeerID   string        `json:"peer_id"`
	Nickname string        `json:"nickname"`
	AddedAt  time.Time     `json:"added_at"`
	LastSeen time.Time     `json:"last_seen"`
	Status   ContactStatus `json:"status"`
	Unread   int           `json:"unread"`
}

func NewContact(peerID, nickname string) *Contact {
	return &Contact{
		PeerID:   peerID,
		Nickname: nickname,
		AddedAt:  time.Now(),
		Status:   ContactPending,
	}
}
