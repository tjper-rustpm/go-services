package http

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

func ErrInternal(logger *zap.Logger, w http.ResponseWriter, err error) {
	logger.Error("internal server error", zap.Error(err))
	http.Error(
		w,
		"An unexpected internal server error occurred, please try again. If the issue persists, please contact support",
		http.StatusInternalServerError,
	)
}

func ErrUnauthorized(w http.ResponseWriter) {
	http.Error(
		w,
		"Unauthorized; please sign-in to continue.",
		http.StatusUnauthorized,
	)
}

func ErrForbidden(w http.ResponseWriter) {
	http.Error(
		w,
		"Forbidden; user does not have permission to carry-out this action.",
		http.StatusForbidden,
	)
}

func ErrBadRequest(logger *zap.Logger, w http.ResponseWriter, err error) {
	logger.Warn("bad request", zap.Error(err))

	var valerrors validator.ValidationErrors
	if !errors.As(err, &valerrors) {
		http.Error(
			w,
			"An unknown field is invalid. Please update your request and retry.",
			http.StatusBadRequest,
		)
		return
	}

	errormsgs := make([]string, len(valerrors))
	for i, err := range valerrors {
		errormsgs[i] = fmt.Sprintf("\"%s\" failed \"%s\" validator", err.Field(), err.Tag())
	}

	http.Error(
		w,
		fmt.Sprintf("Field(s) validation failure: %s. Please update your request and retry.", strings.Join(errormsgs, ", ")),
		http.StatusBadRequest,
	)
}

func ErrConflict(w http.ResponseWriter) {
	http.Error(
		w,
		"Conflict occurred carrying out request. If this is unexpected, please contact support.",
		http.StatusConflict,
	)
}

func ErrNotFound(w http.ResponseWriter) {
	http.Error(
		w,
		"Resource not found. If this is unexpected, please contact support.",
		http.StatusNotFound,
	)
}
