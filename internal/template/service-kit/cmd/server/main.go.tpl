package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/minhgiang16983/service-kit/config"
	"github.com/minhgiang16983/service-kit/infra"
	"github.com/minhgiang16983/service-kit/internal/services"
	"github.com/minhgiang16983/Minh-Kit-Hehe/logger"
	"github.com/minhgiang16983/Minh-Kit-Hehe/secrets"
	"github.com/minhgiang16983/service-kit/metrics"
	pb "github.com/minhgiang16983/service-kit/pb"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
)

func main() {
	ctx := context.Background()
	
	// Load environment variables from AWS Secrets Manager (if enabled)
	// This must run before config.Load() to ensure secrets are available
	if err := secrets.LoadFromAWSSecretsManager(ctx); err != nil {
		// Use fmt for error since logger might not be initialized yet
		fmt.Fprintf(os.Stderr, "⚠️  Failed to load secrets from AWS Secrets Manager: %v\n", err)
		// Continue execution - secrets loading is optional
	}

	l := logger.New()
	cfg, err := config.Load()

	l.Info("Config loaded")

	if err != nil {
		//  Need to log
		l.Fatal("failed to load config", l.String("error", err.Error()))
		return
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Init metric
	metrics.New()

	// Init grpc server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
	)

	// Register metric collector
	grpc_prometheus.Register(grpcServer)

	// Infra init
	infrastructure, err := infra.New(cfg, l)
	if err != nil {
		l.Fatal("failed to initialize infrastructure", l.String("error", err.Error()))
	}

	// Init new project
	pbService := services.New(infrastructure, l)
	pb.RegisterServiceKitServer(grpcServer, pbService)

	// Register signal
	ctxSignal, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Run GRPC gateway
	r, err := RunGrpcGateway(ctx, cfg.GrpcPort)
	if err != nil {
		l.Fatal("failed to run grpc gateway", l.String("error", err.Error()))
	}

	// Listen for grpc, Port 10000
	grpcListener, err := net.Listen("tcp4", fmt.Sprintf(":%d", cfg.GrpcPort))
	if err != nil {
		l.Fatal("Cannot listen grpc port ", l.String("error", err.Error()))
	}

	go func() {
		l.Info("Grpc server started with", l.Int("port", cfg.GrpcPort))

		if err := grpcServer.Serve(grpcListener); err != nil {
			l.Fatal("Cannot serve grpc", l.String("error", err.Error()))
		}
	}()

	l.Info("Http server started with", l.Int("port", cfg.HttpPort))
	err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.HttpPort), r)
	if err != nil {
		l.Fatal("failed to http server", l.String("error", err.Error()))
	}

	// Graceful shutdown
	<-ctxSignal.Done()

	l.Info("Graceful shutdown")

	// Stop grpc server
	go func() {
		grpcServer.GracefulStop()
		if err := grpcListener.Close(); err != nil {
			l.Fatal("failed to close grpc listener", l.String("error", err.Error()))
		}
	}()

	// Stop http server
}
