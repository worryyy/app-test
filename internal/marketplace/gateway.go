package marketplace

import (
	"context"
	"strings"
)

type Gateway interface {
	Name() string
	CreatePayment(context.Context, string, int64) (string, error)
	Refund(context.Context, string, int64) (string, error)
	Settle(context.Context, string, int64) (string, error)
}

type testGateway struct{}
type disabledGateway struct{}

func NewGateway(profile string) Gateway {
	profile = strings.ToLower(strings.TrimSpace(profile))
	if profile == "" {
		profile = "dev"
	}
	if profile == "dev" || profile == "test" {
		return testGateway{}
	}
	return disabledGateway{}
}

func (testGateway) Name() string { return "test" }
func (testGateway) CreatePayment(_ context.Context, requestID string, _ int64) (string, error) {
	return "test-pay-" + requestID, nil
}
func (testGateway) Refund(_ context.Context, requestID string, _ int64) (string, error) {
	return "test-refund-" + requestID, nil
}
func (testGateway) Settle(_ context.Context, requestID string, _ int64) (string, error) {
	return "test-settlement-" + requestID, nil
}

func (disabledGateway) Name() string { return "disabled" }
func (disabledGateway) CreatePayment(context.Context, string, int64) (string, error) {
	return "", ErrPaymentDisabled
}
func (disabledGateway) Refund(context.Context, string, int64) (string, error) {
	return "", ErrPaymentDisabled
}
func (disabledGateway) Settle(context.Context, string, int64) (string, error) {
	return "", ErrPaymentDisabled
}
