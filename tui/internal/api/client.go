package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
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

func (c *Client) Register(ctx context.Context, username, password, nickname string) (model.LoginResp, error) {
	var resp model.LoginResp
	err := c.do(ctx, http.MethodPost, "/auth/register", map[string]string{
		"username": username,
		"password": password,
		"nickname": nickname,
	}, &resp)
	return resp, err
}

func (c *Client) Me(ctx context.Context) (model.User, error) {
	var user model.User
	err := c.do(ctx, http.MethodGet, "/users/me", nil, &user)
	return user, err
}

func (c *Client) UpdateProfile(ctx context.Context, nickname, bio, avatarColor string) (model.User, error) {
	var user model.User
	err := c.do(ctx, http.MethodPut, "/users/me", map[string]string{
		"nickname":    nickname,
		"bio":         bio,
		"avatarColor": avatarColor,
	}, &user)
	return user, err
}

func (c *Client) ChangePassword(ctx context.Context, oldPassword, newPassword string) error {
	return c.do(ctx, http.MethodPut, "/users/me/password", map[string]string{
		"oldPassword": oldPassword,
		"newPassword": newPassword,
	}, nil)
}

func (c *Client) User(ctx context.Context, id int64) (model.User, error) {
	var user model.User
	err := c.do(ctx, http.MethodGet, "/users/"+idString(id), nil, &user)
	return user, err
}

func (c *Client) Hives(ctx context.Context) ([]model.Hive, error) {
	var hives []model.Hive
	err := c.do(ctx, http.MethodGet, "/hives", nil, &hives)
	return hives, err
}

func (c *Client) CreateHive(ctx context.Context, req model.HiveReq) (model.HiveDetail, error) {
	var detail model.HiveDetail
	err := c.do(ctx, http.MethodPost, "/hives", req, &detail)
	return detail, err
}

func (c *Client) HiveDetail(ctx context.Context, id int64) (model.HiveDetail, error) {
	var detail model.HiveDetail
	err := c.do(ctx, http.MethodGet, "/hives/"+idString(id), nil, &detail)
	return detail, err
}

func (c *Client) UpdateHive(ctx context.Context, id int64, req model.HiveReq) (model.Hive, error) {
	var hive model.Hive
	err := c.do(ctx, http.MethodPut, "/hives/"+idString(id), req, &hive)
	return hive, err
}

func (c *Client) DeleteHive(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodDelete, "/hives/"+idString(id), nil, nil)
}

func (c *Client) LeaveHive(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodPost, "/hives/"+idString(id)+"/leave", nil, nil)
}

func (c *Client) Members(ctx context.Context, hiveID int64) ([]model.Member, error) {
	var members []model.Member
	err := c.do(ctx, http.MethodGet, "/hives/"+idString(hiveID)+"/members", nil, &members)
	return members, err
}

func (c *Client) KickMember(ctx context.Context, hiveID, userID int64) error {
	return c.do(ctx, http.MethodDelete, "/hives/"+idString(hiveID)+"/members/"+idString(userID), nil, nil)
}

func (c *Client) MuteMember(ctx context.Context, hiveID, userID int64, minutes int) error {
	return c.do(ctx, http.MethodPost, "/hives/"+idString(hiveID)+"/members/"+idString(userID)+"/mute", map[string]int{
		"minutes": minutes,
	}, nil)
}

func (c *Client) UnmuteMember(ctx context.Context, hiveID, userID int64) error {
	return c.do(ctx, http.MethodDelete, "/hives/"+idString(hiveID)+"/members/"+idString(userID)+"/mute", nil, nil)
}

func (c *Client) CreateInvite(ctx context.Context, hiveID int64, maxUses, expiresHours int) (model.Invite, error) {
	var invite model.Invite
	err := c.do(ctx, http.MethodPost, "/hives/"+idString(hiveID)+"/invites", map[string]int{
		"maxUses":      maxUses,
		"expiresHours": expiresHours,
	}, &invite)
	return invite, err
}

func (c *Client) Invites(ctx context.Context, hiveID int64) ([]model.Invite, error) {
	var invites []model.Invite
	err := c.do(ctx, http.MethodGet, "/hives/"+idString(hiveID)+"/invites", nil, &invites)
	return invites, err
}

func (c *Client) JoinInvite(ctx context.Context, code string) (model.Hive, error) {
	var hive model.Hive
	err := c.do(ctx, http.MethodPost, "/invites/"+url.PathEscape(code)+"/join", nil, &hive)
	return hive, err
}

func (c *Client) CreateChannel(ctx context.Context, hiveID int64, req model.CreateChannelReq) (model.Channel, error) {
	var channel model.Channel
	err := c.do(ctx, http.MethodPost, "/hives/"+idString(hiveID)+"/channels", req, &channel)
	return channel, err
}

func (c *Client) UpdateChannel(ctx context.Context, id int64, req model.UpdateChannelReq) (model.Channel, error) {
	var channel model.Channel
	err := c.do(ctx, http.MethodPut, "/channels/"+idString(id), req, &channel)
	return channel, err
}

func (c *Client) DeleteChannel(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodDelete, "/channels/"+idString(id), nil, nil)
}

func (c *Client) Messages(ctx context.Context, channelID int64, limit int) ([]model.Message, error) {
	return c.MessagesBefore(ctx, channelID, 0, limit)
}

func (c *Client) MessagesBefore(ctx context.Context, channelID, before int64, limit int) ([]model.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	q := url.Values{"limit": {strconv.Itoa(limit)}}
	if before > 0 {
		q.Set("before", idString(before))
	}
	path := "/channels/" + idString(channelID) + "/messages?" + q.Encode()
	var messages []model.Message
	err := c.do(ctx, http.MethodGet, path, nil, &messages)
	return messages, err
}

func (c *Client) MarkRead(ctx context.Context, channelID, lastMessageID int64) error {
	return c.do(ctx, http.MethodPost, "/channels/"+idString(channelID)+"/read", map[string]int64{
		"lastMessageId": lastMessageID,
	}, nil)
}

func (c *Client) DeleteMessage(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodDelete, "/messages/"+idString(id), nil, nil)
}

func (c *Client) AddReaction(ctx context.Context, messageID int64, emoji string) ([]model.Reaction, error) {
	var reactions []model.Reaction
	err := c.do(ctx, http.MethodPost, "/messages/"+idString(messageID)+"/reactions", map[string]string{
		"emoji": emoji,
	}, &reactions)
	return reactions, err
}

func (c *Client) RemoveReaction(ctx context.Context, messageID int64, emoji string) ([]model.Reaction, error) {
	var reactions []model.Reaction
	err := c.do(ctx, http.MethodDelete, "/messages/"+idString(messageID)+"/reactions/"+url.PathEscape(emoji), nil, &reactions)
	return reactions, err
}

func (c *Client) Friends(ctx context.Context) ([]model.Friend, error) {
	var friends []model.Friend
	err := c.do(ctx, http.MethodGet, "/friends", nil, &friends)
	return friends, err
}

func (c *Client) SendFriendRequest(ctx context.Context, username string) error {
	return c.do(ctx, http.MethodPost, "/friends/requests", map[string]string{"username": username}, nil)
}

func (c *Client) FriendRequests(ctx context.Context) ([]model.FriendRequest, error) {
	var requests []model.FriendRequest
	err := c.do(ctx, http.MethodGet, "/friends/requests", nil, &requests)
	return requests, err
}

func (c *Client) AcceptFriendRequest(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodPost, "/friends/requests/"+idString(id)+"/accept", nil, nil)
}

func (c *Client) DeclineFriendRequest(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodDelete, "/friends/requests/"+idString(id), nil, nil)
}

func (c *Client) RemoveFriend(ctx context.Context, userID int64) error {
	return c.do(ctx, http.MethodDelete, "/friends/"+idString(userID), nil, nil)
}

func (c *Client) OpenDM(ctx context.Context, userID int64) (model.OpenDMResp, error) {
	var resp model.OpenDMResp
	err := c.do(ctx, http.MethodPost, "/dms/"+idString(userID), nil, &resp)
	return resp, err
}

func (c *Client) DMs(ctx context.Context) ([]model.DM, error) {
	var dms []model.DM
	err := c.do(ctx, http.MethodGet, "/dms", nil, &dms)
	return dms, err
}

func (c *Client) Roles(ctx context.Context, hiveID int64) ([]model.Role, error) {
	var roles []model.Role
	err := c.do(ctx, http.MethodGet, "/hives/"+idString(hiveID)+"/roles", nil, &roles)
	return roles, err
}

func (c *Client) CreateRole(ctx context.Context, hiveID int64, req model.RoleReq) (model.Role, error) {
	var role model.Role
	err := c.do(ctx, http.MethodPost, "/hives/"+idString(hiveID)+"/roles", req, &role)
	return role, err
}

func (c *Client) UpdateRole(ctx context.Context, id int64, req model.RoleReq) (model.Role, error) {
	var role model.Role
	err := c.do(ctx, http.MethodPut, "/roles/"+idString(id), req, &role)
	return role, err
}

func (c *Client) DeleteRole(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodDelete, "/roles/"+idString(id), nil, nil)
}

func (c *Client) AssignRoles(ctx context.Context, hiveID, userID int64, roleIDs []int64) error {
	return c.do(ctx, http.MethodPut, "/hives/"+idString(hiveID)+"/members/"+idString(userID)+"/roles", map[string][]int64{
		"roleIds": roleIDs,
	}, nil)
}

func (c *Client) UploadFile(ctx context.Context, filePath string) (model.File, error) {
	var uploaded model.File
	file, err := os.Open(filePath)
	if err != nil {
		return uploaded, err
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	name := filepath.Base(filePath)
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(filePath)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, escapeQuotes(name)))
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return uploaded, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return uploaded, err
	}
	if err := writer.Close(); err != nil {
		return uploaded, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/files", &body)
	if err != nil {
		return uploaded, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if err := c.doRequest(req, &uploaded); err != nil {
		return uploaded, err
	}
	return uploaded, nil
}

func (c *Client) Achievements(ctx context.Context) ([]model.Achievement, error) {
	var achievements []model.Achievement
	err := c.do(ctx, http.MethodGet, "/users/me/achievements", nil, &achievements)
	return achievements, err
}

func (c *Client) Heatmap(ctx context.Context) ([]model.HeatRow, error) {
	var rows []model.HeatRow
	err := c.do(ctx, http.MethodGet, "/users/me/heatmap", nil, &rows)
	return rows, err
}

func (c *Client) SearchMessages(ctx context.Context, hiveID int64, q string) ([]model.SearchHit, error) {
	values := url.Values{"hiveId": {idString(hiveID)}, "q": {q}}
	var hits []model.SearchHit
	err := c.do(ctx, http.MethodGet, "/search/messages?"+values.Encode(), nil, &hits)
	return hits, err
}

func (c *Client) HiveStats(ctx context.Context, hiveID int64) (model.HiveStats, error) {
	var stats model.HiveStats
	err := c.do(ctx, http.MethodGet, "/hives/"+idString(hiveID)+"/stats", nil, &stats)
	return stats, err
}

func (c *Client) Konami(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/eggs/konami", nil, nil)
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

	return c.doRequest(req, out)
}

func (c *Client) doRequest(req *http.Request, out any) error {
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

func idString(id int64) string {
	return strconv.FormatInt(id, 10)
}

func escapeQuotes(s string) string {
	return strings.NewReplacer("\\", "\\\\", `"`, "\\\"").Replace(s)
}
