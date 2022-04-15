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

type ServerSubscriptionLimits struct{ API }

func (ep ServerSubscriptionLimits) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		ServerID uuid.UUID `json:"serverId" validate:"required"`
		Maximum  int       `json:"maximum" validate:"required"`
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

	limit := &model.ServerSubscriptionLimit{
		ServerID: b.ServerID,
		Maximum:  uint8(b.Maximum),
	}
	err := ep.store.Create(r.Context(), limit)
	if errors.Is(err, gorm.ErrAlreadyExists) {
		ihttp.ErrConflict(w)
		return
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(limit); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
	}
}
