package rest

import (
	"encoding/json"
	"net/http"

	"github.com/tjper/rustcron/cmd/payment/model"
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

	var subs model.Subscriptions
	if err := ep.store.FindByUserID(r.Context(), &subs, sess.User.ID); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
	}

	w.WriteHeader(http.StatusOK)

	subscriptions := SubscriptionsFromModel(subs)
	if err := json.NewEncoder(w).Encode(subscriptions); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
	}
}
