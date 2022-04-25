// Package stream provides an API for launching a Handler that reads and
// processes all cronman related events from the underlying stream.
package stream

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/model"
	"github.com/tjper/rustcron/internal/event"
	"github.com/tjper/rustcron/internal/gorm"
	"github.com/tjper/rustcron/internal/stream"

	"go.uber.org/zap"
)

// IStore encompasses all interactions with the payment store.
type IStore interface {
	Create(context.Context, gorm.Creator) error
}

// IStream encompasses all interactions with the event stream.
type IStream interface {
	Claim(context.Context, time.Duration) (*stream.Message, error)
	Read(context.Context) (*stream.Message, error)
}

// NewHandler creates a Handler instance.
func NewHandler(
	logger *zap.Logger,
	store IStore,
	stream IStream,
) *Handler {
	return &Handler{
		logger: logger,
		store:  store,
		stream: stream,
	}
}

// Handler is responsible for reading and processing cronman related event
// from the underlying IStream.
type Handler struct {
	logger *zap.Logger
	store  IStore
	stream IStream
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
		case *event.InvoicePaidEvent:
			err = h.handleInvoicePaidEvent(ctx, e)
		default:
			h.logger.Sugar().Debugf("unrecognized event; type: %T", e)
		}
		if err != nil {
			h.logger.Error("handle stream event", zap.Error(err))
			continue
		}

		if err := m.Ack(ctx); err != nil {
			h.logger.Error("acknowledge stream event", zap.Error(err))
		}
	}
}

func (h Handler) handleInvoicePaidEvent(ctx context.Context, event *event.InvoicePaidEvent) error {
	duration := time.Hour * 24 * 31 // 31 days
	vip := &model.Vip{
		SubscriptionID: event.SubscriptionID,
		ServerID:       event.ServerID,
		SteamID:        event.SteamID,
		ExpiresAt:      time.Now().Add(duration),
	}

	if err := h.store.Create(ctx, vip); err != nil {
		return err
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
