package notificationadapter

import (
	"context"
	"strconv"

	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"github.com/Milchstrassse/Ecampus-go/internal/notification"
)

type Adapter struct{ Service *notification.Service }

func (a Adapter) PersistLegacyNotification(ctx context.Context, msg mq.NotifyMsg) error {
	return a.Service.CreateLegacy(ctx, notification.LegacyInput{
		TargetUserID: msg.TargetUserID, SenderUserID: msg.SenderUserID, Type: msg.Type,
		Content: msg.Content, TopicID: msg.TopicID, CommentID: msg.CommentID,
		CreatedTime: msg.CreatedTime, EventID: msg.EventID,
	})
}

func (a Adapter) NotifyModeration(ctx context.Context, rootUserID int64, eventType, title, content, resourceType, resourceID string) error {
	return a.create(ctx, rootUserID, notification.CategoryModeration, eventType, title, content, resourceType, resourceID)
}

func (a Adapter) NotifyReservation(ctx context.Context, rootUserID int64, eventType, title, content, resourceID string) error {
	return a.create(ctx, rootUserID, notification.CategoryReservation, eventType, title, content, "reservation", resourceID)
}

func (a Adapter) NotifyMarketplace(ctx context.Context, rootUserID int64, eventType, title, content, resourceID string) error {
	return a.create(ctx, rootUserID, notification.CategoryMarketplace, eventType, title, content, "marketplaceOrder", resourceID)
}

func (a Adapter) create(ctx context.Context, rootUserID int64, category, eventType, title, content, resourceType, resourceID string) error {
	_, _, err := a.Service.Create(ctx, notification.CreateInput{
		ReceiverRootUserID: rootUserID, ReceiverID: strconv.FormatInt(rootUserID, 10),
		Category: category, EventType: eventType, Title: title, Content: content,
		ResourceType: resourceType, ResourceID: resourceID,
	})
	return err
}
