package worker

import (
	"context"

	"github.com/minhgiang16983/Minh-Kit-Hehe/logger"
)

type WorkerInterface interface {
	Run()
}

type WorkerHandlerInterface interface {
	Handle(ctx context.Context, numberWorker int) error
}

type Worker struct {
	Name          string
	Ctx           context.Context
	NumOfWorker   int
	WorkerHandler WorkerHandlerInterface
	l             logger.LoggerInterface
}

func Init(ctx context.Context, name string, numWorker int, workerHandler WorkerHandlerInterface, l logger.LoggerInterface) WorkerInterface {
	return &Worker{
		Name:          name,
		Ctx:           ctx,
		NumOfWorker:   numWorker,
		WorkerHandler: workerHandler,
		l:             l,
	}
}

func (w *Worker) Run() {
	for i := 0; i < w.NumOfWorker; i++ {
		// Run tasks
		go func(i int) {

			ctx := InjectWorkerName(w.Ctx, w.Name, i)

			if err := w.WorkerHandler.Handle(ctx, i); err != nil {
				w.l.Error("Worker error", w.l.Int("index", i), w.l.ErrorField(err))
			}
		}(i)
	}
}
