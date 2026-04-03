package user

import "context"

type EventProducer interface {
	SendTopicUserUpdate(ctx context.Context, msg TopicUserUpdateMsg) error
	SendCommentUserUpdate(ctx context.Context, msg CommentUserUpdateMsg) error
}

type TopicUserUpdateMsg struct {
	UserID      string `json:"userId"`
	NickName    string `json:"nickName"`
	Avatar      string `json:"avatar"`
	Gender      string `json:"gender"`
	Signature   string `json:"signature"`
	AccountType int    `json:"accountType"`
}

type CommentUserUpdateMsg struct {
	UserID      string `json:"userId"`
	NickName    string `json:"nickName"`
	Avatar      string `json:"avatar"`
	Gender      string `json:"gender"`
	Signature   string `json:"signature"`
	AccountType int    `json:"accountType"`
}
