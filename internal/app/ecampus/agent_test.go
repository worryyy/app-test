package ecampus

import (
	"testing"

	"go.uber.org/zap"

	"github.com/Milchstrassse/Ecampus-go/internal/app/bootstrap"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
)

func TestNewAgentHandlerDoesNotFailWhenAgentDialFails(t *testing.T) {
	handler, closeAgent, err := newAgentHandler(&bootstrap.Infra{
		Config: &config.Config{
			Agent: config.AgentConfig{
				GRPCAddr: "127.0.0.1:1",
			},
		},
		Logger: zap.NewNop(),
	}, nil)
	if err != nil {
		t.Fatalf("newAgentHandler error: %v", err)
	}
	if handler == nil {
		t.Fatalf("expected handler")
	}
	closeAgent()
}
