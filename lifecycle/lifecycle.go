package lifecycle

import (
	"context"
	"fmt"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/minhgiang16983/Minh-Kit-Hehe/logger"
)

// Component represents a runnable module with graceful shutdown support.
type Component interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Lifecycle manages ordered start and reverse-order stop of registered components.
type Lifecycle struct {
	components  []Component
	l           logger.LoggerInterface
	stopTimeout time.Duration
}

func New(l logger.LoggerInterface) *Lifecycle {
	return &Lifecycle{
		l:           l,
		stopTimeout: 30 * time.Second,
	}
}

func (lc *Lifecycle) SetStopTimeout(timeout time.Duration) {
	lc.stopTimeout = timeout
}

func (lc *Lifecycle) Register(components ...Component) {
	lc.components = append(lc.components, components...)
}

func (lc *Lifecycle) Run(ctx context.Context) error {
	for _, component := range lc.components {
		lc.l.Info("starting component", lc.l.String("component", component.Name()))
		if err := component.Start(ctx); err != nil {
			return fmt.Errorf("start %s: %w", component.Name(), err)
		}
	}

	signalCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-signalCtx.Done()
	lc.l.Info("shutdown signal received")

	stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), lc.stopTimeout)
	defer cancel()

	var wg sync.WaitGroup
	for i := len(lc.components) - 1; i >= 0; i-- {
		component := lc.components[i]
		wg.Add(1)
		go func(c Component) {
			defer wg.Done()
			lc.l.Info("stopping component", lc.l.String("component", c.Name()))
			if err := c.Stop(stopCtx); err != nil {
				lc.l.Error(
					"failed to stop component",
					lc.l.String("component", c.Name()),
					lc.l.ErrorField(err),
				)
			}
		}(component)
	}

	wg.Wait()
	lc.l.Info("graceful shutdown completed")
	return nil
}
