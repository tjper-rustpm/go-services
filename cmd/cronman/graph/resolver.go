//go:generate go run github.com/99designs/gqlgen
package graph

import (
	"context"

	"github.com/tjper/rustcron/cmd/cronman/db/model"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// IController represents the API by which the Resolver interacts with the
// Controller.
type IController interface {
	CreateServer(context.Context, model.ServerDefinition) (*model.DormantServer, error)
	ArchiveServer(context.Context, uuid.UUID) (*model.ArchivedServer, error)
	StartServer(context.Context, uuid.UUID) (*model.DormantServer, error)
	MakeServerLive(context.Context, uuid.UUID) (*model.LiveServer, error)
	StopServer(context.Context, uuid.UUID) (*model.DormantServer, error)
	UpdateServer(context.Context, uuid.UUID, map[string]interface{}) (*model.ServerDefinition, error)

	AddServerModerators(context.Context, uuid.UUID, model.DefinitionModerators) (*model.ServerDefinition, error)
	RemoveServerModerators(context.Context, uuid.UUID, []uuid.UUID) (*model.ServerDefinition, error)

	AddServerTags(context.Context, uuid.UUID, model.DefinitionTags) (*model.ServerDefinition, error)
	RemoveServerTags(context.Context, uuid.UUID, []uuid.UUID) (*model.ServerDefinition, error)

	AddServerEvents(context.Context, uuid.UUID, model.DefinitionEvents) (*model.ServerDefinition, error)
	RemoveServerEvents(context.Context, uuid.UUID, []uuid.UUID) (*model.ServerDefinition, error)

	ListServers(context.Context, interface{}) error
}

// Resolver resolves graphql queries and mutations.
type Resolver struct {
	logger *zap.Logger
	ctrl   IController
}

// NewResolver creates a new Resolver object.
func NewResolver(
	logger *zap.Logger,
	ctrl IController,
) *Resolver {
	return &Resolver{
		logger: logger,
		ctrl:   ctrl,
	}
}
