package chat

import (
	"context"
	"strings"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
)

func (s *Service) ListNotifications(
	ctx context.Context,
	userID int64,
	typ string,
	page, size int,
) (*PageResult[Notification], error) {
	if userID <= 0 {
		return nil, bizerr.Param(errMsgInvalidParam)
	}
	if strings.TrimSpace(typ) == "" {
		return nil, ErrNotificationTypeRequired
	}
	page, size = normalizePage(page, size, s.defaultPageSize())

	list, err := s.repo.FindNotificationsPage(ctx, userIDString(userID), typ, page, size)
	if err != nil {
		return nil, bizerr.InternalWrap("查询通知失败", err)
	}
	if len(list) == 0 {
		return NewPageResult([]Notification{}, 0, page, size), nil
	}

	if err := s.repo.MarkLatestNotificationRead(ctx, userIDString(userID), typ); err != nil {
		return nil, bizerr.InternalWrap("更新通知已读状态失败", err)
	}
	return NewPageResult(list, int64(len(list)), page, size), nil
}

func (s *Service) HaveUnreadNotification(ctx context.Context, userID int64, typ string) (bool, error) {
	notification, err := s.LatestNotification(ctx, userID, typ)
	if err != nil {
		return false, err
	}
	if notification == nil {
		return false, nil
	}
	return !notification.IsRead, nil
}

func (s *Service) LatestNotification(ctx context.Context, userID int64, typ string) (*Notification, error) {
	if userID <= 0 {
		return nil, bizerr.Param(errMsgInvalidParam)
	}
	if strings.TrimSpace(typ) == "" {
		return nil, ErrNotificationTypeRequired
	}

	notification, err := s.repo.FindLatestNotification(ctx, userIDString(userID), typ)
	if err != nil {
		return nil, bizerr.InternalWrap("查询最新通知失败", err)
	}
	return notification, nil
}
