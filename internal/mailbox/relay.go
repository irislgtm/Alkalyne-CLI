package mailbox

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

type Store struct {
	mu       sync.RWMutex
	messages map[string][][]byte
}

func NewStore() *Store {
	return &Store{
		messages: make(map[string][][]byte),
	}
}

func (s *Store) Store(peerID string, payload []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[peerID] = append(s.messages[peerID], payload)
}

func (s *Store) Fetch(peerID string) [][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	msgs := s.messages[peerID]
	delete(s.messages, peerID)
	return msgs
}

func (s *Store) ListPending(peerID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs := s.messages[peerID]
	ids := make([]string, len(msgs))
	for i := range msgs {
		ids[i] = fmt.Sprintf("%d", i)
	}
	return ids
}

func (s *Store) Delete(peerID string, messageID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	msgs := s.messages[peerID]
	for i := range msgs {
		if fmt.Sprintf("%d", i) == messageID {
			s.messages[peerID] = append(msgs[:i], msgs[i+1:]...)
			return true
		}
	}
	return false
}

type Relay struct {
	host  host.Host
	store *Store
}

func NewRelay(h host.Host, store *Store) *Relay {
	return &Relay{
		host:  h,
		store: store,
	}
}

func (r *Relay) Start(ctx context.Context) error {
	r.host.SetStreamHandler(ProtocolID, r.handleStream)
	return nil
}

func (r *Relay) handleStream(stream network.Stream) {
	defer stream.Close()

	reader := bufio.NewReader(stream)
	reqData, err := io.ReadAll(reader)
	if err != nil {
		r.sendError(stream, fmt.Errorf("read request: %w", err))
		return
	}

	req, err := DecodeRequest(reqData)
	if err != nil {
		r.sendError(stream, err)
		return
	}

	resp := r.processRequest(req)
	respData, err := EncodeResponse(resp)
	if err != nil {
		r.sendError(stream, err)
		return
	}

	_, _ = stream.Write(respData)
}

func (r *Relay) processRequest(req *Request) *Response {
	switch req.Op {
	case OpStore:
		if req.TargetPID == "" || len(req.Payload) == 0 {
			return &Response{OK: false, Error: "mailbox: store requires target_peer_id and payload"}
		}
		r.store.Store(req.TargetPID, req.Payload)
		return &Response{OK: true}

	case OpFetch:
		if req.TargetPID == "" {
			return &Response{OK: false, Error: "mailbox: fetch requires target_peer_id"}
		}
		msgs := r.store.Fetch(req.TargetPID)
		return &Response{OK: true, Messages: msgs}

	case OpListPending:
		if req.TargetPID == "" {
			return &Response{OK: false, Error: "mailbox: list_pending requires target_peer_id"}
		}
		ids := r.store.ListPending(req.TargetPID)
		return &Response{OK: true, MessageIDs: ids}

	case OpDelete:
		if req.TargetPID == "" || req.MessageID == "" {
			return &Response{OK: false, Error: "mailbox: delete requires target_peer_id and message_id"}
		}
		ok := r.store.Delete(req.TargetPID, req.MessageID)
		if !ok {
			return &Response{OK: false, Error: "mailbox: message not found"}
		}
		return &Response{OK: true}

	default:
		return &Response{OK: false, Error: fmt.Sprintf("mailbox: unknown op: %s", req.Op)}
	}
}

func (r *Relay) sendError(stream network.Stream, err error) {
	resp := &Response{OK: false, Error: err.Error()}
	respData, marshalErr := EncodeResponse(resp)
	if marshalErr != nil {
		return
	}
	_, _ = stream.Write(respData)
}
