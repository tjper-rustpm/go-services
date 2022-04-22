package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/tjper/rustcron/cmd/payment/model"
	"github.com/tjper/rustcron/internal/gorm"
	ihttp "github.com/tjper/rustcron/internal/http"

	"github.com/google/uuid"
)

type CreateServer struct{ API }

func (ep CreateServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		ID                uuid.UUID `json:"id" validate:"required"`
		SubscriptionLimit int       `json:"subscriptionLimit" validate:"required"`
	}

	var b body
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	if err := ep.valid.Struct(b); err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	server := &model.Server{
		ID:                b.ID,
		SubscriptionLimit: uint16(b.SubscriptionLimit),
	}
	err := ep.store.Create(r.Context(), server)
	if errors.Is(err, gorm.ErrAlreadyExists) {
		ihttp.ErrConflict(w)
		return
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(server); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
	}
}
