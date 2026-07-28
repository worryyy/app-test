package ecampuscrm

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Milchstrassse/Ecampus-go/internal/academic"
	"github.com/Milchstrassse/Ecampus-go/internal/comment"
	"github.com/Milchstrassse/Ecampus-go/internal/marketplace"
	"github.com/Milchstrassse/Ecampus-go/internal/moderation"
	"github.com/Milchstrassse/Ecampus-go/internal/notification"
	"github.com/Milchstrassse/Ecampus-go/internal/reservation"
	"github.com/Milchstrassse/Ecampus-go/internal/topic"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

type moderationTargetAdapter struct {
	users       *user.Service
	topics      *topic.Service
	comments    *comment.Service
	academic    *academic.Service
	marketplace *marketplace.Service
}

func (a moderationTargetAdapter) ResolveTarget(ctx context.Context, _, _ int64, targetType, targetID string) (*moderation.TargetSnapshot, error) {
	var ownerUserID int64
	var payload any
	switch targetType {
	case "user":
		idValue, err := strconv.ParseInt(targetID, 10, 64)
		if err != nil {
			return nil, err
		}
		current, err := a.users.GetByID(ctx, idValue)
		if err != nil || current == nil {
			return nil, err
		}
		ownerUserID, payload = current.ID, map[string]any{"id": current.ID, "nickname": current.Nickname}
	case "topic":
		current, err := a.topics.ReportTarget(ctx, targetID)
		if err != nil || current == nil {
			return nil, err
		}
		ownerUserID, _ = strconv.ParseInt(current.UserID, 10, 64)
		payload = current
	case "comment":
		current, err := a.comments.ReportTarget(ctx, targetID)
		if err != nil || current == nil {
			return nil, err
		}
		ownerUserID, _ = strconv.ParseInt(current.User.UserID, 10, 64)
		payload = current
	case "courseReview", "material":
		ownerRootUserID, current, err := a.academic.ReportTarget(ctx, targetType, targetID)
		if err != nil || current == nil {
			return nil, err
		}
		return &moderation.TargetSnapshot{OwnerRootUserID: ownerRootUserID, Payload: current}, nil
	case "marketplaceItem":
		ownerRootUserID, current, err := a.marketplace.ReportTarget(ctx, targetID)
		if err != nil || current == nil {
			return nil, err
		}
		return &moderation.TargetSnapshot{OwnerRootUserID: ownerRootUserID, Payload: current}, nil
	default:
		return nil, fmt.Errorf("unsupported report target %s", targetType)
	}
	owner, err := a.users.GetByID(ctx, ownerUserID)
	if err != nil || owner == nil {
		return nil, err
	}
	rootUserID := owner.RootUserID
	if rootUserID <= 0 {
		rootUserID = owner.ID
	}
	return &moderation.TargetSnapshot{OwnerRootUserID: rootUserID, Payload: payload}, nil
}

func (a moderationTargetAdapter) HideTarget(ctx context.Context, targetType, targetID string) error {
	switch targetType {
	case "topic":
		return a.topics.HideForModeration(ctx, targetID)
	case "comment":
		return a.comments.HideForModeration(ctx, targetID)
	case "chatMessage":
		return fmt.Errorf("chat history is immutable")
	case "courseReview", "material":
		return a.academic.HideReportTarget(ctx, targetType, targetID)
	case "marketplaceItem":
		return a.marketplace.HideForModeration(ctx, targetID)
	default:
		return fmt.Errorf("target type %s cannot be hidden", targetType)
	}
}

type moderationNotifierAdapter struct{ notifications *notification.Service }

func (a moderationNotifierAdapter) NotifyModeration(ctx context.Context, rootUserID int64, eventType, title, content, resourceType, resourceID string) error {
	_, _, err := a.notifications.Create(ctx, notification.CreateInput{ReceiverRootUserID: rootUserID, ReceiverID: strconv.FormatInt(rootUserID, 10), Category: notification.CategoryModeration, EventType: eventType, Title: title, Content: content, ResourceType: resourceType, ResourceID: resourceID})
	return err
}

type reservationNotifierAdapter struct{ notifications *notification.Service }

func (a reservationNotifierAdapter) NotifyReservation(ctx context.Context, rootUserID int64, eventType, title, content, resourceID string) error {
	_, _, err := a.notifications.Create(ctx, notification.CreateInput{ReceiverRootUserID: rootUserID, ReceiverID: strconv.FormatInt(rootUserID, 10), Category: notification.CategoryReservation, EventType: eventType, Title: title, Content: content, ResourceType: "reservation", ResourceID: resourceID})
	return err
}

var _ reservation.Notifier = reservationNotifierAdapter{}

type marketplaceNotifierAdapter struct{ notifications *notification.Service }

func (a marketplaceNotifierAdapter) NotifyMarketplace(ctx context.Context, rootUserID int64, eventType, title, content, resourceID string) error {
	_, _, err := a.notifications.Create(ctx, notification.CreateInput{ReceiverRootUserID: rootUserID, ReceiverID: strconv.FormatInt(rootUserID, 10), Category: notification.CategoryMarketplace, EventType: eventType, Title: title, Content: content, ResourceType: "marketplaceOrder", ResourceID: resourceID})
	return err
}

var _ marketplace.Notifier = marketplaceNotifierAdapter{}
