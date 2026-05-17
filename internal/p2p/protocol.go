package p2p

import "encoding/json"

const DMTopicPrefix = "alkalyne/dm/"

const (
	MsgKindChat     = "chat"
	MsgKindPresence = "presence"
	MsgKindDM       = "dm"
)

type ChatMessage struct {
	Kind        string `json:"kind,omitempty"`
	ID          string `json:"id"`
	SenderID    string `json:"sender_id"`
	SenderName  string `json:"sender_name,omitempty"`
	RecipientID string `json:"recipient_id,omitempty"`
	Text        string `json:"text,omitempty"`
	TimestampNS int64  `json:"timestamp_ns"`
	ConvID      string `json:"conv_id,omitempty"`
}

func EncodeMessage(msg *ChatMessage) ([]byte, error) {
	return json.Marshal(msg)
}

func DecodeMessage(data []byte) (*ChatMessage, error) {
	var msg ChatMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func DMTopicName(peerA, peerB string) string {
	if peerA < peerB {
		return DMTopicPrefix + peerA + "/" + peerB
	}
	return DMTopicPrefix + peerB + "/" + peerA
}
