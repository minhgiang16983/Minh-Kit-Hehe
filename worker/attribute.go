package worker

import (
	"context"
	"fmt"
)

type ctxKey struct{}

var workerKey = ctxKey{}

func InjectWorkerName(ctx context.Context, workerName string, index int) context.Context {
	val := fmt.Sprintf("%s-%d", workerName, index)
	newCtx := context.WithValue(ctx, workerKey, val)

	return newCtx
}

func ExtractWorkerName(ctx context.Context) string {
	val := ctx.Value(workerKey)
	workerName, ok := val.(string)
	if !ok {
		return ""
	}

	return workerName
}
