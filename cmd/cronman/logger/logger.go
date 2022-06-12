package logger

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// key is a key used to store and retrieve a logger from the context.
// SA1029: should not use built-in type string as key for value; define your
// own type to avoid collisions.
type key string

var loggerCtxKey key = "logger_context_key"

// withRequestID creates a new context a requestID value.
func withRequestID(ctx context.Context, requestID uuid.UUID) context.Context {
	return context.WithValue(ctx, loggerCtxKey, requestID)
}

// fromRequestID retrieves the requestID from the context if it exists.
func requestIDFromCtx(ctx context.Context) (uuid.UUID, bool) {
	val, ok := ctx.Value(loggerCtxKey).(uuid.UUID)
	return val, ok
}

// ContextFields checks the context for a set of fields and returns them for
// use in a zap.Logger if they are available.
func ContextFields(ctx context.Context) []zap.Field {
	fields := make([]zap.Field, 0)
	if requestID, ok := requestIDFromCtx(ctx); ok {
		fields = append(fields, zap.String("request_id", requestID.String()))
	}
	return fields
}

// Middleware extends the incoming request's context with request scoped
// information critical to logging.
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := withRequestID(r.Context(), uuid.New())
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}
