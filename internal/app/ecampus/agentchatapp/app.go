package agentchatapp

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/Milchstrassse/Ecampus-go/internal/agentchat"
	"github.com/Milchstrassse/Ecampus-go/internal/app/ecampusruntime"
)

func Run() error {
	app, err := ecampusruntime.New(false)
	if err != nil {
		return err
	}
	defer app.Close(context.Background())

	if app.Infra.Config.Agent.GRPCAddr == "" {
		return fmt.Errorf("agent grpc addr is empty")
	}
	conn, err := grpc.NewClient(
		app.Infra.Config.Agent.GRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("dial agent grpc %s: %w", app.Infra.Config.Agent.GRPCAddr, err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			app.Infra.Logger.Warn("close agent grpc failed", zap.Error(err))
		}
	}()

	client := agentchat.NewClient(conn, app.Infra.Config.Agent.AuthToken)
	handler := agentchat.NewHandler(
		agentchat.NewService(app.Infra.MySQL, app.Infra.Redis, app.Infra.Config, app.Infra.Logger, client),
		app.JWTHelper,
		app.Infra.Redis,
	)
	agentchat.RegisterInfraRoutes(app.Engine, handler)
	agentchat.RegisterProtectedRoutes(app.ProtectedAPI(), handler)

	return app.Run("ecampus-agentchat")
}
