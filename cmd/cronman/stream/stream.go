// Package stream provides an API for launching a Handler that reads and
// processes all cronman related events from the underlying stream.
package stream

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/db"
	"github.com/tjper/rustcron/cmd/cronman/model"
	"github.com/tjper/rustcron/cmd/cronman/rcon"
	"github.com/tjper/rustcron/internal/event"
	igorm "github.com/tjper/rustcron/internal/gorm"
	imodel "github.com/tjper/rustcron/internal/model"
	"github.com/tjper/rustcron/internal/stream"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// IStream encompasses all interactions with the event stream.
type IStream interface {
	Claim(context.Context, time.Duration) (*stream.Message, error)
	Read(context.Context) (*stream.Message, error)
	Ack(context.Context, *stream.Message) error
}

// IRconHub encompasses all interactions with the rcon Hub.
type IRconHub interface {
	Dial(context.Context, string, string) (rcon.IRcon, error)
}

// NewHandler creates a Handler instance.
func NewHandler(
	logger *zap.Logger,
	store *gorm.DB,
	stream IStream,
	rconHub IRconHub,
) *Handler {
	return &Handler{
		logger:  logger,
		store:   store,
		stream:  stream,
		rconHub: rconHub,
	}
}

// Handler is responsible for reading and processing cronman related event
// from the underlying IStream.
type Handler struct {
	logger  *zap.Logger
	store   *gorm.DB
	stream  IStream
	rconHub IRconHub
}

// Launch reads and processes the underlying IStream. This is a blocking
// method. The context may be cancelled to shutdown the Handler.
func (h Handler) Launch(ctx context.Context) error {
	for {
		m, err := h.read(ctx)
		if err != nil {
			return fmt.Errorf("stream Handler.read: %w", err)
		}

		eventI, err := event.Parse(m.Payload)
		if err != nil {
			h.logger.Error("parse event hash", zap.Error(err))
			continue
		}

		switch e := eventI.(type) {
		case *event.VipRefreshEvent:
			err = h.handleVipRefreshEvent(ctx, e)
		default:
			h.logger.Sugar().Debugf("unrecognized event; type: %T", e)
			// Continue to acknowledge event so it is not processed again by this
			// group queue.
		}
		if err != nil {
			h.logger.Error("while handle stream event", zap.Error(err))
		}
		if err := h.stream.Ack(ctx, m); err != nil {
			h.logger.Error("while acknowledge stream event", zap.Error(err))
		}
	}
}

// errInvalidVipRefresh indicates a received vip refresh event is in an invalid
// state.
var errInvalidVipRefresh = errors.New("invalid vip refresh")

func (h Handler) handleVipRefreshEvent(ctx context.Context, event *event.VipRefreshEvent) error {
	var errstr string
	switch {
	case event.SteamID == "":
		errstr = "refresh SteamID empty"
	case event.ServerID == uuid.Nil:
		errstr = "refresh ServerID empty"
	case event.ExpiresAt.Equal(time.Time{}):
		errstr = "refresh ExpiresAt empty"
	}
	if errstr != "" {
		return fmt.Errorf("%w: %s", errInvalidVipRefresh, errstr)
	}

	vip, err := db.GetVipByServerIDAndSteamID(ctx, h.store, event.ServerID, event.SteamID)
	if err != nil && !errors.Is(err, igorm.ErrNotFound) {
		return fmt.Errorf("while retrieving vip to process refresh: %w", err)
	}

	if errors.Is(err, igorm.ErrNotFound) {
		vip = &model.Vip{
			ServerID:  event.ServerID,
			SteamID:   event.SteamID,
			ExpiresAt: event.ExpiresAt,
		}
		if err := h.store.WithContext(ctx).Create(vip).Error; err != nil {
			return fmt.Errorf("while creating vip: %w", err)
		}
	}

	// Vip with the specified server ID and steam was found. Just update its
	// ExpiresAt field.
	if err == nil {
		if err := h.store.
			WithContext(ctx).
			Model(vip).
			Update("expires_at", event.ExpiresAt).Error; err != nil {
			return fmt.Errorf("while updating vip: %w", err)
		}
	}

	server := &model.Server{
		Model: imodel.Model{ID: event.ServerID},
	}
	if err := h.store.WithContext(ctx).First(server).Error; err != nil {
		return fmt.Errorf("while retrieving vip's server: %w", err)
	}

	if !server.IsLive() {
		return nil
	}

	// If server is live dial the server's rcon API and add the user to the
	// queued users.
	client, err := h.rconHub.Dial(
		ctx,
		fmt.Sprintf("%s:28016", server.ElasticIP),
		server.RconPassword,
	)
	if err != nil {
		return fmt.Errorf("while rconhub.Dial: %w", err)
	}
	defer client.Close()

	if err := client.AddToGroup(ctx, event.SteamID, rcon.VipGroup); err != nil {
		return fmt.Errorf("while client.AddToGroup: %w", err)
	}
	return nil
}

func (h Handler) read(ctx context.Context) (*stream.Message, error) {
	m, err := h.stream.Claim(ctx, time.Minute)
	if err == nil {
		return m, nil
	}
	if err != nil && !errors.Is(err, stream.ErrNoPending) {
		return nil, fmt.Errorf("stream.Claim: %w", err)
	}

	// stream.Claim has returned stream.ErrNoPending, therefore we may read
	// the stream.
	m, err = h.stream.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("stream.Read: %w", err)
	}
	return m, nil
}
