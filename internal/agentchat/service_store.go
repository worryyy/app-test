package agentchat

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

func (s *Service) listConversations(ctx context.Context, rootUserID int64, page, size int) ([]Conversation, int64, error) {
	db, err := s.requireDB()
	if err != nil {
		return nil, 0, err
	}

	page, size = normalizeConversationPage(page, size)
	base := db.WithContext(ctx).Model(&Conversation{}).Where("root_user_id = ?", rootUserID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count agent conversations: %w", err)
	}

	var conversations []Conversation
	if err := base.Order("updated_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&conversations).Error; err != nil {
		return nil, 0, fmt.Errorf("list agent conversations: %w", err)
	}
	return conversations, total, nil
}

func (s *Service) getConversation(ctx context.Context, sessionID string) (*Conversation, error) {
	db, err := s.requireDB()
	if err != nil {
		return nil, err
	}

	var conversation Conversation
	err = db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Take(&conversation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find agent conversation %s: %w", sessionID, err)
	}
	return &conversation, nil
}

func (s *Service) saveConversation(ctx context.Context, conversation *Conversation) error {
	db, err := s.requireDB()
	if err != nil {
		return err
	}
	if conversation == nil {
		return fmt.Errorf("conversation is nil")
	}
	if err := db.WithContext(ctx).Save(conversation).Error; err != nil {
		return fmt.Errorf("save agent conversation %s: %w", conversation.SessionID, err)
	}
	return nil
}

func (s *Service) deleteConversationRecord(ctx context.Context, sessionID string, rootUserID int64) (bool, error) {
	db, err := s.requireDB()
	if err != nil {
		return false, err
	}

	result := db.WithContext(ctx).
		Where("session_id = ? AND root_user_id = ?", sessionID, rootUserID).
		Delete(&Conversation{})
	if result.Error != nil {
		return false, fmt.Errorf("delete agent conversation %s: %w", sessionID, result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (s *Service) requireDB() (*gorm.DB, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("agentchat db is not configured")
	}
	return s.db, nil
}

func normalizeConversationPage(page, size int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = defaultConversationPageSize
	}
	if size > maxConversationPageSize {
		size = maxConversationPageSize
	}
	return page, size
}
