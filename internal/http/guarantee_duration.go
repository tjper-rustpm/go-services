package http

import (
	"net/http"
	"time"
)

func EnsureDuration(min time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				dw := durationWriter{
					ResponseWriter: w,
					end:            time.Now().Add(min),
				}

				next.ServeHTTP(dw, r)
			},
		)
	}
}

type durationWriter struct {
	http.ResponseWriter

	end time.Time
}

func (w durationWriter) WriteHeader(statusCode int) {
	if remaining := time.Until(w.end); remaining > 0 {
		time.Sleep(remaining)
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w durationWriter) Write(b []byte) (int, error) {
	if remaining := time.Until(w.end); remaining > 0 {
		time.Sleep(remaining)
	}
	return w.ResponseWriter.Write(b)
}
