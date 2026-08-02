package moderationtarget

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Milchstrassse/Ecampus-go/internal/academic"
	"github.com/Milchstrassse/Ecampus-go/internal/app/useradapter"
	"github.com/Milchstrassse/Ecampus-go/internal/chat"
	"github.com/Milchstrassse/Ecampus-go/internal/comment"
	"github.com/Milchstrassse/Ecampus-go/internal/marketplace"
	"github.com/Milchstrassse/Ecampus-go/internal/moderation"
	"github.com/Milchstrassse/Ecampus-go/internal/topic"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

type Resolver struct {
	Users       *user.Service
	Topics      *topic.Service
	Comments    *comment.Service
	Chat        *chat.Service
	Academic    *academic.Service
	Marketplace *marketplace.Service
}

func (r Resolver) ResolveTarget(ctx context.Context, _, reporterUserID int64, targetType, targetID string) (*moderation.TargetSnapshot, error) {
	var ownerUserID int64
	var payload any
	switch targetType {
	case "user":
		id, err := strconv.ParseInt(targetID, 10, 64)
		if err != nil {
			return nil, err
		}
		current, err := r.Users.GetByID(ctx, id)
		if err != nil || current == nil {
			return nil, err
		}
		ownerUserID, payload = current.ID, map[string]any{"id": current.ID, "nickname": current.Nickname, "avatar": current.Avatar, "accountType": current.AccountType}
	case "topic":
		current, err := r.Topics.ReportTarget(ctx, targetID)
		if err != nil || current == nil {
			return nil, err
		}
		ownerUserID, _ = strconv.ParseInt(current.UserID, 10, 64)
		payload = current
	case "comment":
		current, err := r.Comments.ReportTarget(ctx, targetID)
		if err != nil || current == nil {
			return nil, err
		}
		ownerUserID, _ = strconv.ParseInt(current.User.UserID, 10, 64)
		payload = current
	case "chatMessage":
		current, err := r.Chat.ReportMessage(ctx, reporterUserID, targetID)
		if err != nil || current == nil {
			return nil, err
		}
		ownerUserID, _ = strconv.ParseInt(current.SenderID, 10, 64)
		payload = current
	case "courseReview", "material":
		rootUserID, current, err := r.Academic.ReportTarget(ctx, targetType, targetID)
		if err != nil || current == nil {
			return nil, err
		}
		return &moderation.TargetSnapshot{OwnerRootUserID: rootUserID, Payload: current}, nil
	case "marketplaceItem":
		rootUserID, current, err := r.Marketplace.ReportTarget(ctx, targetID)
		if err != nil || current == nil {
			return nil, err
		}
		return &moderation.TargetSnapshot{OwnerRootUserID: rootUserID, Payload: current}, nil
	default:
		return nil, fmt.Errorf("unsupported report target %s", targetType)
	}
	rootUserID, err := (useradapter.Adapter{Service: r.Users}).ResolveRootUserID(ctx, ownerUserID)
	if err != nil {
		return nil, err
	}
	return &moderation.TargetSnapshot{OwnerRootUserID: rootUserID, Payload: payload}, nil
}

func (r Resolver) HideTarget(ctx context.Context, targetType, targetID string) error {
	switch targetType {
	case "topic":
		return r.Topics.HideForModeration(ctx, targetID)
	case "comment":
		return r.Comments.HideForModeration(ctx, targetID)
	case "chatMessage":
		return fmt.Errorf("chat history is immutable")
	case "courseReview", "material":
		return r.Academic.HideReportTarget(ctx, targetType, targetID)
	case "marketplaceItem":
		return r.Marketplace.HideForModeration(ctx, targetID)
	default:
		return fmt.Errorf("target type %s cannot be hidden", targetType)
	}
}
