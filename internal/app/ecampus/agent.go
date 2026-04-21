package ecampus

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Milchstrassse/Ecampus-go/internal/agentchat"
	"github.com/Milchstrassse/Ecampus-go/internal/app/bootstrap"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/config"
	"github.com/Milchstrassse/Ecampus-go/internal/platform/jwtutil"
)

const defaultAgentConnectTimeout = 5 * time.Second

func newAgentHandler(infra *bootstrap.Infra, jwtHelper *jwtutil.Helper) (*agentchat.Handler, func(), error) {
	if infra == nil {
		return nil, func() {}, fmt.Errorf("infra is nil")
	}

	svc := agentchat.NewService(infra.MySQL, infra.Redis, infra.Config, infra.Logger, nil)
	handler := agentchat.NewHandler(svc, jwtHelper, infra.Redis)
	if infra.Config == nil || !infra.Config.Agent.Enabled {
		return handler, func() {}, nil
	}

	if err := agentchat.EnsureSchema(infra.MySQL); err != nil {
		return nil, func() {}, err
	}

	conn, err := dialAgentConn(infra.Config)
	if err != nil {
		return nil, func() {}, err
	}

	client := agentchat.NewClient(conn, infra.Config.Agent.AuthToken)
	if err := checkAgentHealth(client, infra.Config); err != nil {
		_ = conn.Close()
		return nil, func() {}, err
	}

	infra.Logger.Info("agent grpc ready", zap.String("addr", infra.Config.Agent.GRPCAddr))
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
	ctx, cancel := context.WithTimeout(context.Background(), agentConnectTimeout(cfg.Agent.ConnectTimeoutMS))
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		cfg.Agent.GRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("dial agent grpc %s: %w", cfg.Agent.GRPCAddr, err)
	}
	return conn, nil
}

func checkAgentHealth(client *agentchat.Client, cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), agentConnectTimeout(cfg.Agent.ConnectTimeoutMS))
	defer cancel()

	if err := client.CheckHealth(ctx); err != nil {
		return fmt.Errorf("check agent grpc health %s: %w", cfg.Agent.GRPCAddr, err)
	}
	return nil
}

func agentConnectTimeout(timeoutMS int) time.Duration {
	if timeoutMS <= 0 {
		return defaultAgentConnectTimeout
	}
	return time.Duration(timeoutMS) * time.Millisecond
}
