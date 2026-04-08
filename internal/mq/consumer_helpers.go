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
	ID     int64  `gorm:"column:id"`
	OpenID string `gorm:"column:open_id"`
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

	var user wxUserRecord
	if err := c.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("wx user %s not found", userID)
		}
		return "", fmt.Errorf("query wx user %s: %w", userID, err)
	}
	if strings.TrimSpace(user.OpenID) == "" {
		return "", fmt.Errorf("wx user %s openid is empty", userID)
	}
	return user.OpenID, nil
}
