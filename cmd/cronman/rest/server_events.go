package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	cronmanerrors "github.com/tjper/rustcron/cmd/cronman/errors"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type AddServerEvents struct{ API }

func (ep AddServerEvents) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var b AddServerEventsBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	if err := ep.valid.Struct(b); err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	modelEvents := b.Events.ToModelEvents()

	err := ep.ctrl.AddServerEvents(r.Context(), b.ServerID, modelEvents)
	if errors.Is(err, cronmanerrors.ErrServerDNE) {
		ihttp.ErrNotFound(w)
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)

	events := EventsFromModel(modelEvents)

	if err := json.NewEncoder(w).Encode(events); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}
}

type RemoveServerEvents struct{ API }

func (ep RemoveServerEvents) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var b RemoveServerEventsBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	if err := ep.valid.Struct(b); err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	err := ep.ctrl.RemoveServerEvents(r.Context(), b.ServerID, b.EventIDs)
	if errors.Is(err, cronmanerrors.ErrServerDNE) {
		ihttp.ErrNotFound(w)
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
