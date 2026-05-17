package mailbox

import (
	"encoding/json"
	"fmt"
)

const (
	ProtocolID    = "/alkalyne/mailbox/1.0.0"
	OpStore       = "store"
	OpFetch       = "fetch"
	OpDelete      = "delete"
	OpListPending = "list_pending"
)

type Request struct {
	Op        string `json:"op"`
	TargetPID string `json:"target_peer_id"`
	MessageID string `json:"message_id,omitempty"`
	Payload   []byte `json:"payload,omitempty"`
}

type Response struct {
	OK         bool     `json:"ok"`
	Error      string   `json:"error,omitempty"`
	Messages   [][]byte `json:"messages,omitempty"`
	MessageIDs []string `json:"message_ids,omitempty"`
}

func EncodeRequest(req *Request) ([]byte, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("mailbox: encode request: %w", err)
	}
	return data, nil
}

func DecodeRequest(data []byte) (*Request, error) {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("mailbox: decode request: %w", err)
	}
	return &req, nil
}

func EncodeResponse(resp *Response) ([]byte, error) {
	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("mailbox: encode response: %w", err)
	}
	return data, nil
}

func DecodeResponse(data []byte) (*Response, error) {
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("mailbox: decode response: %w", err)
	}
	return &resp, nil
}
