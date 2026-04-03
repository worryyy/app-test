package file

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *Repository) CreateFile(ctx context.Context, file *File) error {
	if file == nil {
		return nil
	}

	coll, err := r.fileCollection()
	if err != nil {
		return err
	}

	res, err := coll.InsertOne(ctx, file)
	if err != nil {
		return fmt.Errorf("insert file: %w", err)
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		file.ID = oid
	}
	return nil
}

func (r *Repository) IncrementFileRefCount(ctx context.Context, fileID primitive.ObjectID, delta int) error {
	if fileID.IsZero() || delta == 0 {
		return nil
	}

	coll, err := r.fileCollection()
	if err != nil {
		return err
	}

	if _, err := coll.UpdateByID(ctx, fileID, bson.M{"$inc": bson.M{"refCount": delta}}); err != nil {
		return fmt.Errorf("increment file %s refCount by %d: %w", fileID.Hex(), delta, err)
	}
	return nil
}

func (r *Repository) UpdateFileRefCount(ctx context.Context, fileID primitive.ObjectID, refCount int) error {
	if fileID.IsZero() {
		return nil
	}

	coll, err := r.fileCollection()
	if err != nil {
		return err
	}

	if _, err := coll.UpdateByID(ctx, fileID, bson.M{"$set": bson.M{"refCount": refCount}}); err != nil {
		return fmt.Errorf("update file %s refCount: %w", fileID.Hex(), err)
	}
	return nil
}

func (r *Repository) DeleteFile(ctx context.Context, fileID primitive.ObjectID) error {
	if fileID.IsZero() {
		return nil
	}

	coll, err := r.fileCollection()
	if err != nil {
		return err
	}

	if _, err := coll.DeleteOne(ctx, bson.M{"_id": fileID}); err != nil {
		return fmt.Errorf("delete file %s: %w", fileID.Hex(), err)
	}
	return nil
}

func (r *Repository) UpdateFilesPublic(ctx context.Context, fileIDs []primitive.ObjectID, isPublic bool) (int64, error) {
	if len(fileIDs) == 0 {
		return 0, nil
	}

	coll, err := r.fileCollection()
	if err != nil {
		return 0, err
	}

	res, err := coll.UpdateMany(ctx, bson.M{"_id": bson.M{"$in": fileIDs}}, bson.M{"$set": bson.M{"isPublic": isPublic}})
	if err != nil {
		return 0, fmt.Errorf("update file public flag: %w", err)
	}
	return res.ModifiedCount, nil
}
