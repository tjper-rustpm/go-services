package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	cronmanerrors "github.com/tjper/rustcron/cmd/cronman/errors"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type AddServerOwners struct{ API }

func (ep AddServerOwners) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var b AddServerOwnersBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	if err := ep.valid.Struct(b); err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	modelOwners := b.Owners.ToModelOwners()

	err := ep.ctrl.AddServerOwners(r.Context(), b.ServerID, modelOwners)
	if errors.Is(err, cronmanerrors.ErrServerDNE) {
		ihttp.ErrNotFound(w)
		return
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)

	owners := OwnersFromModel(modelOwners)

	if err := json.NewEncoder(w).Encode(owners); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}
}

type RemoveServerOwners struct{ API }

func (ep RemoveServerOwners) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var b RemoveServerOwnersBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	if err := ep.valid.Struct(b); err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	err := ep.ctrl.RemoveServerOwners(r.Context(), b.ServerID, b.OwnerIDs)
	if errors.Is(err, cronmanerrors.ErrServerDNE) {
		ihttp.ErrNotFound(w)
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
