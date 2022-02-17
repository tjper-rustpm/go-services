package rest

import (
	"encoding/json"
	"net/http"

	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"
)

type Subscriptions struct{ API }

func (ep Subscriptions) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sess, ok := session.FromContext(r.Context())
	if !ok {
		ihttp.ErrUnauthorized(w)
		return
	}

	modelSubscriptions, err := ep.ctrl.UserSubscriptions(
		r.Context(),
		sess.User.SubscriptionIDs(),
	)
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	w.WriteHeader(http.StatusOK)

	subscriptions := SubscriptionsFromModel(modelSubscriptions)
	if err := json.NewEncoder(w).Encode(subscriptions); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
	}
}
