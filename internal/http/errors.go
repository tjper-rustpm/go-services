package http

import (
	"fmt"
	"net/http"
)

func ErrInternal(w http.ResponseWriter) {
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

func ErrBadRequest(w http.ResponseWriter, field string) {
	http.Error(
		w,
		fmt.Sprintf("Field \"%s\" is is invalid. Please update your request and retry.", field),
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
