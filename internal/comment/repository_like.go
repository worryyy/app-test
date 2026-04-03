package comment

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (r *Repository) FindLikedCommentIDs(
	ctx context.Context,
	commentIDs []string,
	userID string,
) (map[string]struct{}, error) {
	liked := make(map[string]struct{}, len(commentIDs))
	if userID == "" || len(commentIDs) == 0 {
		return liked, nil
	}

	coll, err := r.mongoCollection(mongoCollCommentLike)
	if err != nil {
		return nil, err
	}

	cur, err := coll.Find(
		ctx,
		bson.M{"commentId": bson.M{"$in": commentIDs}, "userIds": userID},
		options.Find().SetProjection(bson.M{"commentId": 1}),
	)
	if err != nil {
		return nil, fmt.Errorf("find liked comments: %w", err)
	}
	defer func() {
		_ = cur.Close(ctx)
	}()

	var rows []struct {
		CommentID string `bson:"commentId"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("decode liked comments: %w", err)
	}
	for _, row := range rows {
		if row.CommentID != "" {
			liked[row.CommentID] = struct{}{}
		}
	}
	return liked, nil
}

func (r *Repository) CommentLikeExists(ctx context.Context, commentID, userID string) (bool, error) {
	coll, err := r.mongoCollection(mongoCollCommentLike)
	if err != nil {
		return false, err
	}

	count, err := coll.CountDocuments(ctx, bson.M{
		"commentId": commentID,
		"userIds":   userID,
	})
	if err != nil {
		return false, fmt.Errorf("count comment like: %w", err)
	}
	return count > 0, nil
}

func (r *Repository) AddCommentLike(ctx context.Context, commentID, userID string) (bool, error) {
	coll, err := r.mongoCollection(mongoCollCommentLike)
	if err != nil {
		return false, err
	}

	res, err := coll.UpdateOne(
		ctx,
		bson.M{"commentId": commentID},
		bson.M{
			"$setOnInsert": bson.M{"commentId": commentID},
			"$addToSet":    bson.M{"userIds": userID},
		},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return false, fmt.Errorf("like comment %s: %w", commentID, err)
	}
	return res.ModifiedCount > 0 || res.UpsertedCount > 0, nil
}

func (r *Repository) RemoveCommentLike(ctx context.Context, commentID, userID string) (bool, error) {
	coll, err := r.mongoCollection(mongoCollCommentLike)
	if err != nil {
		return false, err
	}

	res, err := coll.UpdateOne(
		ctx,
		bson.M{"commentId": commentID},
		bson.M{"$pull": bson.M{"userIds": userID}},
	)
	if err != nil {
		return false, fmt.Errorf("unlike comment %s: %w", commentID, err)
	}
	return res.ModifiedCount > 0, nil
}

func (r *Repository) IncrementCommentLikeNum(ctx context.Context, commentID primitive.ObjectID, delta int64) (bool, error) {
	coll, err := r.mongoCollection(mongoCollComment)
	if err != nil {
		return false, err
	}

	res, err := coll.UpdateByID(ctx, commentID, bson.M{"$inc": bson.M{"likeNum": delta}})
	if err != nil {
		return false, fmt.Errorf("update comment %s likeNum: %w", commentID.Hex(), err)
	}
	return res.MatchedCount > 0, nil
}
