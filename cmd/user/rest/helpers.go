package rest

import (
	"context"
	"encoding/json"
	http "net/http"

	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"
)

func (api API) read(w http.ResponseWriter, req *http.Request, i interface{}) error {
	if err := json.NewDecoder(req.Body).Decode(i); err != nil {
		ihttp.ErrInternal(w)
		return err
	}
	return nil
}

func (api API) write(w http.ResponseWriter, code int, i interface{}) {
	w.WriteHeader(code)
	if i == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(i); err != nil {
		ihttp.ErrInternal(w)
	}
}

func (api API) session(ctx context.Context, w http.ResponseWriter) (*session.Session, bool) {
	sess, ok := session.FromContext(ctx)
	if !ok {
		ihttp.ErrUnauthorized(w)
	}
	return sess, ok
}
