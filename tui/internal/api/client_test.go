package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hive-tui/internal/api"
)

func TestClientLoginDecodesTokenAndUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/login" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"msg":  "ok",
			"data": map[string]any{
				"token": "jwt",
				"user":  map[string]any{"id": 1, "username": "afeng", "nickname": "阿蜂"},
			},
		})
	}))
	defer srv.Close()

	c := api.NewClient(srv.URL)
	resp, err := c.Login(context.Background(), "afeng", "123456")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Token != "jwt" || resp.User.Username != "afeng" || resp.User.Nickname != "阿蜂" {
		t.Fatalf("login = %#v", resp)
	}
}

func TestClientReturnsBusinessErrorMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 403,
			"msg":  "没有权限",
			"data": nil,
		})
	}))
	defer srv.Close()

	c := api.NewClient(srv.URL)
	_, err := c.Login(context.Background(), "afeng", "bad")
	if err == nil || err.Error() != "没有权限" {
		t.Fatalf("err = %v", err)
	}
}

func TestClientSendsBearerTokenForAuthenticatedRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer jwt" {
			t.Fatalf("Authorization = %q", got)
		}
		if r.URL.Path != "/api/hives" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"msg":  "ok",
			"data": []map[string]any{{"id": 1, "name": "Java 大作业交流群", "iconColor": "#ffb300"}},
		})
	}))
	defer srv.Close()

	c := api.NewClient(srv.URL)
	c.SetToken("jwt")
	hives, err := c.Hives(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(hives) != 1 || hives[0].Name != "Java 大作业交流群" {
		t.Fatalf("hives = %#v", hives)
	}
}

func TestClientLoadsMessagesWithLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/channels/2/messages" || r.URL.Query().Get("limit") != "50" {
			t.Fatalf("url = %s", r.URL.String())
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"msg":  "ok",
			"data": []map[string]any{{"id": 10, "channelId": 2, "senderNickname": "阿蜂", "type": "TEXT", "content": "hello"}},
		})
	}))
	defer srv.Close()

	c := api.NewClient(srv.URL)
	messages, err := c.Messages(context.Background(), 2, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || messages[0].Content != "hello" {
		t.Fatalf("messages = %#v", messages)
	}
}
