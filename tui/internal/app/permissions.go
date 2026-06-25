package app

import (
	"fmt"
	"strconv"
	"strings"
)

type permissionDef struct {
	Name  string
	Bit   int64
	Label string
}

const (
	permAdministrator   int64 = 1
	permManageHive      int64 = 1 << 1
	permManageChannels  int64 = 1 << 2
	permManageRoles     int64 = 1 << 3
	permKickMembers     int64 = 1 << 4
	permMuteMembers     int64 = 1 << 5
	permDeleteMessages  int64 = 1 << 6
	permCreateInvite    int64 = 1 << 7
	permMentionEveryone int64 = 1 << 8
	permSendMessages    int64 = 1 << 9
	permAttachFiles     int64 = 1 << 10
	permAddReactions    int64 = 1 << 11

	permDefaultMember = permCreateInvite | permSendMessages | permAttachFiles | permAddReactions
	permPresetAdmin   = permDefaultMember | permManageChannels | permKickMembers | permMuteMembers | permDeleteMessages | permMentionEveryone
	permAll           = (1 << 12) - 1
)

var permissionDefs = []permissionDef{
	{Name: "ADMINISTRATOR", Bit: permAdministrator, Label: "管理员，等同全部权限"},
	{Name: "MANAGE_HIVE", Bit: permManageHive, Label: "修改群聊资料"},
	{Name: "MANAGE_CHANNELS", Bit: permManageChannels, Label: "频道增删改"},
	{Name: "MANAGE_ROLES", Bit: permManageRoles, Label: "角色管理与分配"},
	{Name: "KICK_MEMBERS", Bit: permKickMembers, Label: "踢出成员"},
	{Name: "MUTE_MEMBERS", Bit: permMuteMembers, Label: "禁言成员"},
	{Name: "DELETE_MESSAGES", Bit: permDeleteMessages, Label: "删除他人消息"},
	{Name: "CREATE_INVITE", Bit: permCreateInvite, Label: "创建邀请码"},
	{Name: "MENTION_EVERYONE", Bit: permMentionEveryone, Label: "@全体成员"},
	{Name: "SEND_MESSAGES", Bit: permSendMessages, Label: "发送消息"},
	{Name: "ATTACH_FILES", Bit: permAttachFiles, Label: "上传图片"},
	{Name: "ADD_REACTIONS", Bit: permAddReactions, Label: "添加表情回应"},
}

func permissionHelpLines() []string {
	lines := []string{
		"presets: default_member, admin_preset, all, none",
		"usage: /role create name|#color|send_messages,attach_files",
	}
	for _, def := range permissionDefs {
		lines = append(lines, fmt.Sprintf("%-17s %4d  %s", def.Name, def.Bit, def.Label))
	}
	return lines
}

func parsePermissions(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return n & permAll, nil
	}
	var perms int64
	for _, token := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '+' || r == '|' || r == ' '
	}) {
		token = normalizePermissionToken(token)
		if token == "" {
			continue
		}
		switch token {
		case "NONE":
			continue
		case "ALL":
			perms |= permAll
			continue
		case "DEFAULT_MEMBER", "MEMBER":
			perms |= permDefaultMember
			continue
		case "ADMIN_PRESET", "PRESET_ADMIN", "MODERATOR":
			perms |= permPresetAdmin
			continue
		}
		found := false
		for _, def := range permissionDefs {
			if token == def.Name {
				perms |= def.Bit
				found = true
				break
			}
		}
		if !found {
			return 0, fmt.Errorf("unknown permission %q", token)
		}
	}
	return perms & permAll, nil
}

func normalizePermissionToken(token string) string {
	token = strings.TrimSpace(strings.ToUpper(token))
	token = strings.ReplaceAll(token, "-", "_")
	return token
}

func formatPermissions(perms int64) string {
	perms &= permAll
	if perms == 0 {
		return "none"
	}
	names := make([]string, 0, len(permissionDefs))
	for _, def := range permissionDefs {
		if perms&def.Bit == def.Bit {
			names = append(names, def.Name)
		}
	}
	return strings.Join(names, ",")
}
