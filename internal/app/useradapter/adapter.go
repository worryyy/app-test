package useradapter

import (
	"context"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/bizerr"
	"github.com/Milchstrassse/Ecampus-go/internal/user"
)

type Adapter struct{ Service *user.Service }

func (a Adapter) ResolveRootUserID(ctx context.Context, userID int64) (int64, error) {
	current, err := a.Service.GetByID(ctx, userID)
	if err != nil || current == nil {
		return userID, err
	}
	if current.RootUserID > 0 {
		return current.RootUserID, nil
	}
	return current.ID, nil
}

func (a Adapter) ListIdentityIDs(ctx context.Context, rootUserID int64) ([]int64, error) {
	result, err := a.Service.ListIdentities(ctx, rootUserID)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(result.Identities))
	for _, identity := range result.Identities {
		if identity != nil {
			ids = append(ids, identity.UserID)
		}
	}
	return ids, nil
}

func (a Adapter) RootSchool(ctx context.Context, rootUserID int64) (string, error) {
	current, err := a.Service.GetByID(ctx, rootUserID)
	if err != nil || current == nil {
		return "", err
	}
	return current.School, nil
}

func (a Adapter) VerifyMarketplaceSeller(ctx context.Context, rootUserID int64) error {
	current, err := a.Service.GetByID(ctx, rootUserID)
	if err != nil {
		return err
	}
	if current == nil || !current.StuIsCheck {
		return bizerr.Forbidden("发布商品需要主账号完成学生认证")
	}
	return nil
}
