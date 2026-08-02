package commentproducer

import (
	"context"

	"github.com/Milchstrassse/Ecampus-go/internal/comment"
	"github.com/Milchstrassse/Ecampus-go/internal/mq"
)

type Adapter struct{ producer *mq.Producer }

func New(producer *mq.Producer) comment.CommentProducer {
	if producer == nil {
		return nil
	}
	return &Adapter{producer: producer}
}

func (a *Adapter) SendAddComment(ctx context.Context, item comment.Comment) error {
	msg := mq.AddCommentMsg{Comment: mq.AddCommentPayload{
		ID: item.ID, TopicID: item.TopicID, Comment: item.Comment, CreatedTime: item.CreatedTime,
		User: messageUser(item.User), ParentCmtID: item.ParentCmtID, RootCmtID: item.RootCmtID,
		IsAuthor: item.IsAuthor, LikeNum: item.LikeNum, CommentNum: item.CommentNum, HasCheck: item.HasCheck,
	}}
	if item.Parent != nil {
		parent := messageUser(*item.Parent)
		msg.Comment.Parent = &parent
	}
	return a.producer.SendAddComment(ctx, msg)
}

func (a *Adapter) SendDeleteComment(ctx context.Context, topicID, commentID string) error {
	return a.producer.SendDeleteComment(ctx, topicID, commentID)
}

func messageUser(item comment.CommentUser) mq.AddCommentUser {
	return mq.AddCommentUser{UserID: item.UserID, Avatar: item.Avatar, NickName: item.NickName, AccountType: item.AccountType, Signature: item.Signature}
}
