package user

import (
	"context"
	"time"
)

var nowFunc = time.Now

func (s *Service) maybeGrantProvisionalCert(ctx context.Context, user *User) error {
	expiresAt, ok := provisionalGrantExpiresAt(user, nowFunc())
	if !ok {
		return nil
	}

	if err := s.repo.UpdateProvisionalExpiresAt(ctx, user.ID, expiresAt); err != nil {
		return err
	}
	user.ProvisionalExpiresAt = &expiresAt
	return nil
}

func provisionalGrantExpiresAt(user *User, now time.Time) (time.Time, bool) {
	if user == nil || user.StuIsCheck {
		return time.Time{}, false
	}
	if !isInProvisionalWindow(now) {
		return time.Time{}, false
	}

	expiresAt := provisionalExpiresAt(now)
	if user.ProvisionalExpiresAt != nil && !user.ProvisionalExpiresAt.Before(expiresAt) {
		return time.Time{}, false
	}
	return expiresAt, true
}

func isInProvisionalWindow(t time.Time) bool {
	local := t.In(provisionalLocation())
	month := local.Month()
	return month >= time.May && month < time.October
}

func provisionalExpiresAt(t time.Time) time.Time {
	loc := provisionalLocation()
	local := t.In(loc)
	return time.Date(local.Year(), time.October, 1, 0, 0, 0, 0, loc)
}

func provisionalLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return loc
}
