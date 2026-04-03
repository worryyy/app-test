package theme

import (
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/bizerr"
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

func parseThemeObjectIDs(ids []string) ([]primitive.ObjectID, error) {
	objectIDs := make([]primitive.ObjectID, 0, len(ids))
	seen := make(map[primitive.ObjectID]struct{}, len(ids))
	for _, id := range ids {
		oid, err := parseThemeObjectID(id)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[oid]; ok {
			continue
		}
		seen[oid] = struct{}{}
		objectIDs = append(objectIDs, oid)
	}
	if len(objectIDs) == 0 {
		return nil, bizerr.Param(errMsgInvalidParam)
	}
	return objectIDs, nil
}
