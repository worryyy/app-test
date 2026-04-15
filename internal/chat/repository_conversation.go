package chat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"gorm.io/gorm"
)

func (r *Repository) FindConversationIDsByUserID(ctx context.Context, userID string) ([]string, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var conversationIDs []string
	if err := db.Model(&ConversationMember{}).
		Where("user_id = ?", userID).
		Pluck("conversation_id", &conversationIDs).Error; err != nil {
		return nil, fmt.Errorf("load conversation ids by user %s: %w", userID, err)
	}
	if conversationIDs == nil {
		return []string{}, nil
	}
	return conversationIDs, nil
}

func (r *Repository) FindConversationsByIDs(ctx context.Context, conversationIDs []string) ([]Conversation, error) {
	if len(conversationIDs) == 0 {
		return []Conversation{}, nil
	}

	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var conversations []Conversation
	if err := db.Where("id IN ?", conversationIDs).
		Order("updated_at DESC").
		Find(&conversations).Error; err != nil {
		return nil, fmt.Errorf("load conversations: %w", err)
	}
	if conversations == nil {
		return []Conversation{}, nil
	}
	return conversations, nil
}

func (r *Repository) FindConversationMember(ctx context.Context, conversationID, userID string) (*ConversationMember, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var member ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("load conversation member %s/%s: %w", conversationID, userID, err)
	}
	return &member, nil
}

func (r *Repository) FindPeerConversationMember(ctx context.Context, conversationID, currentUserID string) (*ConversationMember, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var member ConversationMember
	if err := db.Where("conversation_id = ? AND user_id <> ?", conversationID, currentUserID).
		First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("load peer conversation member %s/%s: %w", conversationID, currentUserID, err)
	}
	return &member, nil
}

func (r *Repository) UpdateConversationMemberReadState(
	ctx context.Context,
	conversationID, userID string,
	lastReadMessageID int64,
) (bool, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return false, err
	}

	tx := db.Model(&ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Updates(map[string]any{
			"unread_count":         0,
			"last_read_message_id": lastReadMessageID,
		})
	if tx.Error != nil {
		return false, fmt.Errorf("update conversation member read state %s/%s: %w", conversationID, userID, tx.Error)
	}
	return tx.RowsAffected > 0, nil
}

func (r *Repository) FindConversationUnreadCounts(
	ctx context.Context,
	conversationID, userID string,
) ([]ConversationUnreadCount, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var list []ConversationUnreadCount
	if err := db.Model(&ConversationMember{}).
		Select("unread_count").
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("load unread count %s/%s: %w", conversationID, userID, err)
	}
	if list == nil {
		return []ConversationUnreadCount{}, nil
	}
	return list, nil
}

func (r *Repository) SumUnreadCount(ctx context.Context, userID string) (int64, error) {
	db, err := r.gormDB(ctx)
	if err != nil {
		return 0, err
	}

	var total int64
	row := db.Model(&ConversationMember{}).
		Select("COALESCE(SUM(unread_count), 0)").
		Where("user_id = ?", userID).
		Row()
	if err := row.Scan(&total); err != nil {
		return 0, fmt.Errorf("sum unread count for user %s: %w", userID, err)
	}
	return total, nil
}

func (r *Repository) CreateConversationWithMembers(
	ctx context.Context,
	conversation *Conversation,
	members []ConversationMember,
) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(conversation).Error; err != nil {
			return fmt.Errorf("create conversation: %w", err)
		}
		if err := tx.Create(&members).Error; err != nil {
			return fmt.Errorf("create conversation members: %w", err)
		}
		return nil
	})
}

func (r *Repository) UpdateConversationAfterMessage(
	ctx context.Context,
	conversationID, senderID, receiverID, content string,
	sentAt time.Time,
	messageID int64,
) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		senderUpdate := tx.Model(&ConversationMember{}).
			Where("conversation_id = ? AND user_id = ?", conversationID, senderID).
			Update("last_read_message_id", messageID)
		if senderUpdate.Error != nil {
			return fmt.Errorf("update sender last read message: %w", senderUpdate.Error)
		}
		if senderUpdate.RowsAffected == 0 {
			return errRepoConversationMemberMiss
		}

		receiverUpdate := tx.Model(&ConversationMember{}).
			Where("conversation_id = ? AND user_id = ?", conversationID, receiverID).
			Update("unread_count", gorm.Expr("unread_count + 1"))
		if receiverUpdate.Error != nil {
			return fmt.Errorf("increase unread count: %w", receiverUpdate.Error)
		}
		if receiverUpdate.RowsAffected == 0 {
			return errRepoConversationMemberMiss
		}

		conversationUpdate := tx.Model(&Conversation{}).
			Where("id = ?", conversationID).
			Updates(map[string]any{
				"last_message_content":   content,
				"last_message_sender_id": senderID,
				"last_message_sent_at":   sentAt,
				"updated_at":             time.Now(),
			})
		if conversationUpdate.Error != nil {
			return fmt.Errorf("update conversation last message: %w", conversationUpdate.Error)
		}
		if conversationUpdate.RowsAffected == 0 {
			return errRepoConversationNotFound
		}
		return nil
	})
}

func (r *Repository) DeleteConversationForUser(ctx context.Context, conversationID, userID string) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		var conversation Conversation
		if err := tx.Where("id = ?", conversationID).First(&conversation).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errRepoConversationNotFound
			}
			return fmt.Errorf("load conversation %s: %w", conversationID, err)
		}

		var member ConversationMember
		if err := tx.Where("conversation_id = ? AND user_id = ?", conversationID, userID).
			First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errRepoConversationMemberMiss
			}
			return fmt.Errorf("load conversation member %s/%s: %w", conversationID, userID, err)
		}

		deleteMemberTx := tx.Where("conversation_id = ? AND user_id = ?", conversationID, userID).
			Delete(&ConversationMember{})
		if deleteMemberTx.Error != nil {
			return fmt.Errorf("delete conversation member %s/%s: %w", conversationID, userID, deleteMemberTx.Error)
		}
		if deleteMemberTx.RowsAffected == 0 {
			return errRepoConversationDeleteFailed
		}

		var remaining int64
		if err := tx.Model(&ConversationMember{}).
			Where("conversation_id = ?", conversationID).
			Count(&remaining).Error; err != nil {
			return fmt.Errorf("count remaining conversation members %s: %w", conversationID, err)
		}
		if remaining > 0 {
			return nil
		}

		messageColl, err := r.mongoCollection(mongoCollMessage)
		if err != nil {
			return err
		}
		if _, err := messageColl.DeleteMany(ctx, bson.M{"conversation_id": conversationID}); err != nil {
			return fmt.Errorf("delete conversation messages %s: %w", conversationID, err)
		}

		deleteConversationTx := tx.Where("id = ?", conversationID).Delete(&Conversation{})
		if deleteConversationTx.Error != nil {
			return fmt.Errorf("delete conversation %s: %w", conversationID, deleteConversationTx.Error)
		}
		if deleteConversationTx.RowsAffected == 0 {
			return errRepoConversationDeleteFailed
		}
		return nil
	})
}

func (r *Repository) DeleteConversationCascade(ctx context.Context, conversationID string) error {
	db, err := r.gormDB(ctx)
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("conversation_id = ?", conversationID).Delete(&ConversationMember{}).Error; err != nil {
			return fmt.Errorf("delete conversation members %s: %w", conversationID, err)
		}
		if err := tx.Where("id = ?", conversationID).Delete(&Conversation{}).Error; err != nil {
			return fmt.Errorf("delete conversation %s: %w", conversationID, err)
		}
		return nil
	})
}
