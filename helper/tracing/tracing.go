package tracinghelper

import (
	"context"

	"github.com/minhgiang16983/Minh-Kit-Hehe/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (newCtx context.Context, end func()) {
	if tracing.GetGlobalTracingConfig().IsEnableTracing() {
		tr := otel.GetTracerProvider().Tracer(tracing.GetGlobalTracingConfig().ServiceName)
		ctx2, span := tr.Start(ctx, name)
		if len(attrs) > 0 {
			span.SetAttributes(attrs...)
		}
		return ctx2, func() { span.End() }
	}
	return ctx, func() {}
}
