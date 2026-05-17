package mailbox

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Client struct {
	host     host.Host
	relayPID peer.ID
	timeout  time.Duration
}

func NewClient(h host.Host, relayPID peer.ID) *Client {
	return &Client{
		host:     h,
		relayPID: relayPID,
		timeout:  10 * time.Second,
	}
}

func (c *Client) Store(ctx context.Context, targetPID string, payload []byte) error {
	req := &Request{
		Op:        OpStore,
		TargetPID: targetPID,
		Payload:   payload,
	}
	return c.doRequest(ctx, req)
}

func (c *Client) Fetch(ctx context.Context, targetPID string) ([][]byte, error) {
	req := &Request{
		Op:        OpFetch,
		TargetPID: targetPID,
	}
	resp, err := c.doRequestWithResponse(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Messages, nil
}

func (c *Client) ListPending(ctx context.Context, targetPID string) ([]string, error) {
	req := &Request{
		Op:        OpListPending,
		TargetPID: targetPID,
	}
	resp, err := c.doRequestWithResponse(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.MessageIDs, nil
}

func (c *Client) Delete(ctx context.Context, targetPID string, messageID string) error {
	req := &Request{
		Op:        OpDelete,
		TargetPID: targetPID,
		MessageID: messageID,
	}
	return c.doRequest(ctx, req)
}

func (c *Client) doRequest(ctx context.Context, req *Request) error {
	resp, err := c.doRequestWithResponse(ctx, req)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("mailbox: %s", resp.Error)
	}
	return nil
}

func (c *Client) doRequestWithResponse(ctx context.Context, req *Request) (*Response, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	stream, err := c.host.NewStream(ctx, c.relayPID, ProtocolID)
	if err != nil {
		return nil, fmt.Errorf("mailbox: open stream: %w", err)
	}
	defer stream.Close()

	reqData, err := EncodeRequest(req)
	if err != nil {
		return nil, err
	}

	_, err = stream.Write(reqData)
	if err != nil {
		return nil, fmt.Errorf("mailbox: write request: %w", err)
	}

	_ = stream.CloseWrite()

	reader := bufio.NewReader(stream)
	respData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("mailbox: read response: %w", err)
	}

	return DecodeResponse(respData)
}
