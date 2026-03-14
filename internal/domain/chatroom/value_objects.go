package chatroom

type ID int64

type RoomType string

const (
	Direct  RoomType = "direct"
	Group   RoomType = "group"
	Channel RoomType = "channel"
	Bot     RoomType = "bot"
)
