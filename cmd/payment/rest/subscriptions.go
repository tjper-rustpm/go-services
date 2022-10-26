package rest

import (
	"encoding/json"
	"net/http"

	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"
)

type SubscriptionsEndpoint struct{ API }

func (ep SubscriptionsEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sess, ok := session.FromContext(r.Context())
	if !ok {
		ihttp.ErrUnauthorized(w)
		return
	}

	modelSubscriptions, err := ep.store.FindSubscriptionsByUserID(r.Context(), sess.User.ID)
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
	}

	w.WriteHeader(http.StatusOK)

	var subscriptions Subscriptions
	subscriptions.FromModelSubscriptions(modelSubscriptions)

	if err := json.NewEncoder(w).Encode(subscriptions); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
	}
}
