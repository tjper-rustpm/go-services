package rest

import (
	"time"

	"github.com/google/uuid"
	"github.com/tjper/rustcron/cmd/payment/model"
)

// Subscription is a re-occurring payment.
type Subscription struct {
	// ID is the Subscription's unique identifier.
	ID uuid.UUID `json:"id"`
	// ServerID is the ID of server the subscription is tied to.
	ServerID uuid.UUID `json:"serverId"`
	// Status is the current status of the subscription. See the type for
	// possible values and their explanation.
	Status model.InvoiceStatus `json:"status"`
	// CreatedAt is when the subscription was created.
	CreatedAt time.Time `json:"createdAt"`
}

// Subscriptions is a slice of Subscription instances.
type Subscriptions []Subscription

// FromModelSubscription converts a []model.Subscription into []Subscription.
func (subs *Subscriptions) FromModelSubscriptions(froms []model.Subscription) {
	if subs == nil {
		*subs = make(Subscriptions, 0, len(froms))
	}
	for _, from := range froms {
		*subs = append(
			*subs,
			Subscription{
				ID:        from.ID,
				ServerID:  from.ServerID,
				Status:    from.Status(),
				CreatedAt: from.CreatedAt,
			},
		)
	}
}

// Redirect contains a URL that should be redirected to by the client. This is
// used instead of http.Redirect as not all client software is capable of
// following a http.Redirect.
type Redirect struct {
	URL string `json:"url"`
}
