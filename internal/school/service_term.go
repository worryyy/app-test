package school

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func (s *Service) AddTerm(ctx context.Context, term *Term) (*Term, error) {
	exists, err := s.mongoDB.Collection("campus_term").CountDocuments(ctx, bson.M{"term": term.Term})
	if err != nil {
		return nil, fmt.Errorf("check duplicated term: %w", err)
	}
	if exists > 0 {
		return nil, result.NewBizError(result.CodeRepeated, fmt.Sprintf("term: %s已存在", term.Term))
	}

	res, err := s.mongoDB.Collection("campus_term").InsertOne(ctx, term)
	if err != nil {
		return nil, fmt.Errorf("add term: %w", err)
	}
	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, fmt.Errorf("term id invalid")
	}
	saved := *term
	saved.ID = oid
	return &saved, nil
}

func (s *Service) DeleteTerm(ctx context.Context, termID string) error {
	term, err := s.termByID(ctx, termID)
	if err != nil {
		return err
	}

	current, err := s.currentTermDoc(ctx)
	if err != nil && err != mongo.ErrNoDocuments {
		return err
	}
	if current != nil && current.Term == term.Term {
		return result.NewBizError(result.CodeParamError, "请先更新当前学期为其他学期后重新删除")
	}

	res, err := s.mongoDB.Collection("campus_term").DeleteOne(ctx, bson.M{"_id": term.ID})
	if err != nil {
		return fmt.Errorf("delete term: %w", err)
	}
	if res.DeletedCount == 0 {
		return result.NewBizError(result.CodeFail, "失败")
	}
	return nil
}

func (s *Service) SetCurrentTerm(ctx context.Context, termID string) (*CurTerm, error) {
	term, err := s.termByID(ctx, termID)
	if err != nil {
		return nil, err
	}

	current, err := s.currentTermDoc(ctx)
	if err == mongo.ErrNoDocuments {
		doc := CurTerm{Term: term.Term}
		res, insertErr := s.mongoDB.Collection("campus_cur_term").InsertOne(ctx, doc)
		if insertErr != nil {
			return nil, fmt.Errorf("insert current term: %w", insertErr)
		}
		oid, ok := res.InsertedID.(primitive.ObjectID)
		if !ok {
			return nil, fmt.Errorf("current term id invalid")
		}
		doc.ID = oid
		return &doc, nil
	}
	if err != nil {
		return nil, err
	}
	if current.Term == term.Term {
		return current, nil
	}

	_, err = s.mongoDB.Collection("campus_cur_term").UpdateOne(
		ctx,
		bson.M{"_id": current.ID},
		bson.M{"$set": bson.M{"term": term.Term}},
		options.Update().SetUpsert(false),
	)
	if err != nil {
		return nil, fmt.Errorf("set current term: %w", err)
	}
	current.Term = term.Term
	return current, nil
}

func (s *Service) termByID(ctx context.Context, termID string) (*Term, error) {
	oid, err := primitive.ObjectIDFromHex(termID)
	if err != nil {
		return nil, fmt.Errorf("invalid term id: %w", err)
	}

	var term Term
	if err := s.mongoDB.Collection("campus_term").FindOne(ctx, bson.M{"_id": oid}).Decode(&term); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, result.ErrNotExisted
		}
		return nil, fmt.Errorf("find term by id: %w", err)
	}
	return &term, nil
}

func (s *Service) currentTermDoc(ctx context.Context) (*CurTerm, error) {
	var cur CurTerm
	if err := s.mongoDB.Collection("campus_cur_term").FindOne(ctx, bson.M{}).Decode(&cur); err != nil {
		return nil, err
	}
	return &cur, nil
}
