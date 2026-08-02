package notification

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const collectionName = "campus_notifications"

type Repository struct {
	collection *mongo.Collection
}

func NewRepository(db *mongo.Database) *Repository {
	var collection *mongo.Collection
	if db != nil {
		collection = db.Collection(collectionName)
	}
	return &Repository{collection: collection}
}

func (r *Repository) EnsureIndexes(ctx context.Context) error {
	if r.collection == nil {
		return errors.New("notification mongo collection not initialized")
	}
	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "event_id", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
		{Keys: bson.D{{Key: "receiver_root_user_id", Value: 1}, {Key: "category", Value: 1}, {Key: "is_read", Value: 1}, {Key: "created_time", Value: -1}}},
		{Keys: bson.D{{Key: "receiver_id", Value: 1}, {Key: "type", Value: 1}, {Key: "created_time", Value: -1}}},
	}
	if _, err := r.collection.Indexes().CreateMany(ctx, indexes); err != nil {
		return fmt.Errorf("create notification indexes: %w", err)
	}
	return nil
}

func (r *Repository) Insert(ctx context.Context, doc *Document) (bool, error) {
	if r.collection == nil {
		return false, errors.New("notification mongo collection not initialized")
	}
	if _, err := r.collection.InsertOne(ctx, doc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return false, nil
		}
		return false, fmt.Errorf("insert notification: %w", err)
	}
	return true, nil
}

func (r *Repository) List(ctx context.Context, rootUserID int64, identityIDs []int64, category string, page, size int) ([]Document, int64, error) {
	filter := receiverFilter(rootUserID, identityIDs)
	if category != "" {
		filter = andFilter(filter, categoryFilter(category))
	}
	return r.list(ctx, filter, page, size)
}

func (r *Repository) ListLegacy(ctx context.Context, rootUserID int64, identityIDs []int64, typ string, page, size int) ([]Document, int64, error) {
	return r.list(ctx, andFilter(receiverFilter(rootUserID, identityIDs), bson.M{"type": typ}), page, size)
}

func (r *Repository) list(ctx context.Context, filter bson.M, page, size int) ([]Document, int64, error) {
	if r.collection == nil {
		return nil, 0, errors.New("notification mongo collection not initialized")
	}
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}
	cur, err := r.collection.Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "created_time", Value: -1}}).
		SetSkip(int64((page-1)*size)).SetLimit(int64(size)))
	if err != nil {
		return nil, 0, fmt.Errorf("find notifications: %w", err)
	}
	defer func() { _ = cur.Close(ctx) }()
	var docs []Document
	if err := cur.All(ctx, &docs); err != nil {
		return nil, 0, fmt.Errorf("decode notifications: %w", err)
	}
	if docs == nil {
		docs = []Document{}
	}
	return docs, total, nil
}

func (r *Repository) UnreadCategories(ctx context.Context, rootUserID int64, identityIDs []int64) (map[string]int64, error) {
	if r.collection == nil {
		return nil, errors.New("notification mongo collection not initialized")
	}
	filter := andFilter(receiverFilter(rootUserID, identityIDs), bson.M{"is_read": false})
	cur, err := r.collection.Find(ctx, filter, options.Find().SetProjection(bson.M{"category": 1}))
	if err != nil {
		return nil, fmt.Errorf("find unread notifications: %w", err)
	}
	defer func() { _ = cur.Close(ctx) }()
	counts := make(map[string]int64, len(categories))
	for cur.Next(ctx) {
		var row struct {
			Category string `bson:"category"`
		}
		if err := cur.Decode(&row); err != nil {
			return nil, fmt.Errorf("decode unread notification: %w", err)
		}
		if row.Category == "" {
			row.Category = CategorySocial
		}
		counts[row.Category]++
	}
	return counts, cur.Err()
}

func (r *Repository) MarkOneRead(ctx context.Context, rootUserID int64, identityIDs []int64, id string) (bool, error) {
	variants := []any{id}
	if oid, err := primitive.ObjectIDFromHex(id); err == nil {
		variants = append(variants, oid)
	}
	if numeric, err := strconv.ParseInt(id, 10, 64); err == nil {
		variants = append(variants, numeric)
	}
	filter := andFilter(receiverFilter(rootUserID, identityIDs), bson.M{"_id": bson.M{"$in": variants}})
	result, err := r.collection.UpdateOne(ctx, filter, bson.M{"$set": bson.M{"is_read": true}})
	if err != nil {
		return false, fmt.Errorf("mark notification read: %w", err)
	}
	return result.MatchedCount > 0, nil
}

func (r *Repository) MarkRead(ctx context.Context, rootUserID int64, identityIDs []int64, category string) (int64, error) {
	filter := andFilter(receiverFilter(rootUserID, identityIDs), bson.M{"is_read": false})
	if category != "" {
		filter = andFilter(filter, categoryFilter(category))
	}
	result, err := r.collection.UpdateMany(ctx, filter, bson.M{"$set": bson.M{"is_read": true}})
	if err != nil {
		return 0, fmt.Errorf("mark notifications read: %w", err)
	}
	return result.ModifiedCount, nil
}

func (r *Repository) LatestLegacy(ctx context.Context, rootUserID int64, identityIDs []int64, typ string) (*Document, error) {
	filter := andFilter(receiverFilter(rootUserID, identityIDs), bson.M{"type": typ})
	var doc Document
	err := r.collection.FindOne(ctx, filter, options.FindOne().SetSort(bson.D{{Key: "created_time", Value: -1}})).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find latest notification: %w", err)
	}
	return &doc, nil
}

func (r *Repository) MarkLatestLegacyRead(ctx context.Context, rootUserID int64, identityIDs []int64, typ string) error {
	filter := andFilter(receiverFilter(rootUserID, identityIDs), bson.M{"type": typ})
	result := r.collection.FindOneAndUpdate(ctx, filter, bson.M{"$set": bson.M{"is_read": true}}, options.FindOneAndUpdate().SetSort(bson.D{{Key: "created_time", Value: -1}}))
	if err := result.Err(); err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("mark latest notification read: %w", err)
	}
	return nil
}

func receiverFilter(rootUserID int64, identityIDs []int64) bson.M {
	ids := make([]string, 0, len(identityIDs)+1)
	seen := make(map[string]struct{}, len(identityIDs)+1)
	for _, id := range append(identityIDs, rootUserID) {
		if id <= 0 {
			continue
		}
		value := strconv.FormatInt(id, 10)
		if _, ok := seen[value]; !ok {
			seen[value] = struct{}{}
			ids = append(ids, value)
		}
	}
	return bson.M{"$or": bson.A{
		bson.M{"receiver_root_user_id": rootUserID},
		bson.M{"receiver_id": bson.M{"$in": ids}},
	}}
}

func categoryFilter(category string) bson.M {
	if category == CategorySocial {
		return bson.M{"$or": bson.A{bson.M{"category": category}, bson.M{"category": bson.M{"$exists": false}}, bson.M{"category": ""}}}
	}
	return bson.M{"category": category}
}

func andFilter(filters ...bson.M) bson.M {
	items := make(bson.A, 0, len(filters))
	for _, filter := range filters {
		if len(filter) > 0 {
			items = append(items, filter)
		}
	}
	if len(items) == 1 {
		return items[0].(bson.M)
	}
	return bson.M{"$and": items}
}
