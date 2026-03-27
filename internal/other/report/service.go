package report

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/pagination"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

type Service struct {
	mongoDB *mongo.Database
}

func NewService(mongoDB *mongo.Database) *Service {
	return &Service{mongoDB: mongoDB}
}

func (s *Service) CreateReportComment(ctx context.Context, report *ReportComment) (*ReportComment, error) {
	if err := s.ensureCommentExists(ctx, report.CommentID); err != nil {
		return nil, err
	}
	if report.CreatedTime.IsZero() {
		report.CreatedTime = time.Now()
	}
	report.HasHandle = false
	res, err := s.mongoDB.Collection("campus_report_comment").InsertOne(ctx, report)
	if err != nil {
		return nil, fmt.Errorf("create report comment: %w", err)
	}
	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, fmt.Errorf("report comment id invalid")
	}
	report.ID = oid
	return report, nil
}

func (s *Service) ReviewReportComment(ctx context.Context, id string, handlerUserID int64, handlerContent string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid report comment id: %w", err)
	}
	count, err := s.mongoDB.Collection("campus_report_comment").CountDocuments(ctx, bson.M{"_id": oid})
	if err != nil {
		return fmt.Errorf("count report comment: %w", err)
	}
	if count == 0 {
		return result.ErrNotExisted
	}

	now := time.Now()
	res, err := s.mongoDB.Collection("campus_report_comment").UpdateByID(ctx, oid, bson.M{"$set": bson.M{
		"handlerUserId":  strconv.FormatInt(handlerUserID, 10),
		"handlerTime":    now,
		"handlerContent": handlerContent,
		"hasHandle":      true,
	}})
	if err != nil {
		return fmt.Errorf("review report comment: %w", err)
	}
	if res.ModifiedCount == 0 {
		return result.NewBizError(result.CodeFail, "失败")
	}
	return nil
}

func (s *Service) ListReportComments(ctx context.Context, page, size int) (*result.CusPage[ReportComment], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	if size > 100 {
		size = 100
	}
	return pagination.ListMongoPage[ReportComment](
		ctx,
		s.mongoDB.Collection("campus_report_comment"),
		bson.M{},
		bson.D{{Key: "hasHandle", Value: 1}},
		page,
		size,
	)
}

func (s *Service) GetReportComment(ctx context.Context, id string) (*ReportComment, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid report comment id: %w", err)
	}
	var report ReportComment
	if err := s.mongoDB.Collection("campus_report_comment").FindOne(ctx, bson.M{"_id": oid}, options.FindOne()).Decode(&report); err != nil {
		return nil, fmt.Errorf("get report comment: %w", err)
	}
	return &report, nil
}

func (s *Service) ensureCommentExists(ctx context.Context, commentID string) error {
	commentID = strings.TrimSpace(commentID)
	oid, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		return result.NewBizError(result.CodeFail, fmt.Sprintf("%s 评论不存在", commentID))
	}
	count, err := s.mongoDB.Collection("campus_comment").CountDocuments(ctx, bson.M{"_id": oid})
	if err != nil {
		return fmt.Errorf("count comment: %w", err)
	}
	if count == 0 {
		return result.NewBizError(result.CodeFail, fmt.Sprintf("%s 评论不存在", commentID))
	}
	return nil
}
