package stream

import (
	"context"
	"fmt"
	"time"

	"github.com/tjper/rustcron/cmd/user/model"
	"github.com/tjper/rustcron/internal/event"
	"github.com/tjper/rustcron/internal/stream"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type IStream interface {
	Read(context.Context) (*stream.Message, error)
}

type ISessionManager interface {
	MarkStaleUserSessionsBefore(context.Context, fmt.Stringer, time.Time) error
}

func NewHandler(
	logger *zap.Logger,
	stream IStream,
	gorm *gorm.DB,
	sessionManager ISessionManager,
) *Handler {
	return &Handler{
		logger:         logger,
		gorm:           gorm,
		sessionManager: sessionManager,
		stream:         stream,
	}
}

type Handler struct {
	logger         *zap.Logger
	gorm           *gorm.DB
	sessionManager ISessionManager
	stream         IStream
}

func (h Handler) Launch(ctx context.Context) error {
	for {
		m, err := h.stream.Read(ctx)
		if err != nil {
			return fmt.Errorf("read stream; error: %w", err)
		}

		go func(ctx context.Context, m *stream.Message) {
			eventI, err := event.Parse(m.Payload)
			if err != nil {
				h.logger.Error("parse event hash", zap.Error(err))
				return
			}

			switch e := eventI.(type) {
			case *event.SubscriptionCreatedEvent:
				err = h.handleSubscriptionCreated(ctx, *e)
			case *event.SubscriptionDeleteEvent:
				err = h.handleSubscriptionDelete(ctx, *e)
			default:
				h.logger.Sugar().Debugf("unrecognized event; type: %T", e)
			}
			if err != nil {
				h.logger.Error("handle stream event", zap.Error(err))
				return
			}

			if err := m.Ack(ctx); err != nil {
				h.logger.Error("acknowledge stream event", zap.Error(err))
				return
			}
		}(ctx, m)
	}
}

func (h Handler) handleSubscriptionCreated(
	ctx context.Context,
	e event.SubscriptionCreatedEvent,
) error {
	if err := h.gorm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user model.User
		if res := tx.First(&user, e.UserID); res.Error != nil {
			return res.Error
		}

		subscription := model.Subscription{
			UserID:         user.ID,
			SubscriptionID: e.SubscriptionID,
			ServerID:       e.ServerID,
		}
		if res := tx.Create(&subscription); res.Error != nil {
			return res.Error
		}
		return nil
	}); err != nil {
		return fmt.Errorf(
			"create user subscription; user-id: %s, subscription-id: %s, error: %w",
			e.UserID,
			e.SubscriptionID,
			err,
		)
	}

	if err := h.sessionManager.MarkStaleUserSessionsBefore(ctx, e.UserID, time.Now()); err != nil {
		return fmt.Errorf("mark user sessions stale; user-id: %s, error: %w", e.UserID, err)
	}
	return nil
}

func (h Handler) handleSubscriptionDelete(
	ctx context.Context,
	e event.SubscriptionDeleteEvent,
) error {
	var subscription model.Subscription
	if err := h.gorm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if res := tx.
			Where("subscription_id = ?", e.SubscriptionID).
			First(&subscription); res.Error != nil {
			return res.Error
		}

		if res := tx.Delete(&subscription, subscription.ID); res.Error != nil {
			return res.Error
		}

		return nil
	}); err != nil {
		return fmt.Errorf("delete user subscription; subscription-id: %s, error: %w", e.SubscriptionID, err)
	}

	if err := h.sessionManager.MarkStaleUserSessionsBefore(ctx, subscription.UserID, time.Now()); err != nil {
		return fmt.Errorf("mark user sessions stale; user-id: %s, error: %w", subscription.UserID, err)
	}
	return nil
}
