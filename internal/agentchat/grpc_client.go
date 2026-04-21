package agentchat

import (
	"context"

	agentv1 "github.com/Milchstrassse/Ecampus-go/internal/agentchat/agentv1"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
)

type Client struct {
	agent  agentv1.AgentServiceClient
	health healthpb.HealthClient
	token  string
}

func NewClient(conn grpc.ClientConnInterface, token string) *Client {
	return &Client{
		agent:  agentv1.NewAgentServiceClient(conn),
		health: healthpb.NewHealthClient(conn),
		token:  token,
	}
}

func (c *Client) CheckHealth(ctx context.Context) error {
	_, err := c.health.Check(c.withAuth(ctx), &healthpb.HealthCheckRequest{})
	return err
}

func (c *Client) HandleTurn(ctx context.Context, req *agentv1.HandleTurnRequest) (*agentv1.HandleTurnResponse, error) {
	return c.agent.HandleTurn(c.withAuth(ctx), req)
}

func (c *Client) StreamHandleTurn(ctx context.Context, req *agentv1.HandleTurnRequest) (agentv1.AgentService_StreamHandleTurnClient, error) {
	return c.agent.StreamHandleTurn(c.withAuth(ctx), req)
}

func (c *Client) GetSessionHistory(ctx context.Context, req *agentv1.GetSessionHistoryRequest) (*agentv1.GetSessionHistoryResponse, error) {
	return c.agent.GetSessionHistory(c.withAuth(ctx), req)
}

func (c *Client) DeleteSession(ctx context.Context, req *agentv1.DeleteSessionRequest) (*agentv1.DeleteSessionResponse, error) {
	return c.agent.DeleteSession(c.withAuth(ctx), req)
}

func (c *Client) withAuth(ctx context.Context) context.Context {
	if c == nil || c.token == "" {
		return ctx
	}
	return metadata.NewOutgoingContext(ctx, metadata.Pairs("x-agent-core-token", c.token))
}
