package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"hive-tui/internal/model"
	"hive-tui/internal/wsproto"
)

type commandAPI interface {
	Register(context.Context, string, string, string) (model.LoginResp, error)
	Me(context.Context) (model.User, error)
	UpdateProfile(context.Context, string, string, string) (model.User, error)
	ChangePassword(context.Context, string, string) error
	User(context.Context, int64) (model.User, error)
	CreateHive(context.Context, model.HiveReq) (model.HiveDetail, error)
	UpdateHive(context.Context, int64, model.HiveReq) (model.Hive, error)
	DeleteHive(context.Context, int64) error
	LeaveHive(context.Context, int64) error
	Members(context.Context, int64) ([]model.Member, error)
	KickMember(context.Context, int64, int64) error
	MuteMember(context.Context, int64, int64, int) error
	UnmuteMember(context.Context, int64, int64) error
	CreateInvite(context.Context, int64, int, int) (model.Invite, error)
	Invites(context.Context, int64) ([]model.Invite, error)
	JoinInvite(context.Context, string) (model.Hive, error)
	CreateChannel(context.Context, int64, model.CreateChannelReq) (model.Channel, error)
	UpdateChannel(context.Context, int64, model.UpdateChannelReq) (model.Channel, error)
	DeleteChannel(context.Context, int64) error
	MessagesBefore(context.Context, int64, int64, int) ([]model.Message, error)
	DeleteMessage(context.Context, int64) error
	AddReaction(context.Context, int64, string) ([]model.Reaction, error)
	RemoveReaction(context.Context, int64, string) ([]model.Reaction, error)
	Friends(context.Context) ([]model.Friend, error)
	SendFriendRequest(context.Context, string) error
	FriendRequests(context.Context) ([]model.FriendRequest, error)
	AcceptFriendRequest(context.Context, int64) error
	DeclineFriendRequest(context.Context, int64) error
	RemoveFriend(context.Context, int64) error
	OpenDM(context.Context, int64) (model.OpenDMResp, error)
	DMs(context.Context) ([]model.DM, error)
	Roles(context.Context, int64) ([]model.Role, error)
	CreateRole(context.Context, int64, model.RoleReq) (model.Role, error)
	UpdateRole(context.Context, int64, model.RoleReq) (model.Role, error)
	DeleteRole(context.Context, int64) error
	AssignRoles(context.Context, int64, int64, []int64) error
	UploadFile(context.Context, string) (model.File, error)
	Achievements(context.Context) ([]model.Achievement, error)
	Heatmap(context.Context) ([]model.HeatRow, error)
	SearchMessages(context.Context, int64, string) ([]model.SearchHit, error)
	HiveStats(context.Context, int64) (model.HiveStats, error)
	Konami(context.Context) error
}

func (m Model) registerCmd() tea.Cmd {
	return func() tea.Msg {
		api, err := m.commandAPI()
		if err != nil {
			return commandResultMsg{Err: err}
		}
		nickname := strings.TrimSpace(m.Username)
		if nickname == "" {
			nickname = "Hive user"
		}
		resp, err := api.Register(context.Background(), m.Username, m.Password, nickname)
		if err != nil {
			return commandResultMsg{Err: err}
		}
		m.Deps.API.SetToken(resp.Token)
		return commandResultMsg{Status: "registered " + resp.User.Username}
	}
}

func (m Model) loadFriendsCmd() tea.Cmd {
	return func() tea.Msg {
		api, err := m.commandAPI()
		if err != nil {
			return commandResultMsg{Err: err}
		}
		friends, err := api.Friends(context.Background())
		if err != nil {
			return commandResultMsg{Err: err}
		}
		requests, _ := api.FriendRequests(context.Background())
		lines := []string{mutedStyle.Render(fmt.Sprintf("friends %d | requests %d", len(friends), len(requests)))}
		actions := make([]panelAction, 0, len(requests)+len(friends))
		for _, request := range requests {
			name := displayName(request.Nickname, request.Username)
			actions = append(actions, panelAction{
				Label: fmt.Sprintf("request #%d  %s  @%s", request.ID, name, request.Username),
				Hint:  "接受",
				Kind:  panelActionAcceptRequest,
				ID:    request.ID,
				Name:  name,
			})
		}
		for _, friend := range friends {
			name := displayName(friend.Nickname, friend.Username)
			actions = append(actions, panelAction{
				Label: fmt.Sprintf("#%d  %s  @%s", friend.UserID, name, friend.Username),
				Hint:  "打开私聊",
				Kind:  panelActionOpenDM,
				ID:    friend.UserID,
				Name:  "dm-" + name,
			})
		}
		if len(actions) == 0 {
			lines = append(lines, mutedStyle.Render("No friends yet. Use /friend add <username>."))
		}
		return commandResultMsg{Title: "Friends", Lines: lines, Actions: actions, Status: "friends loaded"}
	}
}

func (m Model) loadMembersCmd() tea.Cmd {
	return func() tea.Msg {
		api, err := m.commandAPI()
		if err != nil {
			return commandResultMsg{Err: err}
		}
		hiveID, err := m.requireHive()
		if err != nil {
			return commandResultMsg{Err: err}
		}
		members, err := api.Members(context.Background(), hiveID)
		if err != nil {
			return commandResultMsg{Err: err}
		}
		roles, err := api.Roles(context.Background(), hiveID)
		if err != nil {
			return commandResultMsg{Err: err}
		}
		lines := []string{mutedStyle.Render(fmt.Sprintf("成员 %d，Enter 分配角色", len(members)))}
		actions := make([]panelAction, 0, len(members))
		for _, member := range members {
			flag := ""
			if member.Owner {
				flag = " owner"
			}
			if member.MutedUntil != "" {
				flag += " muted"
			}
			name := displayName(member.Nickname, member.Username)
			actions = append(actions, panelAction{
				Label: fmt.Sprintf("#%d  %s  @%s%s", member.UserID, name, member.Username, flag),
				Hint:  "分配角色",
				Kind:  panelActionAssignMemberRoles,
				ID:    member.UserID,
				Name:  name,
			})
		}
		return commandResultMsg{Title: "Members", Lines: lines, Actions: actions, Members: members, Roles: roles, Status: "members loaded"}
	}
}

func (m Model) loadDMsCmd() tea.Cmd {
	return func() tea.Msg {
		api, err := m.commandAPI()
		if err != nil {
			return commandResultMsg{Err: err}
		}
		return m.dmsResult(api)
	}
}

func (m Model) loadRolesCmd() tea.Cmd {
	return func() tea.Msg {
		api, err := m.commandAPI()
		if err != nil {
			return commandResultMsg{Err: err}
		}
		return m.rolesResult(api)
	}
}

func (m Model) commandCmd(input string) tea.Cmd {
	return func() tea.Msg {
		api, _ := m.commandAPI()
		cmd, rest := splitCommand(input)
		switch cmd {
		case "", "help", "?":
			return commandResultMsg{Title: "Commands", Lines: commandHelpLines(), Status: "help"}
		case "me":
			return runWithAPI(api, func(api commandAPI) commandResultMsg {
				user, err := api.Me(context.Background())
				if err != nil {
					return commandResultMsg{Err: err}
				}
				return commandResultMsg{Title: "Me", Lines: userLines(user), Status: "profile loaded"}
			})
		case "profile":
			return runWithAPI(api, func(api commandAPI) commandResultMsg {
				parts := splitPipe(rest, 3)
				if len(parts) < 3 {
					return usage("/profile <nickname>|<bio>|<#color>")
				}
				user, err := api.UpdateProfile(context.Background(), parts[0], parts[1], parts[2])
				if err != nil {
					return commandResultMsg{Err: err}
				}
				return commandResultMsg{Title: "Profile", Lines: userLines(user), Status: "profile updated"}
			})
		case "password":
			return runWithAPI(api, func(api commandAPI) commandResultMsg {
				fields := strings.Fields(rest)
				if len(fields) != 2 {
					return usage("/password <old> <new>")
				}
				if err := api.ChangePassword(context.Background(), fields[0], fields[1]); err != nil {
					return commandResultMsg{Err: err}
				}
				return commandResultMsg{Status: "password changed"}
			})
		case "user":
			return runWithAPI(api, func(api commandAPI) commandResultMsg {
				id, err := parseOneID(rest, "/user <id>")
				if err != nil {
					return commandResultMsg{Err: err}
				}
				user, err := api.User(context.Background(), id)
				if err != nil {
					return commandResultMsg{Err: err}
				}
				return commandResultMsg{Title: "User", Lines: userLines(user), Status: "user loaded"}
			})
		case "permissions":
			return commandResultMsg{Title: "Permissions", Lines: permissionHelpLines(), Status: "permissions"}
		case "hive":
			return m.hiveCommand(api, rest)
		case "members":
			if api == nil {
				return commandResultMsg{Err: fmt.Errorf("API client not configured")}
			}
			return m.loadMembersCmd()()
		case "member":
			return m.memberCommand(api, rest)
		case "invite", "invites", "join":
			return m.inviteCommand(api, cmd, rest)
		case "channel":
			return m.channelCommand(api, rest)
		case "history":
			return m.historyCommand(api, rest)
		case "delete":
			return runWithAPI(api, func(api commandAPI) commandResultMsg {
				id, err := parseOneID(rest, "/delete <messageId>")
				if err != nil {
					return commandResultMsg{Err: err}
				}
				if err := api.DeleteMessage(context.Background(), id); err != nil {
					return commandResultMsg{Err: err}
				}
				return commandResultMsg{Status: fmt.Sprintf("deleted message %d", id)}
			})
		case "react", "unreact":
			return m.reactionCommand(api, cmd, rest)
		case "friends":
			if api == nil {
				return commandResultMsg{Err: fmt.Errorf("API client not configured")}
			}
			return m.loadFriendsCmd()()
		case "friend", "requests", "request":
			return m.friendCommand(api, cmd, rest)
		case "dms", "dm":
			return m.dmCommand(api, cmd, rest)
		case "roles", "role":
			return m.roleCommand(api, cmd, rest)
		case "upload":
			return m.uploadCommand(api, rest)
		case "achievements":
			return runWithAPI(api, func(api commandAPI) commandResultMsg {
				items, err := api.Achievements(context.Background())
				if err != nil {
					return commandResultMsg{Err: err}
				}
				lines := make([]string, 0, len(items))
				for _, item := range items {
					lines = append(lines, fmt.Sprintf("%s %s  %dpt  %s", item.Emoji, item.Name, item.Points, item.Description))
				}
				return commandResultMsg{Title: "Achievements", Lines: nonEmpty(lines), Status: "achievements loaded"}
			})
		case "heatmap":
			return runWithAPI(api, func(api commandAPI) commandResultMsg {
				rows, err := api.Heatmap(context.Background())
				if err != nil {
					return commandResultMsg{Err: err}
				}
				lines := make([]string, 0, len(rows))
				for _, row := range rows {
					if row.Count > 0 {
						lines = append(lines, fmt.Sprintf("%s  %d", row.Date, row.Count))
					}
				}
				return commandResultMsg{Title: "Heatmap", Lines: nonEmpty(lines), Status: "heatmap loaded"}
			})
		case "search":
			return runWithAPI(api, func(api commandAPI) commandResultMsg {
				hiveID, err := m.requireHive()
				if err != nil {
					return commandResultMsg{Err: err}
				}
				query := strings.TrimSpace(rest)
				if query == "" {
					return usage("/search <keyword>")
				}
				hits, err := api.SearchMessages(context.Background(), hiveID, query)
				if err != nil {
					return commandResultMsg{Err: err}
				}
				lines := make([]string, 0, len(hits))
				for _, hit := range hits {
					lines = append(lines, fmt.Sprintf("#%s %s  %s", hit.ChannelName, hit.SenderNickname, hit.Content))
				}
				return commandResultMsg{Title: "Search", Lines: nonEmpty(lines), Status: "search loaded"}
			})
		case "stats":
			return m.statsCommand(api)
		case "konami":
			return runWithAPI(api, func(api commandAPI) commandResultMsg {
				if err := api.Konami(context.Background()); err != nil {
					return commandResultMsg{Err: err}
				}
				return commandResultMsg{Status: "konami unlocked"}
			})
		case "typing":
			if m.Deps.WS == nil {
				return commandResultMsg{Err: fmt.Errorf("websocket not connected")}
			}
			if err := m.Deps.WS.Send("TYPING", map[string]int64{"channelId": m.State.CurrentChannelID}); err != nil {
				return commandResultMsg{Err: err}
			}
			return commandResultMsg{Status: "typing sent"}
		case "ping":
			if m.Deps.WS == nil {
				return commandResultMsg{Err: fmt.Errorf("websocket not connected")}
			}
			if err := m.Deps.WS.Send("PING", map[string]any{}); err != nil {
				return commandResultMsg{Err: err}
			}
			return commandResultMsg{Status: "ping sent"}
		default:
			return commandResultMsg{Title: "Commands", Lines: commandHelpLines(), Status: "unknown command: " + cmd}
		}
	}
}

func (m Model) hiveCommand(api commandAPI, rest string) commandResultMsg {
	return runWithAPI(api, func(api commandAPI) commandResultMsg {
		sub, rest := splitCommand(rest)
		switch sub {
		case "create":
			parts := splitPipe(rest, 3)
			if len(parts) < 1 {
				return usage("/hive create <name>|<description>|<#color>")
			}
			req := model.HiveReq{Name: parts[0], Description: partAt(parts, 1), IconColor: defaultColor(partAt(parts, 2))}
			detail, err := api.CreateHive(context.Background(), req)
			if err != nil {
				return commandResultMsg{Err: err}
			}
			return commandResultMsg{Title: "Hive", Lines: []string{fmt.Sprintf("#%d %s", detail.ID, detail.Name)}, Status: "hive created"}
		case "update":
			hiveID, err := m.requireHive()
			if err != nil {
				return commandResultMsg{Err: err}
			}
			parts := splitPipe(rest, 3)
			if len(parts) < 1 {
				return usage("/hive update <name>|<description>|<#color>")
			}
			hive, err := api.UpdateHive(context.Background(), hiveID, model.HiveReq{Name: parts[0], Description: partAt(parts, 1), IconColor: defaultColor(partAt(parts, 2))})
			if err != nil {
				return commandResultMsg{Err: err}
			}
			return commandResultMsg{Title: "Hive", Lines: []string{fmt.Sprintf("#%d %s", hive.ID, hive.Name)}, Status: "hive updated"}
		case "delete":
			hiveID, err := m.requireHive()
			if err != nil {
				return commandResultMsg{Err: err}
			}
			if err := api.DeleteHive(context.Background(), hiveID); err != nil {
				return commandResultMsg{Err: err}
			}
			return commandResultMsg{Status: "hive deleted"}
		case "leave":
			hiveID, err := m.requireHive()
			if err != nil {
				return commandResultMsg{Err: err}
			}
			if err := api.LeaveHive(context.Background(), hiveID); err != nil {
				return commandResultMsg{Err: err}
			}
			return commandResultMsg{Status: "left hive"}
		case "refresh":
			return commandResultMsg{Status: "use left nav or /join to refresh hives"}
		default:
			return usage("/hive create|update|delete|leave|refresh")
		}
	})
}

func (m Model) memberCommand(api commandAPI, rest string) commandResultMsg {
	return runWithAPI(api, func(api commandAPI) commandResultMsg {
		hiveID, err := m.requireHive()
		if err != nil {
			return commandResultMsg{Err: err}
		}
		sub, rest := splitCommand(rest)
		fields := strings.Fields(rest)
		switch sub {
		case "kick":
			if len(fields) != 1 {
				return usage("/member kick <userId>")
			}
			userID, err := parseID(fields[0])
			if err != nil {
				return commandResultMsg{Err: err}
			}
			if err := api.KickMember(context.Background(), hiveID, userID); err != nil {
				return commandResultMsg{Err: err}
			}
			return commandResultMsg{Status: "member kicked"}
		case "mute":
			if len(fields) != 2 {
				return usage("/member mute <userId> <minutes>")
			}
			userID, err := parseID(fields[0])
			if err != nil {
				return commandResultMsg{Err: err}
			}
			minutes, err := strconv.Atoi(fields[1])
			if err != nil {
				return commandResultMsg{Err: err}
			}
			if err := api.MuteMember(context.Background(), hiveID, userID, minutes); err != nil {
				return commandResultMsg{Err: err}
			}
			return commandResultMsg{Status: "member muted"}
		case "unmute":
			if len(fields) != 1 {
				return usage("/member unmute <userId>")
			}
			userID, err := parseID(fields[0])
			if err != nil {
				return commandResultMsg{Err: err}
			}
			if err := api.UnmuteMember(context.Background(), hiveID, userID); err != nil {
				return commandResultMsg{Err: err}
			}
			return commandResultMsg{Status: "member unmuted"}
		case "roles":
			if len(fields) != 2 {
				return usage("/member roles <userId> <roleId,roleId>")
			}
			userID, err := parseID(fields[0])
			if err != nil {
				return commandResultMsg{Err: err}
			}
			roleIDs, err := parseIDList(fields[1])
			if err != nil {
				return commandResultMsg{Err: err}
			}
			if err := api.AssignRoles(context.Background(), hiveID, userID, roleIDs); err != nil {
				return commandResultMsg{Err: err}
			}
			return commandResultMsg{Status: "member roles updated"}
		default:
			return usage("/member kick|mute|unmute|roles")
		}
	})
}

func (m Model) inviteCommand(api commandAPI, cmd, rest string) commandResultMsg {
	return runWithAPI(api, func(api commandAPI) commandResultMsg {
		hiveID, err := m.requireHive()
		if err != nil && cmd != "join" {
			return commandResultMsg{Err: err}
		}
		switch cmd {
		case "invites":
			invites, err := api.Invites(context.Background(), hiveID)
			if err != nil {
				return commandResultMsg{Err: err}
			}
			lines := make([]string, 0, len(invites))
			for _, invite := range invites {
				lines = append(lines, fmt.Sprintf("%s  used %d/%d  expires %s", invite.Code, invite.UsedCount, invite.MaxUses, emptyDash(invite.ExpiresAt)))
			}
			return commandResultMsg{Title: "Invites", Lines: nonEmpty(lines), Status: "invites loaded"}
		case "join":
			code := strings.TrimSpace(rest)
			if code == "" {
				return usage("/join <inviteCode>")
			}
			hive, err := api.JoinInvite(context.Background(), code)
			if err != nil {
				return commandResultMsg{Err: err}
			}
			return commandResultMsg{Title: "Joined", Lines: []string{fmt.Sprintf("#%d %s", hive.ID, hive.Name)}, Status: "joined hive"}
		default:
			sub, rest := splitCommand(rest)
			if sub != "create" {
				return usage("/invite create [maxUses] [expiresHours] or /invites")
			}
			fields := strings.Fields(rest)
			maxUses, expires := 0, 0
			if len(fields) > 0 {
				maxUses, _ = strconv.Atoi(fields[0])
			}
			if len(fields) > 1 {
				expires, _ = strconv.Atoi(fields[1])
			}
			invite, err := api.CreateInvite(context.Background(), hiveID, maxUses, expires)
			if err != nil {
				return commandResultMsg{Err: err}
			}
			return commandResultMsg{Title: "Invite", Lines: []string{invite.Code}, Status: "invite created"}
		}
	})
}

func (m Model) channelCommand(api commandAPI, rest string) commandResultMsg {
	return runWithAPI(api, func(api commandAPI) commandResultMsg {
		sub, rest := splitCommand(rest)
		switch sub {
		case "create":
			hiveID, err := m.requireHive()
			if err != nil {
				return commandResultMsg{Err: err}
			}
			fields := strings.Fields(rest)
			if len(fields) < 2 {
				return usage("/channel create <TEXT|CATEGORY> <name>|<topic>|<parentId>")
			}
			typ := strings.ToUpper(fields[0])
			parts := splitPipe(strings.TrimSpace(strings.TrimPrefix(rest, fields[0])), 3)
			if len(parts) < 1 {
				return usage("/channel create <TEXT|CATEGORY> <name>|<topic>|<parentId>")
			}
			var parent *int64
			if raw := partAt(parts, 2); raw != "" && raw != "0" {
				id, err := parseID(raw)
				if err != nil {
					return commandResultMsg{Err: err}
				}
				parent = &id
			}
			channel, err := api.CreateChannel(context.Background(), hiveID, model.CreateChannelReq{Name: parts[0], Type: typ, Topic: partAt(parts, 1), ParentID: parent})
			if err != nil {
				return commandResultMsg{Err: err}
			}
			return commandResultMsg{Title: "Channel", Lines: []string{fmt.Sprintf("#%d %s", channel.ID, channel.Name)}, Status: "channel created"}
		case "update":
			fields := strings.Fields(rest)
			if len(fields) < 2 {
				return usage("/channel update <id> <name>|<topic>|<position>")
			}
			id, err := parseID(fields[0])
			if err != nil {
				return commandResultMsg{Err: err}
			}
			parts := splitPipe(strings.TrimSpace(strings.TrimPrefix(rest, fields[0])), 3)
			pos, _ := strconv.Atoi(partAt(parts, 2))
			channel, err := api.UpdateChannel(context.Background(), id, model.UpdateChannelReq{Name: partAt(parts, 0), Topic: partAt(parts, 1), Position: pos})
			if err != nil {
				return commandResultMsg{Err: err}
			}
			return commandResultMsg{Title: "Channel", Lines: []string{fmt.Sprintf("#%d %s", channel.ID, channel.Name)}, Status: "channel updated"}
		case "delete":
			id, err := parseOneID(rest, "/channel delete <id>")
			if err != nil {
				return commandResultMsg{Err: err}
			}
			if err := api.DeleteChannel(context.Background(), id); err != nil {
				return commandResultMsg{Err: err}
			}
			return commandResultMsg{Status: "channel deleted"}
		default:
			return usage("/channel create|update|delete")
		}
	})
}

func (m Model) historyCommand(api commandAPI, rest string) commandResultMsg {
	return runWithAPI(api, func(api commandAPI) commandResultMsg {
		channelID := m.State.CurrentChannelID
		if channelID == 0 {
			return commandResultMsg{Err: fmt.Errorf("no channel selected")}
		}
		fields := strings.Fields(rest)
		before := int64(0)
		limit := 50
		var err error
		if len(fields) > 0 {
			before, err = parseID(fields[0])
			if err != nil {
				return commandResultMsg{Err: err}
			}
		}
		if len(fields) > 1 {
			limit, _ = strconv.Atoi(fields[1])
		}
		messages, err := api.MessagesBefore(context.Background(), channelID, before, limit)
		if err != nil {
			return commandResultMsg{Err: err}
		}
		return commandResultMsg{SetChannel: true, ChannelID: channelID, ChannelName: m.currentChannelName(), Messages: messages, Status: "history loaded"}
	})
}

func (m Model) reactionCommand(api commandAPI, cmd, rest string) commandResultMsg {
	return runWithAPI(api, func(api commandAPI) commandResultMsg {
		fields := strings.Fields(rest)
		if len(fields) != 2 {
			return usage("/react <messageId> <emoji> or /unreact <messageId> <emoji>")
		}
		id, err := parseID(fields[0])
		if err != nil {
			return commandResultMsg{Err: err}
		}
		if cmd == "react" {
			_, err = api.AddReaction(context.Background(), id, fields[1])
		} else {
			_, err = api.RemoveReaction(context.Background(), id, fields[1])
		}
		if err != nil {
			return commandResultMsg{Err: err}
		}
		return commandResultMsg{Status: cmd + " ok"}
	})
}

func (m Model) friendCommand(api commandAPI, cmd, rest string) commandResultMsg {
	return runWithAPI(api, func(api commandAPI) commandResultMsg {
		switch cmd {
		case "requests":
			requests, err := api.FriendRequests(context.Background())
			if err != nil {
				return commandResultMsg{Err: err}
			}
			lines := make([]string, 0, len(requests))
			for _, request := range requests {
				lines = append(lines, fmt.Sprintf("#%d  %s  @%s", request.ID, displayName(request.Nickname, request.Username), request.Username))
			}
			return commandResultMsg{Title: "Friend Requests", Lines: nonEmpty(lines), Status: "requests loaded"}
		case "request":
			sub, rest := splitCommand(rest)
			fields := strings.Fields(rest)
			if len(fields) != 1 {
				return usage("/request accept|decline <id>")
			}
			id, err := parseID(fields[0])
			if err != nil {
				return commandResultMsg{Err: err}
			}
			if sub == "accept" {
				err = api.AcceptFriendRequest(context.Background(), id)
			} else if sub == "decline" {
				err = api.DeclineFriendRequest(context.Background(), id)
			} else {
				return usage("/request accept|decline <id>")
			}
			if err != nil {
				return commandResultMsg{Err: err}
			}
			return commandResultMsg{Status: "request " + sub}
		default:
			sub, rest := splitCommand(rest)
			switch sub {
			case "add":
				username := strings.TrimSpace(rest)
				if username == "" {
					return usage("/friend add <username>")
				}
				if err := api.SendFriendRequest(context.Background(), username); err != nil {
					return commandResultMsg{Err: err}
				}
				return commandResultMsg{Status: "friend request sent"}
			case "remove":
				id, err := parseOneID(rest, "/friend remove <userId>")
				if err != nil {
					return commandResultMsg{Err: err}
				}
				if err := api.RemoveFriend(context.Background(), id); err != nil {
					return commandResultMsg{Err: err}
				}
				return commandResultMsg{Status: "friend removed"}
			default:
				return usage("/friends | /friend add <username> | /requests | /request accept|decline <id>")
			}
		}
	})
}

func (m Model) dmCommand(api commandAPI, cmd, rest string) commandResultMsg {
	return runWithAPI(api, func(api commandAPI) commandResultMsg {
		if cmd == "dms" {
			return m.dmsResult(api)
		}
		sub, rest := splitCommand(rest)
		if sub != "open" {
			return usage("/dm open <userId> or /dms")
		}
		userID, err := parseOneID(rest, "/dm open <userId>")
		if err != nil {
			return commandResultMsg{Err: err}
		}
		resp, err := api.OpenDM(context.Background(), userID)
		if err != nil {
			return commandResultMsg{Err: err}
		}
		return m.openDMChannelResult(api, resp.ChannelID, "dm")
	})
}

func (m Model) dmsResult(api commandAPI) commandResultMsg {
	dms, err := api.DMs(context.Background())
	if err != nil {
		return commandResultMsg{Err: err}
	}
	lines := []string{mutedStyle.Render(fmt.Sprintf("conversations %d", len(dms)))}
	actions := make([]panelAction, 0, len(dms))
	for _, dm := range dms {
		name := "dm-" + displayName(dm.Nickname, dm.Username)
		actions = append(actions, panelAction{
			Label: fmt.Sprintf("channel #%d  %s  unread %d  %s", dm.ChannelID, displayName(dm.Nickname, dm.Username), dm.Unread, emptyDash(dm.LastContent)),
			Hint:  "打开",
			Kind:  panelActionOpenDMChannel,
			ID:    dm.ChannelID,
			Name:  name,
		})
	}
	if len(actions) == 0 {
		lines = append(lines, mutedStyle.Render("No DMs yet. Open one from Friends."))
	}
	return commandResultMsg{Title: "DMs", Lines: lines, Actions: actions, Status: "dms loaded"}
}

func (m Model) openDMCmd(userID int64, name string) tea.Cmd {
	return func() tea.Msg {
		api, err := m.commandAPI()
		if err != nil {
			return commandResultMsg{Err: err}
		}
		resp, err := api.OpenDM(context.Background(), userID)
		if err != nil {
			return commandResultMsg{Err: err}
		}
		return m.openDMChannelResult(api, resp.ChannelID, name)
	}
}

func (m Model) openDMChannelCmd(channelID int64, name string) tea.Cmd {
	return func() tea.Msg {
		api, err := m.commandAPI()
		if err != nil {
			return commandResultMsg{Err: err}
		}
		return m.openDMChannelResult(api, channelID, name)
	}
}

func (m Model) openDMChannelResult(api commandAPI, channelID int64, name string) commandResultMsg {
	if strings.TrimSpace(name) == "" {
		name = "dm"
	}
	messages, err := api.MessagesBefore(context.Background(), channelID, 0, 50)
	if err != nil {
		return commandResultMsg{Err: err}
	}
	return commandResultMsg{SetChannel: true, ChannelID: channelID, ChannelName: name, Messages: messages, Status: fmt.Sprintf("opened %s", name)}
}

func (m Model) acceptFriendRequestCmd(requestID int64) tea.Cmd {
	return func() tea.Msg {
		api, err := m.commandAPI()
		if err != nil {
			return commandResultMsg{Err: err}
		}
		if err := api.AcceptFriendRequest(context.Background(), requestID); err != nil {
			return commandResultMsg{Err: err}
		}
		result := m.dmsResult(api)
		result.Status = "request accepted"
		return result
	}
}

func (m Model) roleCommand(api commandAPI, cmd, rest string) commandResultMsg {
	return runWithAPI(api, func(api commandAPI) commandResultMsg {
		hiveID, err := m.requireHive()
		if err != nil {
			return commandResultMsg{Err: err}
		}
		if cmd == "roles" {
			return m.rolesResult(api)
		}
		sub, rest := splitCommand(rest)
		switch sub {
		case "create":
			req, err := parseRoleReq(rest, "/role create <name>|<#color>|<permissions>")
			if err != nil {
				return commandResultMsg{Err: err}
			}
			role, err := api.CreateRole(context.Background(), hiveID, req)
			if err != nil {
				return commandResultMsg{Err: err}
			}
			return commandResultMsg{Title: "Role", Lines: []string{fmt.Sprintf("#%d %s", role.ID, role.Name)}, Status: "role created"}
		case "update":
			fields := strings.Fields(rest)
			if len(fields) < 2 {
				return usage("/role update <id> <name>|<#color>|<permissions>")
			}
			id, err := parseID(fields[0])
			if err != nil {
				return commandResultMsg{Err: err}
			}
			req, err := parseRoleReq(strings.TrimSpace(strings.TrimPrefix(rest, fields[0])), "/role update <id> <name>|<#color>|<permissions>")
			if err != nil {
				return commandResultMsg{Err: err}
			}
			role, err := api.UpdateRole(context.Background(), id, req)
			if err != nil {
				return commandResultMsg{Err: err}
			}
			return commandResultMsg{Title: "Role", Lines: []string{fmt.Sprintf("#%d %s", role.ID, role.Name)}, Status: "role updated"}
		case "delete":
			id, err := parseOneID(rest, "/role delete <id>")
			if err != nil {
				return commandResultMsg{Err: err}
			}
			if err := api.DeleteRole(context.Background(), id); err != nil {
				return commandResultMsg{Err: err}
			}
			return commandResultMsg{Status: "role deleted"}
		default:
			return usage("/roles | /role create|update|delete")
		}
	})
}

func (m Model) rolesResult(api commandAPI) commandResultMsg {
	hiveID, err := m.requireHive()
	if err != nil {
		return commandResultMsg{Err: err}
	}
	roles, err := api.Roles(context.Background(), hiveID)
	if err != nil {
		return commandResultMsg{Err: err}
	}
	lines := []string{mutedStyle.Render("Enter 编辑角色权限，/permissions 查看权限名和预设")}
	actions := make([]panelAction, 0, len(roles))
	for _, role := range roles {
		actions = append(actions, panelAction{
			Label: fmt.Sprintf("#%d  %s  %s  %s", role.ID, role.Name, role.Color, formatPermissions(role.Permissions)),
			Hint:  "编辑权限",
			Kind:  panelActionEditRole,
			ID:    role.ID,
			Name:  role.Name,
		})
	}
	if len(actions) == 0 {
		lines = append(lines, mutedStyle.Render("还没有角色，可用 /role create <name>|<#color>|<permissions> 创建"))
	}
	return commandResultMsg{Title: "Roles", Lines: nonEmpty(lines), Actions: actions, Roles: roles, Status: "roles loaded"}
}

func (m Model) uploadCommand(api commandAPI, rest string) commandResultMsg {
	return runWithAPI(api, func(api commandAPI) commandResultMsg {
		path := strings.TrimSpace(rest)
		if path == "" {
			return usage("/upload <imagePath>")
		}
		file, err := api.UploadFile(context.Background(), path)
		if err != nil {
			return commandResultMsg{Err: err}
		}
		if m.Deps.WS != nil && m.State.CurrentChannelID != 0 {
			err = m.Deps.WS.Send("MSG_SEND", wsproto.SendMessage{
				ChannelID: m.State.CurrentChannelID,
				Content:   file.URL,
				Type:      "IMAGE",
				Nonce:     fmt.Sprintf("img%d", timeNowNonce()),
			})
			if err != nil {
				return commandResultMsg{Err: err}
			}
		}
		return commandResultMsg{Title: "Upload", Lines: []string{file.URL, file.OriginalName}, Status: "file uploaded"}
	})
}

func (m Model) statsCommand(api commandAPI) commandResultMsg {
	return runWithAPI(api, func(api commandAPI) commandResultMsg {
		hiveID, err := m.requireHive()
		if err != nil {
			return commandResultMsg{Err: err}
		}
		stats, err := api.HiveStats(context.Background(), hiveID)
		if err != nil {
			return commandResultMsg{Err: err}
		}
		lines := []string{mutedStyle.Render("daily")}
		for _, row := range stats.Daily {
			lines = append(lines, fmt.Sprintf("%s  %d", row.Date, row.Count))
		}
		lines = append(lines, "", mutedStyle.Render("top speakers"))
		for _, row := range stats.TopSpeakers {
			lines = append(lines, fmt.Sprintf("%s  %d", row.Name, row.Count))
		}
		return commandResultMsg{Title: "Stats", Lines: nonEmpty(lines), Status: "stats loaded"}
	})
}

func (m Model) commandAPI() (commandAPI, error) {
	if m.Deps.API == nil {
		return nil, fmt.Errorf("API client not configured")
	}
	api, ok := m.Deps.API.(commandAPI)
	if !ok {
		return nil, fmt.Errorf("API client does not support management commands")
	}
	return api, nil
}

func (m Model) requireHive() (int64, error) {
	if m.State.CurrentHiveID == 0 {
		return 0, fmt.Errorf("no hive selected")
	}
	return m.State.CurrentHiveID, nil
}

func runWithAPI(api commandAPI, fn func(commandAPI) commandResultMsg) commandResultMsg {
	if api == nil {
		return commandResultMsg{Err: fmt.Errorf("API client not configured")}
	}
	return fn(api)
}

func commandHelpLines() []string {
	return []string{
		"account: /me /profile /password /user",
		"hive: /hive create update delete leave",
		"member: /members /member kick /member mute /member unmute",
		"member: /member roles",
		"invite: /invite create /invites /join",
		"channel: /channel create update delete",
		"message: /history /delete",
		"reaction: /react /unreact",
		"friends: /friends /friend add remove",
		"requests: /requests /request accept decline",
		"dm: /dms /dm open",
		"roles: /permissions /roles",
		"roles: /role create update delete with permission names",
		"files: /upload <imagePath>",
		"extras: /achievements /heatmap /search /stats",
		"ws: /typing /ping /konami",
	}
}

func usage(line string) commandResultMsg {
	return commandResultMsg{Title: "Usage", Lines: []string{line}, Status: "usage"}
}

func splitCommand(input string) (string, string) {
	input = strings.TrimSpace(strings.TrimPrefix(input, "/"))
	if input == "" {
		return "", ""
	}
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return "", ""
	}
	cmd := strings.ToLower(fields[0])
	return cmd, strings.TrimSpace(strings.TrimPrefix(input, fields[0]))
}

func splitPipe(input string, maxParts int) []string {
	raw := strings.Split(input, "|")
	if maxParts > 0 && len(raw) > maxParts {
		head := raw[:maxParts-1]
		head = append(head, strings.Join(raw[maxParts-1:], "|"))
		raw = head
	}
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func partAt(parts []string, i int) string {
	if i < 0 || i >= len(parts) {
		return ""
	}
	return parts[i]
}

func parseOneID(input, usageLine string) (int64, error) {
	fields := strings.Fields(input)
	if len(fields) != 1 {
		return 0, fmt.Errorf("usage: %s", usageLine)
	}
	return parseID(fields[0])
}

func parseID(raw string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
}

func parseIDList(raw string) ([]int64, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	chunks := strings.Split(raw, ",")
	ids := make([]int64, 0, len(chunks))
	for _, chunk := range chunks {
		id, err := parseID(chunk)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func parseRoleReq(raw, usageLine string) (model.RoleReq, error) {
	parts := splitPipe(raw, 3)
	if len(parts) != 3 {
		return model.RoleReq{}, fmt.Errorf("usage: %s", usageLine)
	}
	permissions, err := parsePermissions(parts[2])
	if err != nil {
		return model.RoleReq{}, err
	}
	return model.RoleReq{Name: parts[0], Color: defaultColor(parts[1]), Permissions: permissions}, nil
}

func defaultColor(color string) string {
	color = strings.TrimSpace(color)
	if color == "" {
		return "#ffb300"
	}
	return color
}

func displayName(nickname, username string) string {
	if strings.TrimSpace(nickname) != "" {
		return nickname
	}
	return username
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func nonEmpty(lines []string) []string {
	if len(lines) == 0 {
		return []string{mutedStyle.Render("No data")}
	}
	return lines
}

func userLines(user model.User) []string {
	return []string{
		fmt.Sprintf("#%d  %s  @%s", user.ID, displayName(user.Nickname, user.Username), user.Username),
		"bio: " + emptyDash(user.Bio),
		"avatar: " + emptyDash(user.AvatarColor),
	}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func timeNowNonce() int64 {
	return time.Now().UnixNano()
}
