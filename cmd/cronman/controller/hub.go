package controller

import (
	"context"
	"fmt"

	"github.com/tjper/rustcron/cmd/cronman/rcon"
	"go.uber.org/zap"
)

func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		logger: logger,
	}
}

// Hub is an IRcon hub.
type Hub struct {
	logger *zap.Logger
}

// Dial creates an IRcon implementation using the specified url and password.
func (h Hub) Dial(ctx context.Context, url, password string) (IRcon, error) {
	h.logger.Info("dialing rcon server...", zap.String("url", url))
	defer h.logger.Info("rcon server dialed.", zap.String("url", url))

	return rcon.Dial(
		ctx,
		zap.NewExample(),
		fmt.Sprintf("ws://%s/%s", url, password),
	)
}
