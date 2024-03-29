package rcon

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// IRcon represents the rcon API by which a process may communicate with a
// cronman Rust server.
type IRcon interface {
	Close()
	Quit(context.Context) error
	Say(context.Context, string) error
	AddModerator(context.Context, string) error
	RemoveModerator(context.Context, string) error
	AddOwner(context.Context, string) error
	RemoveOwner(context.Context, string) error
	GrantPermission(context.Context, string, string) error
	RevokePermission(context.Context, string, string) error
	CreateGroup(context.Context, string) error
	AddToGroup(context.Context, string, string) error
	ServerInfo(context.Context) (*ServerInfo, error)
}

// NewHub creates a Hub instance.
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		logger: logger,
	}
}

// Hub is responsible for managing access to many cronman servers' Rcon
// functionality. Enclosing this functionality into a type allows for simple
// mocking.
type Hub struct {
	logger *zap.Logger
}

// Dial creates an IRcon implementation using the specified url and password.
func (h Hub) Dial(ctx context.Context, url, password string) (IRcon, error) {
	return Dial(
		ctx,
		zap.NewExample(),
		fmt.Sprintf("ws://%s/%s", url, password),
	)
}
