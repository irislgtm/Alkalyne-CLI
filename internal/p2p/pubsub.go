package p2p

import (
	"context"
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

const LobbyTopic = "alkalyne/lobby"

func NewPubSub(ctx context.Context, h host.Host) (*pubsub.PubSub, error) {
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("p2p: new pubsub: %w", err)
	}
	return ps, nil
}

func JoinTopic(ps *pubsub.PubSub, topic string) (*pubsub.Topic, *pubsub.Subscription, error) {
	t, err := ps.Join(topic)
	if err != nil {
		return nil, nil, fmt.Errorf("p2p: join topic %s: %w", topic, err)
	}
	sub, err := t.Subscribe()
	if err != nil {
		return nil, nil, fmt.Errorf("p2p: subscribe %s: %w", topic, err)
	}
	return t, sub, nil
}

func Publish(ctx context.Context, topic *pubsub.Topic, data []byte) error {
	return topic.Publish(ctx, data)
}
