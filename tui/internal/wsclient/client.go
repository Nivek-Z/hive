package wsclient

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"hive-tui/internal/wsproto"
)

type Client struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func BuildURL(wsBase, token string) (string, error) {
	raw := strings.TrimRight(wsBase, "/") + "/ws"
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func Dial(ctx context.Context, wsBase, token string, events chan<- wsproto.Envelope) (*Client, error) {
	rawURL, err := BuildURL(wsBase, token)
	if err != nil {
		return nil, err
	}
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, rawURL, nil)
	if err != nil {
		return nil, err
	}
	c := &Client{conn: conn}
	go c.readLoop(events)
	go c.heartbeatLoop()
	return c, nil
}

func (c *Client) Send(frameType string, data any) error {
	frame, err := wsproto.Encode(frameType, data)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteMessage(websocket.TextMessage, frame)
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.Close()
}

func (c *Client) readLoop(events chan<- wsproto.Envelope) {
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			close(events)
			return
		}
		env, err := wsproto.Decode(raw)
		if err != nil {
			continue
		}
		events <- env
	}
}

func (c *Client) heartbeatLoop() {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := c.Send("PING", map[string]any{}); err != nil {
			return
		}
	}
}
