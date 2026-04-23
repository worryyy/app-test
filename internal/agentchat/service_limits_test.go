package agentchat

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
)

func TestAcquireCounterGuardSetsTTLAndEnforcesLimit(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() {
		_ = rdb.Close()
	}()

	svc := NewService(nil, rdb, &config.Config{}, zap.NewNop(), nil)
	release, err := svc.acquireCounterGuard(context.Background(), "campus:agent:test", 1, 2*time.Minute, ErrAgentConversationBusy)
	if err != nil {
		t.Fatalf("acquireCounterGuard error: %v", err)
	}
	defer release()

	ttl := mr.TTL("campus:agent:test")
	if ttl <= 0 {
		t.Fatalf("ttl = %s, want > 0", ttl)
	}

	_, err = svc.acquireCounterGuard(context.Background(), "campus:agent:test", 1, 2*time.Minute, ErrAgentConversationBusy)
	if err != ErrAgentConversationBusy {
		t.Fatalf("second acquire error = %v, want %v", err, ErrAgentConversationBusy)
	}
}
