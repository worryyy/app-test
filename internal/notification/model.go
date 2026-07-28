package notification

import (
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	CategorySocial      = "social"
	CategoryAcademic    = "academic"
	CategoryMarketplace = "marketplace"
	CategoryReservation = "reservation"
	CategoryModeration  = "moderation"
	CategorySystem      = "system"

	BroadcastChannel = "campus:notification:broadcast"
)

var categories = map[string]struct{}{
	CategorySocial: {}, CategoryAcademic: {}, CategoryMarketplace: {},
	CategoryReservation: {}, CategoryModeration: {}, CategorySystem: {},
}

type Document struct {
	ID                 any            `bson:"_id,omitempty"`
	EventID            string         `bson:"event_id,omitempty"`
	ReceiverRootUserID int64          `bson:"receiver_root_user_id,omitempty"`
	ReceiverID         string         `bson:"receiver_id,omitempty"`
	SenderID           string         `bson:"sender_id,omitempty"`
	Category           string         `bson:"category,omitempty"`
	EventType          string         `bson:"event_type,omitempty"`
	Type               string         `bson:"type,omitempty"`
	Title              string         `bson:"title,omitempty"`
	Content            string         `bson:"content,omitempty"`
	ResourceType       string         `bson:"resource_type,omitempty"`
	ResourceID         string         `bson:"resource_id,omitempty"`
	TopicID            string         `bson:"topic_id,omitempty"`
	CommentID          string         `bson:"comment_id,omitempty"`
	Extra              map[string]any `bson:"extra,omitempty"`
	CreatedTime        time.Time      `bson:"created_time"`
	IsRead             bool           `bson:"is_read"`
}

type Response struct {
	ID           string         `json:"id"`
	Category     string         `json:"category"`
	EventType    string         `json:"eventType"`
	Title        string         `json:"title"`
	Content      string         `json:"content"`
	ResourceType string         `json:"resourceType,omitempty"`
	ResourceID   string         `json:"resourceId,omitempty"`
	Extra        map[string]any `json:"extra,omitempty"`
	CreatedTime  string         `json:"createdTime"`
	IsRead       bool           `json:"isRead"`
}

type LegacyResponse struct {
	ID          string `json:"id"`
	ReceiverID  string `json:"receiverId"`
	SenderID    string `json:"senderId"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	TopicID     string `json:"topicId"`
	CommentID   string `json:"commentId"`
	CreatedTime string `json:"createdTime"`
	IsRead      bool   `json:"isRead"`
}

type Broadcast struct {
	RootUserID       int64          `json:"rootUserId"`
	LegacyReceiverID string         `json:"legacyReceiverId,omitempty"`
	Notification     Response       `json:"notification"`
	Legacy           LegacyResponse `json:"legacy"`
}

type CreateInput struct {
	ReceiverRootUserID int64
	ReceiverID         string
	SenderID           string
	Category           string
	EventType          string
	Title              string
	Content            string
	ResourceType       string
	ResourceID         string
	Extra              map[string]any
	EventID            string
	CreatedTime        time.Time
}

type LegacyInput struct {
	TargetUserID string
	SenderUserID string
	Type         string
	Content      string
	TopicID      string
	CommentID    string
	CreatedTime  time.Time
	EventID      int64
}

func (d Document) response() Response {
	category := d.Category
	if category == "" {
		category = CategorySocial
	}
	eventType := d.EventType
	if eventType == "" {
		eventType = d.Type
	}
	resourceType, resourceID := d.ResourceType, d.ResourceID
	if resourceID == "" && d.CommentID != "" {
		resourceType, resourceID = "comment", d.CommentID
	} else if resourceID == "" && d.TopicID != "" {
		resourceType, resourceID = "topic", d.TopicID
	}
	return Response{
		ID: idString(d.ID), Category: category, EventType: eventType, Title: d.Title,
		Content: d.Content, ResourceType: resourceType, ResourceID: resourceID,
		Extra: d.Extra, CreatedTime: formatTime(d.CreatedTime), IsRead: d.IsRead,
	}
}

func (d Document) legacyResponse() LegacyResponse {
	typ := d.Type
	if typ == "" {
		typ = d.EventType
	}
	return LegacyResponse{
		ID: idString(d.ID), ReceiverID: d.ReceiverID, SenderID: d.SenderID,
		Type: typ, Content: d.Content, TopicID: d.TopicID, CommentID: d.CommentID,
		CreatedTime: formatLegacyDate(d.CreatedTime), IsRead: d.IsRead,
	}
}

func idString(id any) string {
	switch value := id.(type) {
	case primitive.ObjectID:
		return value.Hex()
	case string:
		return value
	case int64:
		return strconv.FormatInt(value, 10)
	case int32:
		return strconv.FormatInt(int64(value), 10)
	default:
		if id == nil {
			return ""
		}
		return fmt.Sprint(id)
	}
}

func validCategory(category string) bool {
	_, ok := categories[category]
	return ok
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return value.In(loc).Format(time.RFC3339)
}

func formatLegacyDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02")
}
