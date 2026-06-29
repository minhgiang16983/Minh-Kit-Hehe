package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/minhgiang16983/Minh-Kit-Hehe/logger"
	"github.com/minhgiang16983/Minh-Kit-Hehe/tracing"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

type HttpMiddleware func(http.Handler) http.Handler

type ServerEndFunc func(ctx context.Context)

type ServerConfig struct {
	HttpServerConfig *HttpServerConfig

	GrpcServerConfig *GrpcServerConfig
}

type GrpcServerConfig struct {
	Host               string
	Port               int
	GrpcOptions        []grpc.ServerOption
	UnaryInterceptors  []grpc.UnaryServerInterceptor
	StreamInterceptors []grpc.StreamServerInterceptor
}

type HttpServerConfig struct {
	Host               string
	Port               int
	HttpMiddlewares    []HttpMiddleware
	EnableGrpcGateway  bool
	GrpcGatewayOptions []runtime.ServeMuxOption
}
type Server struct {
	cfg         *ServerConfig
	grpcServer  *grpc.Server
	httpServer  *http.ServeMux
	grpcGateway *runtime.ServeMux
	l           logger.LoggerInterface
}

func InitGrpcServer(config *GrpcServerConfig) *grpc.Server {
	recoverHandler := grpc_recovery.WithRecoveryHandlerContext(func(ctx context.Context, p interface{}) error {
		l := logger.GetLogger(ctx)
		if panicErr, ok := p.(*grpc_recovery.PanicError); ok {
			l.Error(
				"gRPC panic recovered",
				l.Any("panic", panicErr.Panic),
				l.String("stack", string(panicErr.Stack)),
			)
			return status.Error(codes.Internal, "internal server error")
		}

		l.Error(
			"gRPC panic recovered",
			l.Any("panic", p),
			l.String("stack", string(debug.Stack())),
		)
		return status.Error(codes.Internal, "internal server error")
	})

	unaryInterceptors := []grpc.UnaryServerInterceptor{
		grpc_recovery.UnaryServerInterceptor(recoverHandler),
		grpc_prometheus.UnaryServerInterceptor,
	}
	streamInterceptors := []grpc.StreamServerInterceptor{
		grpc_recovery.StreamServerInterceptor(recoverHandler),
		grpc_prometheus.StreamServerInterceptor,
	}

	grpc_prometheus.EnableHandlingTimeHistogram()
	grpc_prometheus.EnableClientHandlingTimeHistogram()

	if config.UnaryInterceptors != nil {
		unaryInterceptors = append(unaryInterceptors, config.UnaryInterceptors...)
	}
	if config.StreamInterceptors != nil {
		streamInterceptors = append(streamInterceptors, config.StreamInterceptors...)
	}

	options := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
		grpc.ChainStreamInterceptor(streamInterceptors...),
	}

	if config.GrpcOptions != nil {
		options = append(options, config.GrpcOptions...)
	}

	if tracing.GetGlobalTracingConfig().IsEnableGRPCTracing() {
		options = append(options, grpc.StatsHandler(otelgrpc.NewServerHandler()))
		grpc.EnableTracing = true

	}
	// Init grpc server
	grpcServer := grpc.NewServer(
		options...,
	)

	return grpcServer
}

func Init(cfg *ServerConfig) *Server {
	s := &Server{
		cfg: cfg,
	}
	if err := s.initilize(); err != nil {
		s.l.Fatal("failed to initialize server", s.l.ErrorField(err))
	}
	return s
}

func InitGrpcGateway(options ...runtime.ServeMuxOption) (*runtime.ServeMux, error) {
	defaultOptions := []runtime.ServeMuxOption{
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames:     true,
				EmitUnpopulated:   true,
				EmitDefaultValues: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
	}

	options = append(defaultOptions, options...)

	gatewayMux := runtime.NewServeMux(options...)

	return gatewayMux, nil
}

func InitHttpServer(config *HttpServerConfig) (*http.ServeMux, *runtime.ServeMux, error) {

	httpServer := http.NewServeMux()

	if config.EnableGrpcGateway {
		grpcGateway, err := InitGrpcGateway(config.GrpcGatewayOptions...)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to initialize grpc gateway: %w", err)
		}
		httpServer.Handle("/", grpcGateway)

		return httpServer, grpcGateway, nil

	}

	return httpServer, nil, nil
}

func (s *Server) initilize() error {
	if s.cfg.GrpcServerConfig != nil {
		s.grpcServer = InitGrpcServer(s.cfg.GrpcServerConfig)
	}
	if s.cfg.HttpServerConfig != nil {

		httpServer, grpcGateway, err := InitHttpServer(s.cfg.HttpServerConfig)
		if err != nil {
			return fmt.Errorf("failed to initialize http server: %w", err)
		}
		s.httpServer = httpServer
		s.grpcGateway = grpcGateway
	}
	return nil
}

func (s *Server) SetLogger(logger logger.LoggerInterface) *Server {
	s.l = logger
	return s
}

func (s *Server) GetGrpcServer() *grpc.Server {
	return s.grpcServer
}

func (s *Server) GetHttpServer() *http.ServeMux {
	return s.httpServer
}

func (s *Server) GetGrpcGateway() *runtime.ServeMux {
	return s.grpcGateway
}

func (s *Server) Run(signalCtx context.Context) ServerEndFunc {
	l := logger.GetLogger(signalCtx)
	shutdowns := make([]func(ctx context.Context), 0)
	// Listen for grpc, Port 10000

	if s.grpcServer != nil {
		if s.cfg.GrpcServerConfig.Port == 0 {
			l.Fatal("Grpc port is not set")
		}
		listenAddress := fmt.Sprintf("%s:%d", s.cfg.GrpcServerConfig.Host, s.cfg.GrpcServerConfig.Port)

		grpcListener, err := net.Listen("tcp4", listenAddress)
		if err != nil {
			l.Fatal("Cannot listen grpc port ", l.ErrorField(err))
		}

		grpc_prometheus.Register(s.grpcServer)

		go func() {
			l.Info("Grpc server started with", l.String("address", listenAddress))

			if errGrpc := s.grpcServer.Serve(grpcListener); errGrpc != nil {
				l.Fatal("Cannot serve grpc", l.ErrorField(errGrpc))
			}
		}()

		shutdowns = append(shutdowns, func(_ context.Context) {
			s.grpcServer.GracefulStop()
		})
	}

	if s.httpServer != nil {
		httpPort := s.cfg.HttpServerConfig.Port

		if httpPort == 0 {
			l.Fatal("Http port is not set")
		}

		listenAddress := fmt.Sprintf("%s:%d", s.cfg.HttpServerConfig.Host, s.cfg.HttpServerConfig.Port)
		var handler http.Handler = s.httpServer

		// Init http server

		for _, middleware := range s.cfg.HttpServerConfig.HttpMiddlewares {
			handler = middleware(handler)
		}

		if tracing.GetGlobalTracingConfig().IsEnableHTTPTracing() {
			handler = otelhttp.NewHandler(handler, "http", otelhttp.WithFilter(func(r *http.Request) bool {
				return !strings.HasPrefix(r.URL.Path, "/health")
			}))
		}

		httpServer := &http.Server{
			Addr:    listenAddress,
			Handler: handler,
		}

		go func() {
			l.Info("Http server started with", l.String("address", listenAddress))
			err := httpServer.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				l.Fatal("failed to http server", l.ErrorField(err))
			}
		}()

		shutdowns = append(shutdowns, func(ctx context.Context) {
			shutdownCtx := context.WithoutCancel(ctx)
			shutdownCtx, cancel := context.WithTimeout(shutdownCtx, 5*time.Second)
			defer cancel()

			if errShutdownHttpServer := httpServer.Shutdown(shutdownCtx); errShutdownHttpServer != nil {
				l.Fatal("failed to shutdown http server", l.ErrorField(errShutdownHttpServer))
			}
		})
	}

	<-signalCtx.Done()

	return func(ctx context.Context) {
		// Graceful shutdown
		l.Info("Starting Graceful server shutdown...")

		wg := sync.WaitGroup{}
		// Stop grpc server
		for _, shutdown := range shutdowns {
			wg.Add(1)
			go func(fn func(ctx context.Context)) {
				defer wg.Done()
				fn(ctx)
			}(shutdown)
		}
		wg.Wait()
		l.Info("Graceful shutdown server completed")
	}
}
