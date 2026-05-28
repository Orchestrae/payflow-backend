package middleware

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
)

const RequestIDKey ContextKey = "requestID"
const RequestIDHeader = "X-Request-ID"

// RequestID generates a unique request ID and injects it into the context and response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use existing request ID from header if provided (e.g., from load balancer)
		reqID := r.Header.Get(RequestIDHeader)
		if reqID == "" {
			reqID = generateID()
		}

		w.Header().Set(RequestIDHeader, reqID)
		ctx := context.WithValue(r.Context(), RequestIDKey, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
