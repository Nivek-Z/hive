package model

type User struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	Nickname    string `json:"nickname"`
	AvatarColor string `json:"avatarColor"`
	AvatarURL   string `json:"avatarUrl"`
	Bio         string `json:"bio"`
	CreatedAt   string `json:"createdAt"`
}

type LoginResp struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type HiveReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IconColor   string `json:"iconColor"`
}

type Hive struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IconColor   string `json:"iconColor"`
	OwnerID     int64  `json:"ownerId"`
}

type CreateChannelReq struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	ParentID *int64 `json:"parentId"`
	Topic    string `json:"topic"`
}

type UpdateChannelReq struct {
	Name     string `json:"name"`
	Topic    string `json:"topic"`
	Position int    `json:"position"`
}

type Channel struct {
	ID       int64  `json:"id"`
	HiveID   int64  `json:"hiveId"`
	ParentID *int64 `json:"parentId"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Topic    string `json:"topic"`
	Position int    `json:"position"`
}

type UnreadRow struct {
	ChannelID int64 `json:"channelId"`
	Count     int   `json:"count"`
}

type HiveDetail struct {
	ID            int64       `json:"id"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	IconColor     string      `json:"iconColor"`
	OwnerID       int64       `json:"ownerId"`
	MemberCount   int         `json:"memberCount"`
	MyPermissions int64       `json:"myPermissions"`
	Channels      []Channel   `json:"channels"`
	Unreads       []UnreadRow `json:"unreads"`
	Roles         []Role      `json:"roles"`
}

type Member struct {
	UserID       int64   `json:"userId"`
	Username     string  `json:"username"`
	Nickname     string  `json:"nickname"`
	HiveNickname string  `json:"hiveNickname"`
	AvatarColor  string  `json:"avatarColor"`
	AvatarURL    string  `json:"avatarUrl"`
	MutedUntil   string  `json:"mutedUntil"`
	JoinedAt     string  `json:"joinedAt"`
	Owner        bool    `json:"owner"`
	RoleIDs      []int64 `json:"roleIds"`
}

type Invite struct {
	Code      string `json:"code"`
	MaxUses   int    `json:"maxUses"`
	UsedCount int    `json:"usedCount"`
	ExpiresAt string `json:"expiresAt"`
}

type Reaction struct {
	Emoji   string  `json:"emoji"`
	Count   int     `json:"count"`
	UserIDs []int64 `json:"userIds"`
}

type Message struct {
	ID                int64      `json:"id"`
	ChannelID         int64      `json:"channelId"`
	SenderID          int64      `json:"senderId"`
	SenderNickname    string     `json:"senderNickname"`
	SenderAvatarColor string     `json:"senderAvatarColor"`
	SenderAvatarURL   string     `json:"senderAvatarUrl"`
	Type              string     `json:"type"`
	Content           string     `json:"content"`
	ReplyToID         *int64     `json:"replyToId"`
	ReplySenderName   string     `json:"replySenderNickname"`
	ReplyContent      string     `json:"replyContent"`
	CreatedAt         string     `json:"createdAt"`
	Reactions         []Reaction `json:"reactions"`
}

type Friend struct {
	UserID      int64  `json:"userId"`
	Username    string `json:"username"`
	Nickname    string `json:"nickname"`
	AvatarColor string `json:"avatarColor"`
	AvatarURL   string `json:"avatarUrl"`
	Bio         string `json:"bio"`
}

type FriendRequest struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"userId"`
	Username    string `json:"username"`
	Nickname    string `json:"nickname"`
	AvatarColor string `json:"avatarColor"`
	AvatarURL   string `json:"avatarUrl"`
	CreatedAt   string `json:"createdAt"`
}

type DM struct {
	ChannelID   int64  `json:"channelId"`
	UserID      int64  `json:"userId"`
	Username    string `json:"username"`
	Nickname    string `json:"nickname"`
	AvatarColor string `json:"avatarColor"`
	AvatarURL   string `json:"avatarUrl"`
	LastContent string `json:"lastContent"`
	LastAt      string `json:"lastAt"`
	Unread      int    `json:"unread"`
}

type OpenDMResp struct {
	ChannelID int64 `json:"channelId"`
}

type RoleReq struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Permissions int64  `json:"permissions"`
}

type Role struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Permissions int64  `json:"permissions"`
	Position    int    `json:"position"`
	IsDefault   bool   `json:"isDefault"`
}

type File struct {
	URL          string `json:"url"`
	OriginalName string `json:"originalName"`
	Size         int64  `json:"size"`
}

type Achievement struct {
	ID          int64  `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Emoji       string `json:"emoji"`
	Secret      bool   `json:"secret"`
	Points      int    `json:"points"`
	UnlockedAt  string `json:"unlockedAt"`
}

type HeatRow struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type NameCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type SearchHit struct {
	ID             int64  `json:"id"`
	ChannelID      int64  `json:"channelId"`
	ChannelName    string `json:"channelName"`
	SenderNickname string `json:"senderNickname"`
	Content        string `json:"content"`
	CreatedAt      string `json:"createdAt"`
}

type HiveStats struct {
	Daily       []HeatRow   `json:"daily"`
	TopSpeakers []NameCount `json:"topSpeakers"`
}
