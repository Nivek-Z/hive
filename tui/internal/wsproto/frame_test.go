package wsproto_test

import (
	"encoding/json"
	"strings"
	"testing"

	"hive-tui/internal/model"
	"hive-tui/internal/wsproto"
)

func TestEncodeSendMessageFrame(t *testing.T) {
	frame, err := wsproto.Encode("MSG_SEND", wsproto.SendMessage{
		ChannelID: 2,
		Content:   "hello",
		Type:      "TEXT",
		Nonce:     "n1",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(frame), `"type":"MSG_SEND"`) || !strings.Contains(string(frame), `"nonce":"n1"`) {
		t.Fatalf("unexpected frame: %s", frame)
	}
}

func TestDecodeMessageEnvelope(t *testing.T) {
	raw := []byte(`{"type":"MSG_NEW","data":{"message":{"id":10,"channelId":2,"content":"hello"},"nonce":"n1"}}`)

	env, err := wsproto.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}

	var payload wsproto.MessageNew
	if err := json.Unmarshal(env.Data, &payload); err != nil {
		t.Fatal(err)
	}
	if env.Type != "MSG_NEW" || payload.Nonce != "n1" || payload.Message.Content != "hello" {
		t.Fatalf("env = %#v payload = %#v", env, payload)
	}
}

func TestMessageDeletedPayloadMatchesServerShape(t *testing.T) {
	payload := wsproto.MessageDeleted{ChannelID: 2, MessageID: 10}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), `"messageId":10`) {
		t.Fatalf("payload = %s", data)
	}
}

func TestReadyPayloadIncludesUserAndOnlineIDs(t *testing.T) {
	raw := []byte(`{"user":{"id":1,"username":"afeng"},"onlineUserIds":[1,2]}`)
	var ready wsproto.Ready
	if err := json.Unmarshal(raw, &ready); err != nil {
		t.Fatal(err)
	}

	if ready.User.Username != "afeng" || len(ready.OnlineUserIDs) != 2 {
		t.Fatalf("ready = %#v", ready)
	}
}

func TestErrorPayload(t *testing.T) {
	errPayload := wsproto.ErrorPayload{Code: 403, Message: "没有权限"}
	if errPayload.Message != "没有权限" {
		t.Fatalf("payload = %#v", errPayload)
	}
}

var _ = model.Message{}
