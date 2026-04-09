package mq

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

func tokenizeText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return trimmed
	}
	return strings.Join(parts, " ")
}

func pickFiltered(origin, filtered string) string {
	if strings.TrimSpace(filtered) == "" {
		return origin
	}
	return filtered
}

func isRisky(suggest string) bool {
	v := strings.ToLower(strings.TrimSpace(suggest))
	return v == "risky" || v == "block"
}

type wxUserRecord struct {
	ID          int64  `gorm:"column:id"`
	RootUserID  int64  `gorm:"column:root_user_id"`
	AccountType string `gorm:"column:account_type"`
	OpenID      string `gorm:"column:open_id"`
}

func (wxUserRecord) TableName() string {
	return "campus_user"
}

func (c *Consumers) resolveWXOpenID(ctx context.Context, userID string) (string, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", fmt.Errorf("wx user id is empty")
	}
	if c.db == nil {
		return "", fmt.Errorf("gorm db not initialized")
	}

	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil || id <= 0 {
		return "", fmt.Errorf("invalid wx user id %q", userID)
	}

	user, err := c.findWXUserRecord(ctx, id)
	if err != nil {
		return "", err
	}

	openIDOwnerID, err := resolveWXOpenIDOwnerID(user)
	if err != nil {
		return "", err
	}
	if openIDOwnerID == user.ID {
		return validateWXOpenID(user)
	}

	rootUser, err := c.findWXUserRecord(ctx, openIDOwnerID)
	if err != nil {
		return "", fmt.Errorf("query root wx user %d for user %s: %w", openIDOwnerID, userID, err)
	}
	return validateWXOpenID(rootUser)
}

func (c *Consumers) findWXUserRecord(ctx context.Context, userID int64) (*wxUserRecord, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid wx user id %d", userID)
	}

	var user wxUserRecord
	if err := c.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("wx user %d not found", userID)
		}
		return nil, fmt.Errorf("query wx user %d: %w", userID, err)
	}
	return &user, nil
}

func resolveWXOpenIDOwnerID(user *wxUserRecord) (int64, error) {
	if user == nil || user.ID <= 0 {
		return 0, fmt.Errorf("wx user record is invalid")
	}

	if user.RootUserID > 0 && user.RootUserID != user.ID {
		return user.RootUserID, nil
	}
	if strings.EqualFold(strings.TrimSpace(user.AccountType), "anonymous") {
		return 0, fmt.Errorf("anonymous wx user %d root_user_id is invalid", user.ID)
	}
	return user.ID, nil
}

func validateWXOpenID(user *wxUserRecord) (string, error) {
	if user == nil || user.ID <= 0 {
		return "", fmt.Errorf("wx user record is invalid")
	}

	openID := strings.TrimSpace(user.OpenID)
	if openID == "" {
		return "", fmt.Errorf("wx user %d openid is empty", user.ID)
	}
	return openID, nil
}
