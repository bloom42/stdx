package middleware

import (
	"context"
	"net/http"

	"github.com/bloom42/stdx/guid"
)

type requestIDContextKey struct{}

// RequestIDCtxKey is the key that holds the unique request ID in a request context.
var RequestIDCtxKey = requestIDContextKey{}

func RequestID(header string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			requestID := guid.NewTimeBased()

			w.Header().Set(header, requestID.String())

			ctx := r.Context()
			ctx = context.WithValue(ctx, RequestIDCtxKey, requestID)

			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}
