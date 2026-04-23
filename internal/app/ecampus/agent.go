package ecampus

import (
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Milchstrassse/Ecampus-go/internal/agentchat"
	"github.com/Milchstrassse/Ecampus-go/internal/app/bootstrap"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
)

func newAgentHandler(infra *bootstrap.Infra, jwtHelper *jwtutil.Helper) (*agentchat.Handler, func(), error) {
	if infra == nil {
		return nil, func() {}, fmt.Errorf("infra is nil")
	}
	if infra.Config == nil {
		return nil, func() {}, fmt.Errorf("config is nil")
	}

	conn, err := dialAgentConn(infra.Config)
	if err != nil {
		return nil, func() {}, err
	}

	client := agentchat.NewClient(conn, infra.Config.Agent.AuthToken)
	infra.Logger.Info("agent grpc configured", zap.String("addr", infra.Config.Agent.GRPCAddr))
	return agentchat.NewHandler(
			agentchat.NewService(infra.MySQL, infra.Redis, infra.Config, infra.Logger, client),
			jwtHelper,
			infra.Redis,
		), func() {
			if err := conn.Close(); err != nil {
				infra.Logger.Warn("close agent grpc failed", zap.Error(err))
			}
		}, nil
}

func dialAgentConn(cfg *config.Config) (*grpc.ClientConn, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if cfg.Agent.GRPCAddr == "" {
		return nil, fmt.Errorf("agent grpc addr is empty")
	}

	conn, err := grpc.NewClient(
		cfg.Agent.GRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial agent grpc %s: %w", cfg.Agent.GRPCAddr, err)
	}
	return conn, nil
}
