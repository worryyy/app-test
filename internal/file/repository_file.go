package file

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func normalizePage(page, size int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	return page, size
}

func (r *Repository) FindFileByMD5(ctx context.Context, md5Value string) (*File, error) {
	if md5Value == "" {
		return nil, nil
	}

	coll, err := r.fileCollection()
	if err != nil {
		return nil, err
	}

	var file File
	if err := coll.FindOne(ctx, bson.M{"md5": md5Value}).Decode(&file); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find file by md5 %s: %w", md5Value, err)
	}
	return &file, nil
}

func (r *Repository) FindFileByUserAndMD5(ctx context.Context, userID, md5Value string) (*File, error) {
	if userID == "" || md5Value == "" {
		return nil, nil
	}

	coll, err := r.fileCollection()
	if err != nil {
		return nil, err
	}

	var file File
	if err := coll.FindOne(ctx, bson.M{"userId": userID, "md5": md5Value}).Decode(&file); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find file by user %s and md5 %s: %w", userID, md5Value, err)
	}
	return &file, nil
}

func (r *Repository) FindFilesPage(ctx context.Context, filter bson.M, page, size int) ([]File, error) {
	page, size = normalizePage(page, size)

	coll, err := r.fileCollection()
	if err != nil {
		return nil, err
	}

	opts := options.Find().
		SetSkip(int64((page - 1) * size)).
		SetLimit(int64(size)).
		SetSort(bson.M{"_id": -1})
	cur, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find files: %w", err)
	}
	defer func() {
		_ = cur.Close(ctx)
	}()

	var files []File
	if err := cur.All(ctx, &files); err != nil {
		return nil, fmt.Errorf("decode files: %w", err)
	}
	if files == nil {
		return []File{}, nil
	}
	return files, nil
}

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
