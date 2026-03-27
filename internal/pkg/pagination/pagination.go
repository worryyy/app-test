package pagination

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

func PageSize(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "15"))
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	return page, size
}

func ListMongoPage[T any](
	ctx context.Context,
	coll *mongo.Collection,
	filter bson.M,
	sort interface{},
	page, size int,
) (*result.CusPage[T], error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 15
	}
	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("count mongo docs: %w", err)
	}
	opts := options.Find().
		SetSkip(int64((page - 1) * size)).
		SetLimit(int64(size)).
		SetSort(sort)
	cur, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("find mongo docs: %w", err)
	}

	var list []T
	if err := cur.All(ctx, &list); err != nil {
		if closeErr := cur.Close(ctx); closeErr != nil {
			return nil, fmt.Errorf("close mongo cursor after decode failure: %w", closeErr)
		}
		return nil, fmt.Errorf("decode mongo docs: %w", err)
	}
	if closeErr := cur.Close(ctx); closeErr != nil {
		return nil, fmt.Errorf("close mongo cursor: %w", closeErr)
	}
	return result.NewCusPage(list, total, page, size), nil
}
