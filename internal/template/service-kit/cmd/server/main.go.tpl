package main

import (
	"context"
	"fmt"
	"os"

	"github.com/minhgiang16983/service-kit/config"
	"github.com/minhgiang16983/service-kit/infra"
	"github.com/minhgiang16983/service-kit/internal/services"
	"github.com/minhgiang16983/service-kit/metrics"
	pb "github.com/minhgiang16983/service-kit/pb"
	"github.com/minhgiang16983/Minh-Kit-Hehe/lifecycle"
	"github.com/minhgiang16983/Minh-Kit-Hehe/logger"
	"github.com/minhgiang16983/Minh-Kit-Hehe/secrets"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
)

func main() {
	ctx := context.Background()

	if err := secrets.LoadFromAWSSecretsManager(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load secrets from AWS Secrets Manager: %v\n", err)
	}

	l := logger.New()
	cfg, err := config.Load()
	if err != nil {
		l.Fatal("failed to load config", l.String("error", err.Error()))
	}
	l.Info("config loaded")

	metrics.New()

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
	)
	grpc_prometheus.Register(grpcServer)

	infrastructure, err := infra.New(cfg, l)
	if err != nil {
		l.Fatal("failed to initialize infrastructure", l.String("error", err.Error()))
	}

	pbService := services.New(infrastructure, l)
	pb.RegisterServiceKitServer(grpcServer, pbService)

	lc := lifecycle.New(l)
	registerLifecycleComponents(lc, infrastructure, cfg, l, grpcServer, ctx)

	if err := lc.Run(ctx); err != nil {
		l.Fatal("application stopped with error", l.String("error", err.Error()))
	}
}

func registerLifecycleComponents(
	lc *lifecycle.Lifecycle,
	infrastructure infra.InfraInterface,
	cfg *config.Config,
	l logger.LoggerInterface,
	grpcServer *grpc.Server,
	ctx context.Context,
) {
	// 1. Infrastructure: mariadb -> redis -> kafka
	lc.Register(infrastructure.Components()...)

	// 2. Kafka consumers/producers (register after kafka client, before servers)
	// kafkaClient := infrastructure.GetKafkaClient()
	// _ = kafkaClient.InitConsumer("my-consumer", cfg.Kafka.GroupID, nil)
	// lc.Register(kafka.NewMessageConsumerComponent(kafkaClient, "my-consumer", []string{"topic"}, handler))
	// lc.Register(kafka.NewAsyncProducerComponent(kafkaClient, "my-producer"))

	// 3. Application servers
	lc.Register(
		NewGrpcServerComponent(cfg, l, grpcServer),
		NewHttpServerComponent(ctx, cfg, l),
	)
}
