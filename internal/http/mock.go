package http

import (
	"net/http"
	"testing"

	"github.com/tjper/rustcron/internal/session"
)

// NewSessionMiddlewareMock creates a new SessionMiddlewareMock instance.
func NewSessionMiddlewareMock(options ...SessionMiddlewareMockOption) *SessionMiddlewareMock {
	mock := &SessionMiddlewareMock{}

	for _, option := range options {
		option(mock)
	}

	return mock
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
func WithHasRole(middleware func(session.Role) func(http.Handler) http.Handler) SessionMiddlewareMockOption {
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
	hasRole              func(session.Role) func(http.Handler) http.Handler
	isAuthenticated      func(http.Handler) http.Handler
}

// InjectSessionIntoCtx returns the http middleware set with
// WithInjectSessionIntoCtx.
func (mock SessionMiddlewareMock) InjectSessionIntoCtx() func(http.Handler) http.Handler {
	if mock.injectSessionIntoCtx == nil {
		return unconfiguredMiddleware
	}
	return mock.injectSessionIntoCtx
}

// Touch returns the http middleware set with WithTouch.
func (mock SessionMiddlewareMock) Touch() func(http.Handler) http.Handler {
	if mock.touch == nil {
		return unconfiguredMiddleware
	}
	return mock.touch
}

// HasRole returns the http middleware set with WithHasRole.
func (mock SessionMiddlewareMock) HasRole(role session.Role) func(http.Handler) http.Handler {
	if mock.hasRole == nil {
		return unconfiguredMiddleware
	}
	return mock.hasRole(role)
}

// IsAuthenticated returns the http middleware set with WithIsAuthenticated.
func (mock SessionMiddlewareMock) IsAuthenticated() func(http.Handler) http.Handler {
	if mock.isAuthenticated == nil {
		return unconfiguredMiddleware
	}
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

// ExpectMiddlewareCalled is used to check if the middleware is called. The
// called channel will be closed when the middleware has executed.
func ExpectMiddlewareCalled() (middleware func(http.Handler) http.Handler, called chan struct{}) {
	called = make(chan struct{})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				close(called)
				next.ServeHTTP(w, r)
			})
	}, called
}

// ExpectRoleMiddleware is used to check if the expected role is passed the
// HasRole middleware. The returned called channel will be closed when the
// middleware has executed.
func ExpectRoleMiddleware(
	t *testing.T,
	expected session.Role,
) (hasRole func(session.Role) func(http.Handler) http.Handler, called chan struct{}) {
	called = make(chan struct{})

	return func(role session.Role) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					close(called)

					if expected != role {
						t.Fatalf("expected role (%s) does not match actual role (%s)", expected, role)
					}
					next.ServeHTTP(w, r)
				})
		}
	}, called
}

// SkipHasRoleMiddleware is used to skip role middlware checks in tests where a
// user's role is not critical to the behavior being tested.
func SkipHasRoleMiddleware(_ session.Role) func(http.Handler) http.Handler {
	return SkipMiddleware
}

// unconfiguredMiddleware indicates that a middleware was called without being
// explicitly mocked via a SessionMiddlewareMockOption.
func unconfiguredMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			panic("unconfigured mock middleware call")
		})
}
