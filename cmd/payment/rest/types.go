package rest

import (
	"time"

	"github.com/google/uuid"
	"github.com/tjper/rustcron/cmd/payment/model"
)

type Subscription struct {
	ID        uuid.UUID           `json:"id"`
	Status    model.InvoiceStatus `json:"status"`
	CreatedAt time.Time           `json:"createdAt"`
}

func SubscriptionsFromModel(modelSubscriptions []model.Subscription) []Subscription {
	subscriptions := make([]Subscription, 0, len(modelSubscriptions))
	for _, sub := range modelSubscriptions {
		subscriptions = append(
			subscriptions,
			Subscription{
				ID:        sub.ID,
				Status:    sub.Status(),
				CreatedAt: sub.CreatedAt,
			},
		)
	}

	return subscriptions
}
