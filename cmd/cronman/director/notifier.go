package director

import (
	"context"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// NewNotifier creates a new Notifier object.
func NewNotifier(logger *zap.Logger, redis *redis.Client) *Notifier {
	return &Notifier{
		logger: logger,
		redis:  redis,
	}
}

// Notifier is responsible for notifying the Director it should reset and
// re-evaluate the event schedule.
type Notifier struct {
	logger *zap.Logger
	redis  *redis.Client
}

// Notify notifies the acting director that there has been a schedule cahange
// and the upcoming events should be re-evaluated.
func (n Notifier) Notify(ctx context.Context) error {
	n.logger.Info("notifying director")
	return n.redis.Publish(ctx, refreshSubj, "notify").Err()
}
