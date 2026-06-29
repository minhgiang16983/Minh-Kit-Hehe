package temporal

import (
	"crypto/tls"
	"time"

	"context"
	"fmt"
	"net"

	"github.com/minhgiang16983/Minh-Kit-Hehe/logger"
	temporalClient "go.temporal.io/sdk/client"
	"google.golang.org/grpc"
)

type TemporalAuth struct {
	ApiKey string `mapstructure:"api_key" yaml:"api_key"`
}

type TemporalConfig struct {
	Host      string       `mapstructure:"host" yaml:"host"`
	Port      int          `mapstructure:"port" yaml:"port"`
	Namespace string       `mapstructure:"namespace" yaml:"namespace"`
	Auth      TemporalAuth `mapstructure:"auth" yaml:"auth"`
}

func New(ctx context.Context, cfg *TemporalConfig, l logger.LoggerInterface) temporalClient.Client {
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

	// Add API key
	if cfg.Auth.ApiKey != "-" {
		opts.Credentials = temporalClient.NewAPIKeyStaticCredentials(cfg.Auth.ApiKey)
	}

	// Initialize temporal client
	temporal, err := temporalClient.DialContext(ctx, opts)

	if err != nil {
		l.Fatal("Failed to initialize Temporal Client", l.ErrorField(err))
	}

	l.Info(
		"Connected to temporal client",
		l.String("host", cfg.Host),
		l.Int("port", cfg.Port),
		l.String("namespace", cfg.Namespace),
	)

	return temporal
}
