package ecampus

import (
	"testing"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Milchstrassse/Ecampus-go/internal/app/bootstrap"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
)

func TestNewAgentHandlerDoesNotFailWhenAgentDialFails(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	handler, closeAgent, err := newAgentHandler(&bootstrap.Infra{
		Config: &config.Config{
			Agent: config.AgentConfig{
				GRPCAddr:         "127.0.0.1:1",
				ConnectTimeoutMS: 50,
			},
		},
		Logger: zap.NewNop(),
		MySQL:  db,
	}, nil)
	if err != nil {
		t.Fatalf("newAgentHandler error: %v", err)
	}
	if handler == nil {
		t.Fatalf("expected handler")
	}
	closeAgent()
}
