package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/minhgiang16983/service-kit/config"
	"github.com/minhgiang16983/Minh-Kit-Hehe/lifecycle"
	"github.com/minhgiang16983/Minh-Kit-Hehe/logger"
	"google.golang.org/grpc"
)

var (
	_ lifecycle.Component = (*GrpcServerComponent)(nil)
	_ lifecycle.Component = (*HttpServerComponent)(nil)
)

type GrpcServerComponent struct {
	cfg      *config.Config
	l        logger.LoggerInterface
	server   *grpc.Server
	listener net.Listener
}

func NewGrpcServerComponent(cfg *config.Config, l logger.LoggerInterface, server *grpc.Server) *GrpcServerComponent {
	return &GrpcServerComponent{
		cfg:    cfg,
		l:      l,
		server: server,
	}
}

func (c *GrpcServerComponent) Name() string { return "grpc-server" }

func (c *GrpcServerComponent) Start(_ context.Context) error {
	listener, err := net.Listen("tcp4", fmt.Sprintf(":%d", c.cfg.GrpcPort))
	if err != nil {
		return fmt.Errorf("listen grpc port: %w", err)
	}
	c.listener = listener

	go func() {
		c.l.Info("grpc server started", c.l.Int("port", c.cfg.GrpcPort))
		if err := c.server.Serve(listener); err != nil {
			c.l.Error("grpc server stopped", c.l.ErrorField(err))
		}
	}()

	return nil
}

func (c *GrpcServerComponent) Stop(_ context.Context) error {
	if c.server != nil {
		c.server.GracefulStop()
	}
	if c.listener != nil {
		if err := c.listener.Close(); err != nil {
			return fmt.Errorf("close grpc listener: %w", err)
		}
	}
	return nil
}

type HttpServerComponent struct {
	ctx    context.Context
	cfg    *config.Config
	l      logger.LoggerInterface
	server *http.Server
}

func NewHttpServerComponent(ctx context.Context, cfg *config.Config, l logger.LoggerInterface) *HttpServerComponent {
	return &HttpServerComponent{
		ctx: ctx,
		cfg: cfg,
		l:   l,
	}
}

func (c *HttpServerComponent) Name() string { return "http-server" }

func (c *HttpServerComponent) Start(_ context.Context) error {
	handler, err := RunGrpcGateway(c.ctx, c.cfg.GrpcPort)
	if err != nil {
		return fmt.Errorf("init grpc gateway: %w", err)
	}

	c.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", c.cfg.HttpPort),
		Handler: handler,
	}

	go func() {
		c.l.Info("http server started", c.l.Int("port", c.cfg.HttpPort))
		if err := c.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			c.l.Error("http server stopped", c.l.ErrorField(err))
		}
	}()

	return nil
}

func (c *HttpServerComponent) Stop(ctx context.Context) error {
	if c.server == nil {
		return nil
	}
	if err := c.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}
	return nil
}
