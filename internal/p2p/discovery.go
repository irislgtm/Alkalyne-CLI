package p2p

import (
	"context"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

type Discovery struct {
	service mdns.Service
}

type discoveryNotifee struct {
	h host.Host
}

func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	if pi.ID == n.h.ID() {
		return
	}
	log.Printf("discovery: found peer %s", pi.ID)
	ctx := context.Background()
	if len(pi.Addrs) > 0 {
		if err := n.h.Connect(ctx, pi); err != nil {
			log.Printf("discovery: connect to %s: %v", pi.ID, err)
		}
	}
}

func NewDiscovery(h host.Host) *Discovery {
	return &Discovery{service: mdns.NewMdnsService(h, "_alkalyne._udp", &discoveryNotifee{h: h})}
}

func (d *Discovery) Start() error {
	if err := d.service.Start(); err != nil {
		return fmt.Errorf("p2p: mdns start: %w", err)
	}
	return nil
}

func (d *Discovery) Close() error {
	return d.service.Close()
}
