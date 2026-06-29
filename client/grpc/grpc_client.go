package grpc

import (
	"context"

	"google.golang.org/grpc/metadata"

	"github.com/minhgiang16983/Minh-Kit-Hehe/tracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GrpcClientConfig struct {
	Address  string `mapstructure:"address" yaml:"address"`
	Prefix   string `mapstructure:"prefix" yaml:"prefix"`
	Insecure bool   `mapstructure:"insecure" yaml:"insecure"`
}

func NewClient(config *GrpcClientConfig, options ...grpc.DialOption) (*grpc.ClientConn, error) {

	opts := []grpc.DialOption{
		grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
	}
	if config.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	if tracing.GetGlobalTracingConfig().IsEnableGRPCTracing() {
		opts = append(opts, grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	}
	if config.Prefix != "" {
		opts = append(opts, grpc.WithUnaryInterceptor(WithUnaryPrefixInterceptor(config.Prefix)))
	}

	options = append(options, opts...)

	conn, err := grpc.NewClient(config.Address, options...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func WithUnaryPrefixInterceptor(prefix string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoker(ctx, prefix+method, req, reply, cc, opts...)
	}
}

func WithUnaryInjectServiceNameInterceptor(serviceName string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		ctxWithMetadata := metadata.AppendToOutgoingContext(ctx, "x-caller-base_service", serviceName)

		return invoker(ctxWithMetadata, method, req, reply, cc, opts...)
	}
}
