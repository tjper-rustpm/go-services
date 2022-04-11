package controller

import (
	"context"
	"fmt"

	"github.com/tjper/rustcron/cmd/cronman/rcon"
	"go.uber.org/zap"
)

// NewRconHub creates an RconHub instance.
func NewRconHub(logger *zap.Logger) *RconHub {
	return &RconHub{
		logger: logger,
	}
}

// RconHub is responsible for managing access to many cronman servers' Rcon
// functionality. Enclosing this functionality into a type allows for simple
// mocking.
type RconHub struct {
	logger *zap.Logger
}

// Dial creates an IRcon implementation using the specified url and password.
func (h RconHub) Dial(ctx context.Context, url, password string) (IRcon, error) {
	h.logger.Info("dialing rcon server...", zap.String("url", url))
	defer h.logger.Info("rcon server dialed.", zap.String("url", url))

	return rcon.Dial(
		ctx,
		zap.NewExample(),
		fmt.Sprintf("ws://%s/%s", url, password),
	)
}
