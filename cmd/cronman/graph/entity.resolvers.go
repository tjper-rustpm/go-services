package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/tjper/rustcron/cmd/cronman/graph/generated"
	"github.com/tjper/rustcron/cmd/cronman/graph/model"
)

func (r *entityResolver) FindServerDefinitionByID(ctx context.Context, id string) (*model.ServerDefinition, error) {
	panic(fmt.Errorf("not implemented"))
}

// Entity returns generated.EntityResolver implementation.
func (r *Resolver) Entity() generated.EntityResolver { return &entityResolver{r} }

type entityResolver struct{ *Resolver }
