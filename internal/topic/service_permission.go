package topic

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/Milchstrassse/Ecampus-go/internal/pkg/jwtutil"
	"github.com/Milchstrassse/Ecampus-go/internal/pkg/result"
)

const merchantPowerBit = 2

func (s *Service) ensureMerchantPostAllowed(
	ctx context.Context,
	claims *jwtutil.Claims,
	themeID string,
) error {
	if claims == nil {
		return result.ErrParam
	}

	count, err := s.mongoDB.Collection("campus_merchant_theme").
		CountDocuments(ctx, bson.M{"themeId": themeID})
	if err != nil {
		return fmt.Errorf("check merchant theme: %w", err)
	}
	if count == 0 {
		return nil
	}
	if isMerchantPower(claims.Power) {
		return nil
	}
	return result.NewBizError(result.CodeFail, "当前帖子类型只有商家可以发布")
}

func isMerchantPower(power int) bool {
	return ((power >> merchantPowerBit) & 1) == 1
}
