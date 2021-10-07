package graph

import "errors"

var (
	ErrInternalServer = errors.New("internal server error; please contact support")
	ErrInvalidUUID    = errors.New("invalid UUID; please contact support")

	ErrUnauthenticated    = errors.New("unauthenticated")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrSessionDNE         = errors.New("user does not have an active session")
	ErrBadArgument        = errors.New("bad argument")
	ErrResetWindowExpired = errors.New("reset window expired")

	ErrActivationLinkSent = errors.New("a link to activate your account has been sent to address provided")
	ErrInvalidLogin       = errors.New("login failed; invalid user ID or password")
)
