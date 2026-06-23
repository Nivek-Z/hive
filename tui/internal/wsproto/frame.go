package wsproto

import (
	"encoding/json"

	"hive-tui/internal/model"
)

type Envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type SendMessage struct {
	ChannelID int64  `json:"channelId"`
	Content   string `json:"content"`
	Type      string `json:"type,omitempty"`
	ReplyToID *int64 `json:"replyToId,omitempty"`
	Nonce     string `json:"nonce,omitempty"`
}

type Ready struct {
	User          model.User `json:"user"`
	OnlineUserIDs []int64    `json:"onlineUserIds"`
}

type MessageNew struct {
	Message model.Message `json:"message"`
	Nonce   string        `json:"nonce"`
}

type MessageDeleted struct {
	ChannelID int64 `json:"channelId"`
	MessageID int64 `json:"messageId"`
}

type ErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func Encode(frameType string, data any) ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
		Data any    `json:"data"`
	}{
		Type: frameType,
		Data: data,
	})
}

func Decode(raw []byte) (Envelope, error) {
	var env Envelope
	err := json.Unmarshal(raw, &env)
	return env, err
}
