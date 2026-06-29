package ratelimit

import (
	"net/http"
)

// HTTPMiddleware returns an HTTP middleware that performs request rate limiting
func HTTPMiddleware(limiter Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			// Check rate limit
			if err := limiter.Limit(ctx); err != nil {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			// Continue to next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
