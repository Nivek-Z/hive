package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"hive-tui/internal/model"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    http.DefaultClient,
	}
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) Login(ctx context.Context, username, password string) (model.LoginResp, error) {
	var resp model.LoginResp
	err := c.do(ctx, http.MethodPost, "/auth/login", map[string]string{
		"username": username,
		"password": password,
	}, &resp)
	return resp, err
}

func (c *Client) Hives(ctx context.Context) ([]model.Hive, error) {
	var hives []model.Hive
	err := c.do(ctx, http.MethodGet, "/hives", nil, &hives)
	return hives, err
}

func (c *Client) HiveDetail(ctx context.Context, id int64) (model.HiveDetail, error) {
	var detail model.HiveDetail
	err := c.do(ctx, http.MethodGet, "/hives/"+strconv.FormatInt(id, 10), nil, &detail)
	return detail, err
}

func (c *Client) Messages(ctx context.Context, channelID int64, limit int) ([]model.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	path := "/channels/" + strconv.FormatInt(channelID, 10) + "/messages?limit=" + url.QueryEscape(strconv.Itoa(limit))
	var messages []model.Message
	err := c.do(ctx, http.MethodGet, path, nil, &messages)
	return messages, err
}

func (c *Client) MarkRead(ctx context.Context, channelID, lastMessageID int64) error {
	return c.do(ctx, http.MethodPost, "/channels/"+strconv.FormatInt(channelID, 10)+"/read", map[string]int64{
		"lastMessageId": lastMessageID,
	}, nil)
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+"/api"+path, payload)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusUnauthorized {
		return errors.New("登录已失效")
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", res.StatusCode)
	}

	var envelope struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		return err
	}
	if envelope.Code != 0 {
		if envelope.Msg != "" {
			return errors.New(envelope.Msg)
		}
		return fmt.Errorf("请求失败: %d", envelope.Code)
	}
	if out == nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil
	}
	return json.Unmarshal(envelope.Data, out)
}
