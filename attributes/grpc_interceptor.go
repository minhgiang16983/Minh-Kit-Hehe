package attributes

import (
	"context"

	"google.golang.org/grpc"
)

// AttributeExtractionUnaryInterceptor creates a gRPC unary interceptor that extracts
// request attributes from metadata and injects them into the context
func DefaultAttributeExtractionUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Create attribute extractor for gRPC request
		extractor := NewGRPCAttributeExtractor(ctx, info)

		// Extract attributes
		attrs, err := extractor.Extract(ctx)
		if err != nil {
			// Log error but continue processing
			// You might want to add proper logging here
			return handler(ctx, req)
		}

		// Inject attributes into context
		ctx = WithRequestAttributes(ctx, attrs)

		// Continue to next interceptor/handler
		return handler(ctx, req)
	}
}
