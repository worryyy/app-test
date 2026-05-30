package comment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"
)

func (r *Repository) CreateComment(ctx context.Context, comment *Comment) (primitive.ObjectID, error) {
	coll, err := r.mongoCollection(mongoCollComment)
	if err != nil {
		return primitive.NilObjectID, err
	}

	res, err := coll.InsertOne(ctx, comment)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("insert comment: %w", err)
	}

	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, fmt.Errorf("comment inserted id type invalid")
	}
	comment.ID = oid
	return oid, nil
}

func (r *Repository) HideCommentByUser(
	ctx context.Context,
	topicID string,
	commentID primitive.ObjectID,
	userID string,
) (bool, bool, error) {
	return r.hideComment(ctx, bson.M{
		"_id":         commentID,
		"topicId":     topicID,
		"user.userId": userID,
	})
}

func (r *Repository) HideCommentAdmin(
	ctx context.Context,
	topicID string,
	commentID primitive.ObjectID,
) (bool, bool, error) {
	return r.hideComment(ctx, bson.M{
		"_id":     commentID,
		"topicId": topicID,
	})
}

func (r *Repository) hideComment(ctx context.Context, filter bson.M) (bool, bool, error) {
	coll, err := r.mongoCollection(mongoCollComment)
	if err != nil {
		return false, false, err
	}

	res, err := coll.UpdateOne(ctx, filter, bson.M{
		"$set": bson.M{
			"hasCheck":    false,
			"deletedTime": time.Now(),
		},
	})
	if err != nil {
		commentID, _ := filter["_id"].(primitive.ObjectID)
		return false, false, fmt.Errorf("hide comment %s: %w", commentID.Hex(), err)
	}
	return res.MatchedCount > 0, res.ModifiedCount > 0, nil
}

func (r *Repository) IncrementTopicCommentNum(ctx context.Context, topicID primitive.ObjectID, delta int64) (bool, error) {
	coll, err := r.mongoCollection(mongoCollTopic)
	if err != nil {
		return false, err
	}

	res, err := coll.UpdateByID(ctx, topicID, bson.M{"$inc": bson.M{"commentNum": delta}})
	if err != nil {
		return false, fmt.Errorf("update topic %s commentNum: %w", topicID.Hex(), err)
	}
	return res.MatchedCount > 0, nil
}

func (r *Repository) IncrementRootCommentNum(ctx context.Context, rootCommentID primitive.ObjectID, delta int64) (bool, error) {
	coll, err := r.mongoCollection(mongoCollComment)
	if err != nil {
		return false, err
	}

	res, err := coll.UpdateByID(ctx, rootCommentID, bson.M{"$inc": bson.M{"commentNum": delta}})
	if err != nil {
		return false, fmt.Errorf("update root comment %s commentNum: %w", rootCommentID.Hex(), err)
	}
	return res.MatchedCount > 0, nil
}

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

func (r *Repository) FindUserByID(ctx context.Context, userID int64) (*userRecord, error) {
	if userID <= 0 {
		return nil, nil
	}

	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var user userRecord
	if err := db.Table(user.TableName()).Where("id = ?", userID).Take(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user %d: %w", userID, err)
	}
	return &user, nil
}

func (r *Repository) FindUsersByIDs(ctx context.Context, userIDs []int64) (map[int64]userRecord, error) {
	if len(userIDs) == 0 {
		return map[int64]userRecord{}, nil
	}

	db, err := r.gormDB(ctx)
	if err != nil {
		return nil, err
	}

	var users []userRecord
	var user userRecord
	if err := db.Table(user.TableName()).Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("find users by ids: %w", err)
	}

	result := make(map[int64]userRecord, len(users))
	for _, user := range users {
		result[user.ID] = user
	}
	return result, nil
}

func (r *Repository) FindTopicByID(
	ctx context.Context,
	topicID primitive.ObjectID,
	onlyChecked bool,
) (*CommentTopic, error) {
	coll, err := r.mongoCollection(mongoCollTopic)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"_id": topicID}
	if onlyChecked {
		filter["hasCheck"] = true
	}

	var topic CommentTopic
	if err := coll.FindOne(ctx, filter).Decode(&topic); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find topic %s: %w", topicID.Hex(), err)
	}
	return &topic, nil
}

func (r *Repository) FindCommentByID(ctx context.Context, commentID primitive.ObjectID) (*Comment, error) {
	coll, err := r.mongoCollection(mongoCollComment)
	if err != nil {
		return nil, err
	}

	var cmt Comment
	if err := coll.FindOne(ctx, bson.M{"_id": commentID}).Decode(&cmt); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find comment %s: %w", commentID.Hex(), err)
	}
	return &cmt, nil
}
