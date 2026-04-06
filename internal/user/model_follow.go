package user

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Follow struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FollowerID  int64              `bson:"followerId" json:"followerId"`
	FollowingID int64              `bson:"followingId" json:"followingId"`
	FollowAt    time.Time          `bson:"followAt" json:"followAt"`
	CoFollow    bool               `bson:"-" json:"co_follow,omitempty"`
}

type UserStats struct {
	FollowerCount  int64 `json:"followerCount"`
	FollowingCount int64 `json:"followingCount"`
	LikeCount      int64 `json:"likeCount"`
}
