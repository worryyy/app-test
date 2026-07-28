package ecampus

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Milchstrassse/Ecampus-go/internal/academic"
	"github.com/Milchstrassse/Ecampus-go/internal/chat"
	"github.com/Milchstrassse/Ecampus-go/internal/comment"
	"github.com/Milchstrassse/Ecampus-go/internal/marketplace"
	"github.com/Milchstrassse/Ecampus-go/internal/moderation"
	"github.com/Milchstrassse/Ecampus-go/internal/mq"
	"github.com/Milchstrassse/Ecampus-go/internal/notification"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/reservation"
	"github.com/Milchstrassse/Ecampus-go/internal/topic"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

type topicProducerAdapter struct {
	producer *mq.Producer
}

func newTopicProducerAdapter(producer *mq.Producer) topic.EventProducer {
	if producer == nil {
		return nil
	}
	return &topicProducerAdapter{producer: producer}
}

func (a *topicProducerAdapter) SendTopicCheck(ctx context.Context, msg topic.TopicCheckMsg) error {
	return a.producer.SendTopicCheck(ctx, mq.TopicCheckMsg(msg))
}

func (a *topicProducerAdapter) SendDeleteTopic(ctx context.Context, msg topic.TopicDeleteMsg) error {
	return a.producer.SendDeleteTopic(ctx, mq.TopicDeleteMsg(msg))
}

func (a *topicProducerAdapter) SendNotifyUser(ctx context.Context, msg topic.NotifyMsg) error {
	return a.producer.SendNotifyUser(ctx, mq.NotifyMsg{
		TargetUserID: msg.TargetUserID, SenderUserID: msg.SenderUserID, Type: msg.Type,
		Content: msg.Content, TopicID: msg.TopicID, CommentID: msg.CommentID, CreatedTime: msg.CreatedTime,
	})
}

type notificationUserAdapter struct{ service *user.Service }

func (a notificationUserAdapter) ResolveRootUserID(ctx context.Context, userID int64) (int64, error) {
	current, err := a.service.GetByID(ctx, userID)
	if err != nil || current == nil {
		return userID, err
	}
	if current.RootUserID > 0 {
		return current.RootUserID, nil
	}
	return current.ID, nil
}

func (a notificationUserAdapter) ListIdentityIDs(ctx context.Context, rootUserID int64) ([]int64, error) {
	result, err := a.service.ListIdentities(ctx, rootUserID)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(result.Identities))
	for _, identity := range result.Identities {
		if identity != nil {
			ids = append(ids, identity.UserID)
		}
	}
	return ids, nil
}

type notificationWriterAdapter struct{ service *notification.Service }

func (a notificationWriterAdapter) PersistLegacyNotification(ctx context.Context, msg mq.NotifyMsg) error {
	return a.service.CreateLegacy(ctx, notification.LegacyInput{
		TargetUserID: msg.TargetUserID, SenderUserID: msg.SenderUserID, Type: msg.Type,
		Content: msg.Content, TopicID: msg.TopicID, CommentID: msg.CommentID,
		CreatedTime: msg.CreatedTime, EventID: msg.EventID,
	})
}

type moderationCapabilityAdapter struct {
	moderation *moderation.Service
	users      *user.Service
}

func (a moderationCapabilityAdapter) CheckCapability(ctx context.Context, userID, rootUserID int64, capability string) error {
	if rootUserID <= 0 {
		resolved, err := (notificationUserAdapter{service: a.users}).ResolveRootUserID(ctx, userID)
		if err != nil {
			return err
		}
		rootUserID = resolved
	}
	return a.moderation.Check(ctx, rootUserID, capability)
}

type moderationTargetAdapter struct {
	users       *user.Service
	topics      *topic.Service
	comments    *comment.Service
	chat        *chat.Service
	academic    *academic.Service
	marketplace *marketplace.Service
}

func (a moderationTargetAdapter) ResolveTarget(ctx context.Context, _, reporterUserID int64, targetType, targetID string) (*moderation.TargetSnapshot, error) {
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
		ownerUserID = current.ID
		payload = map[string]any{"id": current.ID, "nickname": current.Nickname, "avatar": current.Avatar, "accountType": current.AccountType}
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
	case "chatMessage":
		current, err := a.chat.ReportMessage(ctx, reporterUserID, targetID)
		if err != nil || current == nil {
			return nil, err
		}
		ownerUserID, _ = strconv.ParseInt(current.SenderID, 10, 64)
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
	rootUserID, err := (notificationUserAdapter{service: a.users}).ResolveRootUserID(ctx, ownerUserID)
	if err != nil {
		return nil, err
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

type academicProfileAdapter struct{ users *user.Service }

func (a academicProfileAdapter) RootSchool(ctx context.Context, rootUserID int64) (string, error) {
	current, err := a.users.GetByID(ctx, rootUserID)
	if err != nil || current == nil {
		return "", err
	}
	return current.School, nil
}

type moderationNotifierAdapter struct{ service *notification.Service }

func (a moderationNotifierAdapter) NotifyModeration(ctx context.Context, rootUserID int64, eventType, title, content, resourceType, resourceID string) error {
	_, _, err := a.service.Create(ctx, notification.CreateInput{
		ReceiverRootUserID: rootUserID, ReceiverID: strconv.FormatInt(rootUserID, 10),
		Category: notification.CategoryModeration, EventType: eventType, Title: title,
		Content: content, ResourceType: resourceType, ResourceID: resourceID,
	})
	return err
}

type reservationNotifierAdapter struct{ service *notification.Service }

func (a reservationNotifierAdapter) NotifyReservation(ctx context.Context, rootUserID int64, eventType, title, content, resourceID string) error {
	_, _, err := a.service.Create(ctx, notification.CreateInput{ReceiverRootUserID: rootUserID, ReceiverID: strconv.FormatInt(rootUserID, 10), Category: notification.CategoryReservation, EventType: eventType, Title: title, Content: content, ResourceType: "reservation", ResourceID: resourceID})
	return err
}

var _ reservation.Notifier = reservationNotifierAdapter{}

type marketplaceSellerAdapter struct{ users *user.Service }

func (a marketplaceSellerAdapter) VerifyMarketplaceSeller(ctx context.Context, rootUserID int64) error {
	current, err := a.users.GetByID(ctx, rootUserID)
	if err != nil {
		return err
	}
	if current == nil || !current.StuIsCheck {
		return bizerr.Forbidden("发布商品需要主账号完成学生认证")
	}
	return nil
}

type marketplaceNotifierAdapter struct{ service *notification.Service }

func (a marketplaceNotifierAdapter) NotifyMarketplace(ctx context.Context, rootUserID int64, eventType, title, content, resourceID string) error {
	_, _, err := a.service.Create(ctx, notification.CreateInput{ReceiverRootUserID: rootUserID, ReceiverID: strconv.FormatInt(rootUserID, 10), Category: notification.CategoryMarketplace, EventType: eventType, Title: title, Content: content, ResourceType: "marketplaceOrder", ResourceID: resourceID})
	return err
}

var _ marketplace.SellerVerifier = marketplaceSellerAdapter{}
var _ marketplace.Notifier = marketplaceNotifierAdapter{}

type userProducerAdapter struct {
	producer *mq.Producer
}

func newUserProducerAdapter(producer *mq.Producer) user.EventProducer {
	if producer == nil {
		return nil
	}
	return &userProducerAdapter{producer: producer}
}

func (a *userProducerAdapter) SendTopicUserUpdate(ctx context.Context, msg user.TopicUserUpdateMsg) error {
	return a.producer.SendUpdateTopicUser(ctx, mq.TopicUserUpdateMsg(msg))
}

func (a *userProducerAdapter) SendCommentUserUpdate(ctx context.Context, msg user.CommentUserUpdateMsg) error {
	return a.producer.SendUpdateCommentUser(ctx, mq.CommentUserUpdateMsg(msg))
}
