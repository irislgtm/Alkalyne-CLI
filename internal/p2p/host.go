package p2p

import (
	"fmt"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
)

func NewHost(privKey crypto.PrivKey, listenAddrs []string) (host.Host, error) {
	if len(listenAddrs) == 0 {
		listenAddrs = []string{"/ip4/0.0.0.0/tcp/0"}
	}

	h, err := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(listenAddrs...),
		libp2p.DefaultTransports,
	)
	if err != nil {
		return nil, fmt.Errorf("p2p: create host: %w", err)
	}

	return h, nil
}
