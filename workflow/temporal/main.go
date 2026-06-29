package temporal

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/minhgiang16983/Minh-Kit-Hehe/lifecycle"
	"github.com/minhgiang16983/Minh-Kit-Hehe/logger"
	temporalClient "go.temporal.io/sdk/client"
	"google.golang.org/grpc"
)

var _ lifecycle.Component = (*Client)(nil)

type TemporalAuth struct {
	ApiKey string `mapstructure:"api_key" yaml:"api_key"`
}

type TemporalConfig struct {
	Host      string       `mapstructure:"host" yaml:"host"`
	Port      int          `mapstructure:"port" yaml:"port"`
	Namespace string       `mapstructure:"namespace" yaml:"namespace"`
	Auth      TemporalAuth `mapstructure:"auth" yaml:"auth"`
}

// Client wraps a Temporal SDK client with lifecycle hooks.
type Client struct {
	temporalClient.Client
}

func NewClient(ctx context.Context, cfg *TemporalConfig, l logger.LoggerInterface) (*Client, error) {
	client, err := dial(ctx, cfg, l)
	if err != nil {
		return nil, err
	}
	return &Client{Client: client}, nil
}

func (c *Client) Name() string { return "temporal" }

func (c *Client) Start(_ context.Context) error { return nil }

func (c *Client) Stop(_ context.Context) error {
	if c.Client == nil {
		return nil
	}
	c.Client.Close()
	return nil
}

func New(ctx context.Context, cfg *TemporalConfig, l logger.LoggerInterface) temporalClient.Client {
	client, err := dial(ctx, cfg, l)
	if err != nil {
		l.Fatal("Failed to initialize Temporal Client", l.ErrorField(err))
	}
	return client
}

func dial(ctx context.Context, cfg *TemporalConfig, l logger.LoggerInterface) (temporalClient.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	ipv4Dialer := func(ctx context.Context, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "tcp4", addr)
	}

	opts := temporalClient.Options{
		HostPort:  fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Namespace: cfg.Namespace,
		Logger:    NewLogger(l),
		ConnectionOptions: temporalClient.ConnectionOptions{
			TLS: &tls.Config{},
			DialOptions: []grpc.DialOption{
				grpc.WithContextDialer(ipv4Dialer),
			},
		},
	}

	if cfg.Auth.ApiKey != "-" {
		opts.Credentials = temporalClient.NewAPIKeyStaticCredentials(cfg.Auth.ApiKey)
	}

	temporal, err := temporalClient.DialContext(ctx, opts)
	if err != nil {
		return nil, err
	}

	l.Info(
		"Connected to temporal client",
		l.String("host", cfg.Host),
		l.Int("port", cfg.Port),
		l.String("namespace", cfg.Namespace),
	)

	return temporal, nil
}
