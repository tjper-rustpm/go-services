package graph

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"go.uber.org/zap"
)

// ErrorInterceptor intercepts a response and logs its errors using the passed
// logger.
func ErrorInterceptor(logger *zap.Logger) graphql.ResponseMiddleware {
	return func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
		var (
			errs = graphql.GetErrors(ctx)
		)
		if len(errs) > 0 {
			logger.Error("response interceptor", zap.Error(errs))
		}
		return next(ctx)
	}
}
