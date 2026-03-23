package other

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) CreateReportComment(ctx context.Context, report *ReportComment) (string, error) {
	if report.CreatedAt.IsZero() {
		report.CreatedAt = time.Now()
	}
	res, err := s.mongoDB.Collection("campus_report_comment").InsertOne(ctx, report)
	if err != nil {
		return "", fmt.Errorf("create report comment: %w", err)
	}
	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("report comment id invalid")
	}
	return oid.Hex(), nil
}

func (s *Service) ReviewReportComment(ctx context.Context, id string, status int) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid report comment id: %w", err)
	}
	_, err = s.mongoDB.Collection("campus_report_comment").UpdateByID(ctx, oid, bson.M{"$set": bson.M{"status": status}})
	if err != nil {
		return fmt.Errorf("review report comment: %w", err)
	}
	return nil
}

func (s *Service) ListReportComments(ctx context.Context, page, size int) (*result.CusPage[ReportComment], error) {
	return listMongoPage[ReportComment](
		ctx,
		s.mongoDB.Collection("campus_report_comment"),
		bson.M{},
		bson.M{"createdAt": -1},
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
