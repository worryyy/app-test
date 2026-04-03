package comment

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (r *Repository) FindCommentsPage(
	ctx context.Context,
	filter bson.M,
	sort bson.D,
	page, size int,
) ([]Comment, int64, error) {
	coll, err := r.mongoCollection(mongoCollComment)
	if err != nil {
		return nil, 0, err
	}

	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count comments: %w", err)
	}

	cur, err := coll.Find(ctx, filter, options.Find().
		SetSort(sort).
		SetSkip(int64((page-1)*size)).
		SetLimit(int64(size)))
	if err != nil {
		return nil, 0, fmt.Errorf("find comments: %w", err)
	}
	defer func() {
		_ = cur.Close(ctx)
	}()

	var comments []Comment
	if err := cur.All(ctx, &comments); err != nil {
		return nil, 0, fmt.Errorf("decode comments: %w", err)
	}
	return comments, total, nil
}
