package attributes

import (
	"net/http"
)

// AttributeExtractionMiddleware creates middleware that extracts request attributes
// and injects them into the context for use by other middleware
func DefaultAttributeExtractionMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create attribute extractor for HTTP request
			extractor := NewHTTPAttributeExtractor(r)

			// Extract attributes
			attrs, err := extractor.Extract(r.Context())
			if err != nil {
				// Log error but continue processing
				// You might want to add proper logging here
				next.ServeHTTP(w, r)
				return
			}

			// Inject attributes into context
			ctx := WithRequestAttributes(r.Context(), attrs)

			// Create new request with updated context
			r = r.WithContext(ctx)

			// Continue to next middleware/handler
			next.ServeHTTP(w, r)
		})
	}
}
