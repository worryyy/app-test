package chat

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (r *Repository) FindMessageByID(ctx context.Context, messageID int64) (*Message, error) {
	coll, err := r.mongoCollection(mongoCollMessage)
	if err != nil {
		return nil, err
	}
	var message Message
	err = coll.FindOne(ctx, bson.M{"message_id": messageID}).Decode(&message)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find message: %w", err)
	}
	return &message, nil
}

func (r *Repository) FindMessagesAfter(
	ctx context.Context,
	conversationIDs []string,
	lastMessageID int64,
) ([]Message, error) {
	if len(conversationIDs) == 0 {
		return []Message{}, nil
	}

	coll, err := r.mongoCollection(mongoCollMessage)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"conversation_id": bson.M{"$in": conversationIDs}}
	if lastMessageID > 0 {
		filter["message_id"] = bson.M{"$gt": lastMessageID}
	}

	cur, err := coll.Find(ctx, filter, options.Find().SetSort(bson.M{"message_id": 1}))
	if err != nil {
		return nil, fmt.Errorf("find offline messages: %w", err)
	}
	defer func() {
		_ = cur.Close(ctx)
	}()

	var messages []Message
	if err := cur.All(ctx, &messages); err != nil {
		return nil, fmt.Errorf("decode offline messages: %w", err)
	}
	if messages == nil {
		return []Message{}, nil
	}
	return messages, nil
}

func (r *Repository) FindConversationMessagesBefore(
	ctx context.Context,
	conversationID string,
	oldestMessageID *int64,
	limit int64,
) ([]Message, int64, error) {
	if limit <= 0 {
		limit = 15
	}

	coll, err := r.mongoCollection(mongoCollMessage)
	if err != nil {
		return nil, 0, err
	}

	filter := bson.M{"conversation_id": conversationID}
	if oldestMessageID != nil {
		filter["message_id"] = bson.M{"$lt": *oldestMessageID}
	}
	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count history messages: %w", err)
	}

	cur, err := coll.Find(ctx, filter, options.Find().
		SetSort(bson.M{"message_id": -1}).
		SetLimit(limit))
	if err != nil {
		return nil, 0, fmt.Errorf("find history messages: %w", err)
	}
	defer func() {
		_ = cur.Close(ctx)
	}()

	var messages []Message
	if err := cur.All(ctx, &messages); err != nil {
		return nil, 0, fmt.Errorf("decode history messages: %w", err)
	}
	if messages == nil {
		return []Message{}, total, nil
	}
	reverseMessages(messages)
	return messages, total, nil
}

func (r *Repository) InsertMessage(ctx context.Context, message *Message) error {
	if message == nil {
		return nil
	}

	coll, err := r.mongoCollection(mongoCollMessage)
	if err != nil {
		return err
	}

	res, err := coll.InsertOne(ctx, message)
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		message.ID = oid
	}
	return nil
}

func reverseMessages(messages []Message) {
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
}
