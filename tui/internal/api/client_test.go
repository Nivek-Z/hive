package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hive-tui/internal/api"
	"hive-tui/internal/model"
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

func TestClientCoversBackendRESTSurface(t *testing.T) {
	type wantReq struct {
		method string
		path   string
		query  string
	}
	var seen []wantReq
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, wantReq{method: r.Method, path: r.URL.EscapedPath(), query: r.URL.RawQuery})
		data := any(map[string]any{"id": 1, "name": "ok"})
		switch {
		case r.URL.Path == "/api/auth/register":
			data = map[string]any{"token": "jwt", "user": map[string]any{"id": 1, "username": "neo", "nickname": "neo"}}
		case r.URL.Path == "/api/users/me" || r.URL.Path == "/api/users/9":
			data = map[string]any{"id": 9, "username": "neo", "nickname": "neo"}
		case strings.HasSuffix(r.URL.Path, "/members"):
			data = []map[string]any{{"userId": 9, "username": "neo", "nickname": "neo", "roleIds": []int64{1}}}
		case strings.HasSuffix(r.URL.Path, "/invites"):
			if r.Method == http.MethodGet {
				data = []map[string]any{{"code": "abc", "maxUses": 0, "usedCount": 0}}
			} else {
				data = map[string]any{"code": "abc", "maxUses": 0, "usedCount": 0}
			}
		case r.URL.Path == "/api/friends":
			data = []map[string]any{{"userId": 9, "username": "neo", "nickname": "neo"}}
		case r.URL.Path == "/api/friends/requests" && r.Method == http.MethodGet:
			data = []map[string]any{{"id": 7, "userId": 9, "username": "neo", "nickname": "neo"}}
		case r.URL.Path == "/api/dms":
			data = []map[string]any{{"channelId": 44, "userId": 9, "username": "neo", "nickname": "neo", "unread": 2}}
		case r.URL.Path == "/api/dms/9":
			data = map[string]any{"channelId": 44}
		case r.URL.Path == "/api/channels/2/messages":
			data = []map[string]any{{"id": 10, "channelId": 2, "content": "hello"}}
		case strings.Contains(r.URL.Path, "/reactions"):
			data = []map[string]any{{"emoji": "😀", "count": 1, "userIds": []int64{1}}}
		case strings.HasSuffix(r.URL.Path, "/roles"):
			if r.Method == http.MethodGet {
				data = []map[string]any{{"id": 3, "name": "admin", "color": "#ffb300", "permissions": 1}}
			} else {
				data = map[string]any{"id": 3, "name": "admin", "color": "#ffb300", "permissions": 1}
			}
		case r.URL.Path == "/api/users/me/achievements":
			data = []map[string]any{{"id": 1, "code": "first", "name": "First", "points": 5}}
		case r.URL.Path == "/api/users/me/heatmap":
			data = []map[string]any{{"date": "2026-06-24", "count": 3}}
		case r.URL.Path == "/api/search/messages":
			data = []map[string]any{{"id": 99, "channelId": 2, "channelName": "lobby", "content": "needle"}}
		case strings.HasSuffix(r.URL.Path, "/stats"):
			data = map[string]any{
				"daily":       []map[string]any{{"date": "2026-06-24", "count": 3}},
				"topSpeakers": []map[string]any{{"name": "neo", "count": 3}},
			}
		case r.URL.Path == "/api/files":
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
				t.Fatalf("upload Content-Type = %q", r.Header.Get("Content-Type"))
			}
			data = map[string]any{"url": "/uploads/a.png", "originalName": "a.png", "size": 4}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "ok", "data": data})
	}))
	defer srv.Close()

	dir := t.TempDir()
	imagePath := filepath.Join(dir, "a.png")
	if err := os.WriteFile(imagePath, []byte{0x89, 'P', 'N', 'G'}, 0o600); err != nil {
		t.Fatal(err)
	}

	c := api.NewClient(srv.URL)
	c.SetToken("jwt")
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}

	_, err := c.Register(ctx, "neo", "123456", "neo")
	must(err)
	_, err = c.Me(ctx)
	must(err)
	_, err = c.UpdateProfile(ctx, "neo", "bio", "#ffb300")
	must(err)
	must(c.ChangePassword(ctx, "oldpass", "newpass"))
	_, err = c.User(ctx, 9)
	must(err)
	_, err = c.CreateHive(ctx, model.HiveReq{Name: "h", Description: "d", IconColor: "#ffb300"})
	must(err)
	_, err = c.UpdateHive(ctx, 1, model.HiveReq{Name: "h", Description: "d", IconColor: "#ffb300"})
	must(err)
	must(c.DeleteHive(ctx, 1))
	must(c.LeaveHive(ctx, 1))
	_, err = c.Members(ctx, 1)
	must(err)
	must(c.KickMember(ctx, 1, 9))
	must(c.MuteMember(ctx, 1, 9, 10))
	must(c.UnmuteMember(ctx, 1, 9))
	_, err = c.CreateInvite(ctx, 1, 0, 24)
	must(err)
	_, err = c.Invites(ctx, 1)
	must(err)
	_, err = c.JoinInvite(ctx, "abc")
	must(err)
	_, err = c.CreateChannel(ctx, 1, model.CreateChannelReq{Name: "lobby", Type: "TEXT", Topic: "t"})
	must(err)
	_, err = c.UpdateChannel(ctx, 2, model.UpdateChannelReq{Name: "lobby", Topic: "t", Position: 1})
	must(err)
	must(c.DeleteChannel(ctx, 2))
	_, err = c.MessagesBefore(ctx, 2, 99, 20)
	must(err)
	must(c.DeleteMessage(ctx, 10))
	_, err = c.AddReaction(ctx, 10, "😀")
	must(err)
	_, err = c.RemoveReaction(ctx, 10, "😀")
	must(err)
	_, err = c.Friends(ctx)
	must(err)
	must(c.SendFriendRequest(ctx, "neo"))
	_, err = c.FriendRequests(ctx)
	must(err)
	must(c.AcceptFriendRequest(ctx, 7))
	must(c.DeclineFriendRequest(ctx, 7))
	must(c.RemoveFriend(ctx, 9))
	_, err = c.OpenDM(ctx, 9)
	must(err)
	_, err = c.DMs(ctx)
	must(err)
	_, err = c.Roles(ctx, 1)
	must(err)
	_, err = c.CreateRole(ctx, 1, model.RoleReq{Name: "admin", Color: "#ffb300", Permissions: 1})
	must(err)
	_, err = c.UpdateRole(ctx, 3, model.RoleReq{Name: "admin", Color: "#ffb300", Permissions: 1})
	must(err)
	must(c.DeleteRole(ctx, 3))
	must(c.AssignRoles(ctx, 1, 9, []int64{3}))
	_, err = c.UploadFile(ctx, imagePath)
	must(err)
	_, err = c.Achievements(ctx)
	must(err)
	_, err = c.Heatmap(ctx)
	must(err)
	_, err = c.SearchMessages(ctx, 1, "needle")
	must(err)
	_, err = c.HiveStats(ctx, 1)
	must(err)
	must(c.Konami(ctx))

	want := []wantReq{
		{http.MethodPost, "/api/auth/register", ""},
		{http.MethodGet, "/api/users/me", ""},
		{http.MethodPut, "/api/users/me", ""},
		{http.MethodPut, "/api/users/me/password", ""},
		{http.MethodGet, "/api/users/9", ""},
		{http.MethodPost, "/api/hives", ""},
		{http.MethodPut, "/api/hives/1", ""},
		{http.MethodDelete, "/api/hives/1", ""},
		{http.MethodPost, "/api/hives/1/leave", ""},
		{http.MethodGet, "/api/hives/1/members", ""},
		{http.MethodDelete, "/api/hives/1/members/9", ""},
		{http.MethodPost, "/api/hives/1/members/9/mute", ""},
		{http.MethodDelete, "/api/hives/1/members/9/mute", ""},
		{http.MethodPost, "/api/hives/1/invites", ""},
		{http.MethodGet, "/api/hives/1/invites", ""},
		{http.MethodPost, "/api/invites/abc/join", ""},
		{http.MethodPost, "/api/hives/1/channels", ""},
		{http.MethodPut, "/api/channels/2", ""},
		{http.MethodDelete, "/api/channels/2", ""},
		{http.MethodGet, "/api/channels/2/messages", url.Values{"before": {"99"}, "limit": {"20"}}.Encode()},
		{http.MethodDelete, "/api/messages/10", ""},
		{http.MethodPost, "/api/messages/10/reactions", ""},
		{http.MethodDelete, "/api/messages/10/reactions/%F0%9F%98%80", ""},
		{http.MethodGet, "/api/friends", ""},
		{http.MethodPost, "/api/friends/requests", ""},
		{http.MethodGet, "/api/friends/requests", ""},
		{http.MethodPost, "/api/friends/requests/7/accept", ""},
		{http.MethodDelete, "/api/friends/requests/7", ""},
		{http.MethodDelete, "/api/friends/9", ""},
		{http.MethodPost, "/api/dms/9", ""},
		{http.MethodGet, "/api/dms", ""},
		{http.MethodGet, "/api/hives/1/roles", ""},
		{http.MethodPost, "/api/hives/1/roles", ""},
		{http.MethodPut, "/api/roles/3", ""},
		{http.MethodDelete, "/api/roles/3", ""},
		{http.MethodPut, "/api/hives/1/members/9/roles", ""},
		{http.MethodPost, "/api/files", ""},
		{http.MethodGet, "/api/users/me/achievements", ""},
		{http.MethodGet, "/api/users/me/heatmap", ""},
		{http.MethodGet, "/api/search/messages", url.Values{"hiveId": {"1"}, "q": {"needle"}}.Encode()},
		{http.MethodGet, "/api/hives/1/stats", ""},
		{http.MethodPost, "/api/eggs/konami", ""},
	}
	if len(seen) != len(want) {
		t.Fatalf("seen %d requests, want %d: %#v", len(seen), len(want), seen)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("request %d = %#v, want %#v", i, seen[i], want[i])
		}
	}
}
