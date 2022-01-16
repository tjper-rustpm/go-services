package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func NewZapLogFormatter(logger *zap.Logger) *ZapLogFormatter {
	return &ZapLogFormatter{
		logger: logger,
	}
}

const kongRequestID = "KONG_REQUEST_ID"

type ZapLogFormatter struct{ logger *zap.Logger }

func (f ZapLogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	return logEntry{
		logger:    f.logger,
		method:    r.Method,
		uri:       r.RequestURI,
		requestID: r.Header.Get(kongRequestID),
	}
}

type logEntry struct {
	logger    *zap.Logger
	method    string
	uri       string
	requestID string
}

func (e logEntry) Write(
	status, bytes int,
	header http.Header,
	elapsed time.Duration,
	_ interface{},
) {
	e.logger.Debug(
		"[HTTP Request]",
		zap.String(kongRequestID, e.requestID),
		zap.String("method", e.method),
		zap.String("uri", e.uri),
		zap.Int("status", status),
		zap.Int("bytes", bytes),
		zap.Duration("elapsed", elapsed),
	)
}

func (e logEntry) Panic(v interface{}, stack []byte) {
	e.logger.Sugar().Panic(
		"v", v,
		"stack", stack,
	)
}
