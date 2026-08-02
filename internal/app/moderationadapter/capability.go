package moderationadapter

import (
	"context"

	"github.com/Milchstrassse/Ecampus-go/internal/app/useradapter"
	"github.com/Milchstrassse/Ecampus-go/internal/moderation"
)

type Capability struct {
	Moderation *moderation.Service
	Users      useradapter.Adapter
}

func (a Capability) CheckCapability(ctx context.Context, userID, rootUserID int64, capability string) error {
	if rootUserID <= 0 {
		resolved, err := a.Users.ResolveRootUserID(ctx, userID)
		if err != nil {
			return err
		}
		rootUserID = resolved
	}
	return a.Moderation.Check(ctx, rootUserID, capability)
}
