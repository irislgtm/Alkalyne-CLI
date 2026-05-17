package models

import "time"

type RelayStatus string

const (
	RelayOnline  RelayStatus = "online"
	RelayOffline RelayStatus = "offline"
)

type Relay struct {
	PeerID   string      `json:"peer_id"`
	Nickname string      `json:"nickname"`
	Addrs    []string    `json:"addrs"`
	AddedAt  time.Time   `json:"added_at"`
	LastSeen time.Time   `json:"last_seen"`
	Status   RelayStatus `json:"status"`
	Queued   int         `json:"queued"`
	Enabled  bool        `json:"enabled"`
}

func NewRelay(peerID, nickname string, addrs []string) *Relay {
	return &Relay{
		PeerID:   peerID,
		Nickname: nickname,
		Addrs:    addrs,
		AddedAt:  time.Now(),
		Status:   RelayOffline,
		Enabled:  true,
	}
}
