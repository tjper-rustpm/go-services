package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	cronmanerrors "github.com/tjper/rustcron/cmd/cronman/errors"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type AddServerModerators struct{ API }

func (ep AddServerModerators) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var b AddServerModeratorsBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	if err := ep.valid.Struct(b); err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	modelModerators := b.Moderators.ToModelModerators()

	err := ep.ctrl.AddServerModerators(r.Context(), b.ServerID, modelModerators)
	if errors.Is(err, cronmanerrors.ErrServerDNE) {
		ihttp.ErrNotFound(w)
		return
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)

	moderators := ModeratorsFromModel(modelModerators)

	if err := json.NewEncoder(w).Encode(moderators); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}
}

type RemoveServerModerators struct{ API }

func (ep RemoveServerModerators) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var b RemoveServerModeratorsBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	if err := ep.valid.Struct(b); err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	err := ep.ctrl.RemoveServerModerators(r.Context(), b.ServerID, b.ModeratorIDs)
	if errors.Is(err, cronmanerrors.ErrServerDNE) {
		ihttp.ErrNotFound(w)
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
