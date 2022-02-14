package stream

import (
	"context"
	"fmt"
	"time"

	"github.com/tjper/rustcron/cmd/user/model"
	"github.com/tjper/rustcron/internal/event"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type IStream interface {
	Read(context.Context) (map[string]interface{}, error)
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
		kv, err := h.stream.Read(ctx)
		if err != nil {
			return fmt.Errorf("read stream; error: %w", err)
		}

		eventI, err := event.ParseHash(kv)
		if err != nil {
			h.logger.Error("parse event hash", zap.Error(err))
		}

		switch e := eventI.(type) {
		case event.SubscriptionCreatedEvent:
			go h.handleSubscriptionCreated(ctx, e)
		case event.SubscriptionDeleteEvent:
			go h.handleSubscriptionDelete(ctx, e)
		}
	}
}

func (h Handler) handleSubscriptionCreated(
	ctx context.Context,
	e event.SubscriptionCreatedEvent,
) {
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
		h.logger.Error("create user subscription", zap.Error(err))
		return
	}

	if err := h.sessionManager.MarkStaleUserSessionsBefore(ctx, e.UserID, time.Now()); err != nil {
		h.logger.Error(
			"mark user sessions stale",
			zap.Stringer("user-id", e.UserID),
			zap.Error(err),
		)
		return
	}
}

func (h Handler) handleSubscriptionDelete(
	ctx context.Context,
	e event.SubscriptionDeleteEvent,
) {
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
		h.logger.Error("delete user subscription", zap.Error(err))
		return
	}

	if err := h.sessionManager.MarkStaleUserSessionsBefore(ctx, subscription.UserID, time.Now()); err != nil {
		h.logger.Error(
			"mark user sessions stale",
			zap.Stringer("user-id", subscription.UserID),
			zap.Error(err),
		)
		return
	}
}
