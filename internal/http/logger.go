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

type ZapLogFormatter struct{ logger *zap.Logger }

func (f ZapLogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	return logEntry(f)
}

type logEntry struct{ logger *zap.Logger }

func (e logEntry) Write(
	status, bytes int,
	header http.Header,
	elapsed time.Duration,
	_ interface{},
) {

	var level func(string, ...zap.Field)
	switch {
	case status < http.StatusOK:
		level = e.logger.Debug
	case status < http.StatusMultipleChoices:
		level = e.logger.Debug
	case status < http.StatusBadRequest:
		level = e.logger.Info
	case status < http.StatusInternalServerError:
		level = e.logger.Warn
	default:
		level = e.logger.Error
	}

	level(
		"[HTTP Request]",
		zap.String("KONG_REQUEST_ID", header.Get("KONG_REQUEST_ID")),
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
