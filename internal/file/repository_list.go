package file

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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
