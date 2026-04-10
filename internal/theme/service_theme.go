package theme

import (
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
)

func parseThemeObjectID(id string) (primitive.ObjectID, error) {
	id = strings.TrimSpace(id)
	if !primitive.IsValidObjectID(id) {
		return primitive.NilObjectID, bizerr.Param(errMsgInvalidParam)
	}

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.NilObjectID, bizerr.Param(errMsgInvalidParam)
	}
	return oid, nil
}
