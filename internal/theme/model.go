package theme

import "go.mongodb.org/mongo-driver/bson/primitive"

type Theme struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name              string             `bson:"name" json:"name"`
	CategoryName      string             `bson:"category_name" json:"categoryName"`
	NeedSearch        bool               `bson:"needSearch" json:"needSearch"`
	NeedSuggest       bool               `bson:"needSuggest" json:"needSuggest"`
	SuggestBasicScore int64              `bson:"suggestBasicScore" json:"suggestBasicScore"`
	SuggestNumber     int                `bson:"suggestNumber" json:"suggestNumber"`
	SuggestSetName    string             `bson:"suggestSetName" json:"suggestSetName"`
	SuggestType       string             `bson:"suggestType" json:"suggestType"`
}
