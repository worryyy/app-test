package chat

import (
	"context"
	"strconv"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
)

func (s *Service) ReportMessage(ctx context.Context, reporterUserID int64, messageID string) (*Message, error) {
	id, err := strconv.ParseInt(messageID, 10, 64)
	if err != nil || id <= 0 {
		return nil, bizerr.Param("消息ID格式错误")
	}
	message, err := s.repo.FindMessageByID(ctx, id)
	if err != nil || message == nil {
		return message, err
	}
	member, err := s.repo.FindConversationMember(ctx, message.ConversationID, strconv.FormatInt(reporterUserID, 10))
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, ErrConversationAccessDenied
	}
	return message, nil
}
