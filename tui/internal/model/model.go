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

type Hive struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IconColor   string `json:"iconColor"`
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
