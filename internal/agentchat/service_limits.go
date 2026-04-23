package agentchat

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var counterGuardAcquireScript = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
	redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return count
`)

func (s *Service) acquireTurnGuards(ctx context.Context, rootUserID int64, sessionID string) (func(), error) {
	if err := s.enforcePerMinute(ctx, rootUserID); err != nil {
		return func() {}, err
	}

	userRelease, err := s.acquireCounterGuard(
		ctx,
		activeUserKey(rootUserID),
		s.maxConcurrentPerUser(),
		s.guardTTL(),
		ErrAgentUserBusy,
	)
	if err != nil {
		return func() {}, err
	}

	sessionRelease, err := s.acquireCounterGuard(
		ctx,
		activeConversationKey(sessionID),
		s.maxConcurrentPerConversation(),
		s.guardTTL(),
		ErrAgentConversationBusy,
	)
	if err != nil {
		userRelease()
		return func() {}, err
	}

	return func() {
		sessionRelease()
		userRelease()
	}, nil
}

func (s *Service) enforcePerMinute(ctx context.Context, rootUserID int64) error {
	if s.redis == nil {
		return nil
	}

	key := rateLimitKey(rootUserID, time.Now().UTC())
	count, err := s.redis.Incr(ctx, key).Result()
	if err != nil {
		return nil
	}
	if count == 1 {
		_ = s.redis.Expire(ctx, key, time.Minute).Err()
	}
	if count <= int64(s.rateLimitPerMinute()) {
		return nil
	}

	_, _ = s.redis.Decr(ctx, key).Result()
	return ErrAgentRateLimited
}

func (s *Service) acquireCounterGuard(ctx context.Context, key string, limit int, ttl time.Duration, limitErr error) (func(), error) {
	if s.redis == nil {
		return func() {}, nil
	}

	count, err := counterGuardAcquireScript.Run(ctx, s.redis, []string{key}, ttl.Milliseconds()).Int64()
	if err != nil {
		return func() {}, nil
	}
	if count <= int64(limit) {
		return func() {
			_, _ = s.redis.Decr(context.Background(), key).Result()
		}, nil
	}

	_, _ = s.redis.Decr(context.Background(), key).Result()
	return func() {}, limitErr
}

func activeUserKey(rootUserID int64) string {
	return fmt.Sprintf("campus:agent:active:user:%d", rootUserID)
}

func activeConversationKey(sessionID string) string {
	return "campus:agent:active:conversation:" + sessionID
}

func rateLimitKey(rootUserID int64, now time.Time) string {
	return fmt.Sprintf("campus:agent:rate:%d:%s", rootUserID, now.Format("200601021504"))
}
