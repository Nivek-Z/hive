package wsclient_test

import (
	"strings"
	"testing"

	"hive-tui/internal/wsclient"
)

func TestBuildURLAppendsToken(t *testing.T) {
	got, err := wsclient.BuildURL("ws://localhost:8080", "jwt token")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(got, "ws://localhost:8080/ws?") || !strings.Contains(got, "token=jwt+token") {
		t.Fatalf("url = %s", got)
	}
}
