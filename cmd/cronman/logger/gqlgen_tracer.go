package logger

import (
	"context"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"go.uber.org/zap"
)

var _ interface {
	graphql.HandlerExtension
	graphql.ResponseInterceptor
} = (*Tracer)(nil)

func NewTracer(logger *zap.Logger) *Tracer {
	return &Tracer{
		logger: logger,
	}
}

type Tracer struct {
	logger *zap.Logger
}

func (t Tracer) ExtensionName() string {
	return "Logging Trace"
}

func (t Tracer) Validate(_ graphql.ExecutableSchema) error {
	return nil
}

func (t Tracer) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	var (
		operationCtx = graphql.GetOperationContext(ctx)
		logger       = t.logger.With(ContextFields(ctx)...)
	)
	logger.Info("operation complete", zap.Duration("duration", time.Since(operationCtx.Stats.OperationStart)))
	return next(ctx)
}
