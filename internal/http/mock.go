package http

import (
	"net/http"

	"github.com/tjper/rustcron/internal/session"
)

// NewSessionMiddlewareMock creates a new SessionMiddlewareMock instance.
func NewSessionMiddlewareMock(options ...SessionMiddlewareMockOption) *SessionMiddlewareMock {
	mock := SessionMiddlewareMock{
		injectSessionIntoCtx: misconfiguredMockMiddleware,
		touch:                misconfiguredMockMiddleware,
		hasRole:              misconfiguredMockMiddleware,
		isAuthenticated:      misconfiguredMockMiddleware,
	}

	for _, option := range options {
		option(&mock)
	}

	return &mock
}

// SessionMiddlewareMockOption is a function type that should configure the
// SessionMiddlewareMock instance.
type SessionMiddlewareMockOption func(*SessionMiddlewareMock)

// WithInjectSessionIntoCtx provides a SessionMiddlewareMockOption that
// configures a SessionMiddlewareMockOption to utilize the passed middleware.
func WithInjectSessionIntoCtx(middleware func(http.Handler) http.Handler) SessionMiddlewareMockOption {
	return func(mock *SessionMiddlewareMock) {
		mock.injectSessionIntoCtx = middleware
	}
}

// WithTouch provides a SessionMiddlewareMockOption that configures a
// SessionMiddlewareMockOption to utilize the passed middleware.
func WithTouch(middleware func(http.Handler) http.Handler) SessionMiddlewareMockOption {
	return func(mock *SessionMiddlewareMock) {
		mock.touch = middleware
	}
}

// WithHasRole provides a SessionMiddlewareMockOption that configures a
// SessionMiddlewareMockOption to utilize the passed middleware.
func WithHasRole(middleware func(http.Handler) http.Handler) SessionMiddlewareMockOption {
	return func(mock *SessionMiddlewareMock) {
		mock.hasRole = middleware
	}
}

// WithIsAuthenticated provides a SessionMiddlewareMockOption that configures a
// SessionMiddlewareMockOption to utilize the passed middleware.
func WithIsAuthenticated(middleware func(http.Handler) http.Handler) SessionMiddlewareMockOption {
	return func(mock *SessionMiddlewareMock) {
		mock.isAuthenticated = middleware
	}
}

// SessionMiddlewareMock provides an adaptable SessionMiddleware mock,
// typically used for testing.
type SessionMiddlewareMock struct {
	injectSessionIntoCtx func(http.Handler) http.Handler
	touch                func(http.Handler) http.Handler
	hasRole              func(http.Handler) http.Handler
	isAuthenticated      func(http.Handler) http.Handler
}

// InjectSessionIntoCtx returns the http middleware set with
// WithInjectSessionIntoCtx.
func (mock SessionMiddlewareMock) InjectSessionIntoCtx() func(http.Handler) http.Handler {
	return mock.injectSessionIntoCtx
}

// Touch returns the http middleware set with WithTouch.
func (mock SessionMiddlewareMock) Touch() func(http.Handler) http.Handler {
	return mock.touch
}

// HasRole returns the http middleware set with WithHasRole.
func (mock SessionMiddlewareMock) HasRole(role session.Role) func(http.Handler) http.Handler {
	return mock.hasRole
}

// IsAuthenticated returns the http middleware set with WithIsAuthenticated.
func (mock SessionMiddlewareMock) IsAuthenticated() func(http.Handler) http.Handler {
	return mock.isAuthenticated
}

// SkipMiddleware is a middleware that skips to the next middleware. Typically
// used in unit-testing to bypass a middleware as it is not critical to the
// test being performed.
func SkipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
}

func misconfiguredMockMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	)
}
