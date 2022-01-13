package errors

import "errors"

var (
	ErrServerDNE         = errors.New("server does not exist")
	ErrServerNotArchived = errors.New("server is not archived")
	ErrServerNotDormant  = errors.New("server is not dormant")
	ErrServerNotLive     = errors.New("server is not live")
)
