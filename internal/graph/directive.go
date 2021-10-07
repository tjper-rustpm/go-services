package graph

import (
	"context"
	"errors"

	gerrors "github.com/tjper/rustcron/internal/graph/errors"
	"github.com/tjper/rustcron/internal/graph/model"
	"github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"

	"github.com/99designs/gqlgen/graphql"
	"go.uber.org/zap"
)

// Retriever retrieves a Session for the current process.
type Retriever interface {
	Retrieve(context.Context, string) (*session.Session, error)
}

func NewDirective(logger *zap.Logger, retriever Retriever) *Directive {
	return &Directive{
		logger:    logger,
		retriever: retriever,
	}
}

type Directive struct {
	logger    *zap.Logger
	retriever Retriever
}

func (d Directive) IsAuthenticated(
	ctx context.Context,
	_ interface{},
	next graphql.Resolver,
) (interface{}, error) {
	d.logger.Info("ensuring request is part of an authenticated session ...")
	access, ok := http.AccessFromContext(ctx)
	if !ok {
		d.logger.Error("error retrieving HTTP access from context")
		return nil, gerrors.ErrInternalServer
	}
	sessionID, ok := access.SessionID()
	if !ok {
		return nil, gerrors.ErrUnauthenticated
	}
	sess, err := d.retriever.Retrieve(ctx, sessionID)
	if errors.Is(err, session.ErrSessionDNE) {
		return nil, gerrors.ErrUnauthenticated
	}
	if err != nil {
		d.logger.Error("error retrieving session", zap.Error(err))
		return nil, gerrors.ErrInternalServer
	}
	d.logger.Info(
		"request authenticated, session retrieved; adding to context",
		zap.String("sess-email", sess.User.Email),
	)
	ctx = session.WithSession(ctx, sess)
	return next(ctx)
}

func (d Directive) HasRole(
	ctx context.Context,
	_ interface{},
	next graphql.Resolver,
	role model.RoleKind,
) (interface{}, error) {
	sess, ok := session.FromContext(ctx)
	if !ok {
		d.logger.Error("error checking role; context session dne")
		return nil, gerrors.ErrInternalServer
	}
	d.logger.Info(
		"ensuring session user has required role",
		zap.String("user-role", string(sess.User.Role)),
		zap.String("required-role", string(role)),
	)
	if sess.User.Role != role {
		return nil, gerrors.ErrUnauthorized
	}
	return next(ctx)
}
