package chat

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"gorm.io/gorm"
)

func (s *Service) ListConversations(ctx context.Context, userID int64) ([]Conversation, error) {
	conversationIDs, err := s.getConversationIDs(ctx, userIDString(userID))
	if err != nil {
		return nil, err
	}
	if len(conversationIDs) == 0 {
		return []Conversation{}, nil
	}

	var conversations []Conversation
	err = s.db.WithContext(ctx).
		Where("id IN ?", conversationIDs).
		Order("updated_at DESC").
		Find(&conversations).Error
	if err != nil {
		return nil, fmt.Errorf("load conversations: %w", err)
	}
	return conversations, nil
}

func (s *Service) EnterConversation(ctx context.Context, userID int64, conversationID, lastMessageID string) error {
	if strings.TrimSpace(conversationID) == "" {
		return newFail("会话ID不能为空")
	}

	var member ConversationMember
	err := s.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userIDString(userID)).
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return newNotExisted("")
	}
	if err != nil {
		return fmt.Errorf("load conversation member: %w", err)
	}
	if member.UnreadCount == 0 {
		return nil
	}
	if strings.TrimSpace(lastMessageID) == "" {
		return newFail("最后一条消息ID不能为空")
	}

	tx := s.db.WithContext(ctx).
		Model(&ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userIDString(userID)).
		Updates(map[string]interface{}{
			"unread_count":         0,
			"last_read_message_id": lastMessageID,
		})
	if tx.Error != nil {
		return fmt.Errorf("enter conversation: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return newFail("更新未读数失败")
	}
	return nil
}

func (s *Service) GetUnreadCount(ctx context.Context, userID int64, conversationID string) ([]ConversationUnreadCount, error) {
	if strings.TrimSpace(conversationID) == "" {
		return nil, newFail("会话ID不能为空")
	}

	var list []ConversationUnreadCount
	err := s.db.WithContext(ctx).
		Model(&ConversationMember{}).
		Select("unread_count").
		Where("conversation_id = ? AND user_id = ?", conversationID, userIDString(userID)).
		Find(&list).Error
	if err != nil {
		return nil, fmt.Errorf("get unread count: %w", err)
	}
	if list == nil {
		return []ConversationUnreadCount{}, nil
	}
	return list, nil
}

func (s *Service) QueryConversation(ctx context.Context, userID int64, targetUserID string) ([]string, error) {
	if strings.TrimSpace(targetUserID) == "" {
		return nil, newFail("目标用户ID不能为空")
	}

	targetConversationIDs, err := s.getConversationIDs(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	userConversationIDs, err := s.getConversationIDs(ctx, userIDString(userID))
	if err != nil {
		return nil, err
	}

	resultIDs := make([]string, 0)
	for _, conversationID := range targetConversationIDs {
		if slices.Contains(userConversationIDs, conversationID) {
			resultIDs = append(resultIDs, conversationID)
		}
	}
	return resultIDs, nil
}

func (s *Service) GetPeerUserID(ctx context.Context, conversationID string, currentUserID int64) (string, error) {
	if strings.TrimSpace(conversationID) == "" {
		return "", newFail("会话ID不能为空")
	}

	var member ConversationMember
	err := s.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id <> ?", conversationID, userIDString(currentUserID)).
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", newNotExisted("会话中未找到聊天对象")
	}
	if err != nil {
		return "", fmt.Errorf("get peer user id by conversation: %w", err)
	}
	return member.UserID, nil
}

func (s *Service) DeleteConversation(ctx context.Context, userID int64, conversationID string) error {
	if strings.TrimSpace(conversationID) == "" {
		return newFail("会话ID不能为空")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var conversation Conversation
		err := tx.Where("id = ?", conversationID).First(&conversation).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return newNotExisted("会话不存在")
		}
		if err != nil {
			return fmt.Errorf("load conversation: %w", err)
		}

		var member ConversationMember
		err = tx.Where("conversation_id = ? AND user_id = ?", conversationID, userIDString(userID)).First(&member).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return newFail("无权限删除该会话")
		}
		if err != nil {
			return fmt.Errorf("load conversation member for delete: %w", err)
		}

		deleteTx := tx.Where("conversation_id = ? AND user_id = ?", conversationID, userIDString(userID)).Delete(&ConversationMember{})
		if deleteTx.Error != nil {
			return fmt.Errorf("delete conversation member: %w", deleteTx.Error)
		}
		if deleteTx.RowsAffected == 0 {
			return newFail("删除会话成员失败")
		}

		if _, err := s.messageColl().DeleteMany(ctx, bson.M{"conversation_id": conversationID}); err != nil {
			return fmt.Errorf("delete conversation messages: %w", err)
		}

		var remaining int64
		err = tx.Model(&ConversationMember{}).Where("conversation_id = ?", conversationID).Count(&remaining).Error
		if err != nil {
			return fmt.Errorf("count remaining conversation members: %w", err)
		}
		if remaining > 0 {
			return nil
		}

		deleteConversationTx := tx.Where("id = ?", conversationID).Delete(&Conversation{})
		if deleteConversationTx.Error != nil {
			return fmt.Errorf("delete conversation: %w", deleteConversationTx.Error)
		}
		if deleteConversationTx.RowsAffected == 0 {
			return newFail("删除会话记录失败")
		}
		return nil
	})
}

func (s *Service) getConversationIDs(ctx context.Context, userID string) ([]string, error) {
	var conversationIDs []string
	err := s.db.WithContext(ctx).
		Model(&ConversationMember{}).
		Where("user_id = ?", userID).
		Pluck("conversation_id", &conversationIDs).Error
	if err != nil {
		return nil, fmt.Errorf("load conversation ids: %w", err)
	}
	if conversationIDs == nil {
		return []string{}, nil
	}
	return conversationIDs, nil
}
