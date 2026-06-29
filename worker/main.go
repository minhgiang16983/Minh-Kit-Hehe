package worker

import (
	"context"
	"sync"

	"github.com/minhgiang16983/Minh-Kit-Hehe/lifecycle"
	"github.com/minhgiang16983/Minh-Kit-Hehe/logger"
)

var _ lifecycle.Component = (*Worker)(nil)

type WorkerInterface interface {
	lifecycle.Component
}

type WorkerHandlerInterface interface {
	Handle(ctx context.Context, numberWorker int) error
}

type Worker struct {
	WorkerName    string
	Ctx           context.Context
	NumOfWorker   int
	WorkerHandler WorkerHandlerInterface
	l             logger.LoggerInterface
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

func Init(ctx context.Context, name string, numWorker int, workerHandler WorkerHandlerInterface, l logger.LoggerInterface) WorkerInterface {
	return &Worker{
		WorkerName:    name,
		Ctx:           ctx,
		NumOfWorker:   numWorker,
		WorkerHandler: workerHandler,
		l:             l,
	}
}

func (w *Worker) Name() string {
	if w.WorkerName != "" {
		return w.WorkerName
	}
	return "worker"
}

func (w *Worker) Start(ctx context.Context) error {
	baseCtx := ctx
	if w.Ctx != nil {
		baseCtx = w.Ctx
	}

	runCtx, cancel := context.WithCancel(baseCtx)
	w.cancel = cancel

	for i := 0; i < w.NumOfWorker; i++ {
		w.wg.Add(1)
		go func(index int) {
			defer w.wg.Done()
			workerCtx := InjectWorkerName(runCtx, w.WorkerName, index)
			if err := w.WorkerHandler.Handle(workerCtx, index); err != nil {
				w.l.Error("worker error", w.l.Int("index", index), w.l.ErrorField(err))
			}
		}(i)
	}

	return nil
}

func (w *Worker) Stop(_ context.Context) error {
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
	return nil
}

func (w *Worker) Run() {
	_ = w.Start(context.Background())
}
