package p2p

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
)

func NewHost(privKey crypto.PrivKey, listenAddrs []string, enableRelay bool) (host.Host, error) {
	if len(listenAddrs) == 0 {
		listenAddrs = []string{"/ip4/0.0.0.0/tcp/0"}
	}

	opts := []libp2p.Option{
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(listenAddrs...),
		libp2p.DefaultTransports,
		libp2p.EnableRelay(),
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("p2p: create host: %w", err)
	}

	if enableRelay {
		_, err := relay.New(h)
		if err != nil {
			return nil, fmt.Errorf("p2p: enable relay: %w", err)
		}
	}

	return h, nil
}

func SetupDHT(ctx context.Context, h host.Host) (*dht.IpfsDHT, error) {
	d, err := dht.New(ctx, h, dht.Mode(dht.ModeAuto))
	if err != nil {
		return nil, fmt.Errorf("p2p: create dht: %w", err)
	}
	return d, nil
}

func BootstrapDHT(ctx context.Context, dht *dht.IpfsDHT) error {
	if err := dht.Bootstrap(ctx); err != nil {
		return fmt.Errorf("p2p: bootstrap dht: %w", err)
	}
	return nil
}

func ConnectToPeers(ctx context.Context, h host.Host, addrs []string) []error {
	var errs []error
	for _, addr := range addrs {
		pi, err := peer.AddrInfoFromString(addr)
		if err != nil {
			errs = append(errs, fmt.Errorf("p2p: parse %s: %w", addr, err))
			continue
		}
		dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		if err := h.Connect(dialCtx, *pi); err != nil {
			errs = append(errs, fmt.Errorf("p2p: connect %s: %w", addr, err))
		}
		cancel()
	}
	return errs
}
