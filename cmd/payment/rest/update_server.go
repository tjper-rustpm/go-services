package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/tjper/rustcron/internal/gorm"
	ihttp "github.com/tjper/rustcron/internal/http"

	"github.com/google/uuid"
)

type UpdateServer struct{ API }

func (ep UpdateServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type body struct {
		ID      uuid.UUID              `json:"id" validate:"required"`
		Changes map[string]interface{} `json:"changes" validate:"required,dive,keys,eq=subscriptionLimit"`
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

	server, err := ep.store.UpdateServer(r.Context(), b.ID, b.Changes)
	if errors.Is(err, gorm.ErrNotFound) {
		ihttp.ErrNotFound(w)
		return
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(&server); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}
}
