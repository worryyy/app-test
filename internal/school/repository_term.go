package school

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func (r *Repository) CountTermsByValue(ctx context.Context, term string) (int64, error) {
	coll, err := r.mongoCollection(mongoCollTerm)
	if err != nil {
		return 0, err
	}

	count, err := coll.CountDocuments(ctx, bson.M{"term": term})
	if err != nil {
		return 0, fmt.Errorf("count terms by value %s: %w", term, err)
	}
	return count, nil
}

func (r *Repository) CreateTerm(ctx context.Context, term *Term) error {
	if term == nil {
		return nil
	}

	coll, err := r.mongoCollection(mongoCollTerm)
	if err != nil {
		return err
	}

	res, err := coll.InsertOne(ctx, term)
	if err != nil {
		return fmt.Errorf("insert term: %w", err)
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		term.ID = oid
	}
	return nil
}

func (r *Repository) FindTermByID(ctx context.Context, id primitive.ObjectID) (*Term, error) {
	coll, err := r.mongoCollection(mongoCollTerm)
	if err != nil {
		return nil, err
	}

	var term Term
	if err := coll.FindOne(ctx, bson.M{"_id": id}).Decode(&term); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find term %s: %w", id.Hex(), err)
	}
	return &term, nil
}

func (r *Repository) FindTermByValue(ctx context.Context, termValue string) (*Term, error) {
	coll, err := r.mongoCollection(mongoCollTerm)
	if err != nil {
		return nil, err
	}

	var term Term
	if err := coll.FindOne(ctx, bson.M{"term": termValue}).Decode(&term); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find term by value %s: %w", termValue, err)
	}
	return &term, nil
}

func (r *Repository) DeleteTermByID(ctx context.Context, id primitive.ObjectID) (bool, error) {
	coll, err := r.mongoCollection(mongoCollTerm)
	if err != nil {
		return false, err
	}

	res, err := coll.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return false, fmt.Errorf("delete term %s: %w", id.Hex(), err)
	}
	return res.DeletedCount > 0, nil
}

func (r *Repository) FindCurrentTerm(ctx context.Context) (*CurTerm, error) {
	coll, err := r.mongoCollection(mongoCollCurTerm)
	if err != nil {
		return nil, err
	}

	var cur CurTerm
	if err := coll.FindOne(ctx, bson.M{}).Decode(&cur); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find current term: %w", err)
	}
	return &cur, nil
}

func (r *Repository) CreateCurrentTerm(ctx context.Context, cur *CurTerm) error {
	if cur == nil {
		return nil
	}

	coll, err := r.mongoCollection(mongoCollCurTerm)
	if err != nil {
		return err
	}

	res, err := coll.InsertOne(ctx, cur)
	if err != nil {
		return fmt.Errorf("insert current term: %w", err)
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		cur.ID = oid
	}
	return nil
}

func (r *Repository) UpdateCurrentTerm(ctx context.Context, id primitive.ObjectID, term string) error {
	coll, err := r.mongoCollection(mongoCollCurTerm)
	if err != nil {
		return err
	}

	if _, err := coll.UpdateByID(ctx, id, bson.M{"$set": bson.M{"term": term}}); err != nil {
		return fmt.Errorf("update current term %s: %w", id.Hex(), err)
	}
	return nil
}

func (r *Repository) ListTerms(ctx context.Context) ([]Term, error) {
	coll, err := r.mongoCollection(mongoCollTerm)
	if err != nil {
		return nil, err
	}

	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("find terms: %w", err)
	}
	defer cursor.Close(ctx)

	var terms []Term
	if err := cursor.All(ctx, &terms); err != nil {
		return nil, fmt.Errorf("decode terms: %w", err)
	}
	return terms, nil
}
