package tracing

import (
	"context"

	"github.com/minhgiang16983/Minh-Kit-Hehe/lifecycle"
)

var _ lifecycle.Component = (*Provider)(nil)

// Provider wraps OpenTelemetry setup with lifecycle hooks.
type Provider struct {
	config   *TracingConfig
	shutdown func(context.Context) error
}

func NewProvider(config *TracingConfig) *Provider {
	return &Provider{config: config}
}

func (p *Provider) Name() string { return "tracing" }

func (p *Provider) Start(ctx context.Context) error {
	shutdown, err := SetupOTelSDK(ctx, p.config)
	if err != nil {
		return err
	}
	p.shutdown = shutdown
	return nil
}

func (p *Provider) Stop(ctx context.Context) error {
	if p.shutdown == nil {
		return nil
	}
	return p.shutdown(ctx)
}
