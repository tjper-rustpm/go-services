package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	cronmanerrors "github.com/tjper/rustcron/cmd/cronman/errors"
	ihttp "github.com/tjper/rustcron/internal/http"
)

type AddServerTags struct{ API }

func (ep AddServerTags) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var b AddServerTagsBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	if err := ep.valid.Struct(b); err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	modelTags := b.Tags.ToModelTags()

	err := ep.ctrl.AddServerTags(r.Context(), b.ServerID, modelTags)
	if errors.Is(err, cronmanerrors.ErrServerDNE) {
		ihttp.ErrNotFound(w)
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)

	tags := TagsFromModel(modelTags)

	if err := json.NewEncoder(w).Encode(tags); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}
}

type RemoveServerTags struct{ API }

func (ep RemoveServerTags) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var b RemoveServerTagsBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	if err := ep.valid.Struct(b); err != nil {
		ihttp.ErrBadRequest(ep.logger, w, err)
		return
	}

	err := ep.ctrl.RemoveServerTags(r.Context(), b.ServerID, b.TagIDs)
	if errors.Is(err, cronmanerrors.ErrServerDNE) {
		ihttp.ErrNotFound(w)
	}
	if err != nil {
		ihttp.ErrInternal(ep.logger, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
