package report

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ReportComment struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CommentID      string             `bson:"commentId" json:"commentId"`
	ReportContent  string             `bson:"reportContent" json:"reportContent"`
	CreatedTime    time.Time          `bson:"createdTime" json:"createdTime"`
	ReportUserID   string             `bson:"reportUserId" json:"reportUserId"`
	HasHandle      bool               `bson:"hasHandle" json:"hasHandle"`
	HandlerContent string             `bson:"handlerContent" json:"handlerContent"`
	HandlerUserID  string             `bson:"handlerUserId" json:"handlerUserId"`
	HandlerTime    *time.Time         `bson:"handlerTime" json:"handlerTime"`
}
