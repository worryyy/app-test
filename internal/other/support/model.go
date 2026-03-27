package support

import "go.mongodb.org/mongo-driver/bson/primitive"

type FrontendSupport struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Key     string             `bson:"key" json:"key"`
	Val     string             `bson:"val" json:"val"`
	KeyDesc string             `bson:"keyDesc,omitempty" json:"keyDesc"`
}
