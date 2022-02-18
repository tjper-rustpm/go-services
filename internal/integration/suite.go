package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/rand"
	"github.com/tjper/rustcron/internal/session"
	"github.com/tjper/rustcron/internal/stream"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func InitSuite(
	ctx context.Context,
	t *testing.T,
	options ...Option,
) *Suite {
	t.Helper()

	const (
		redisAddr     = "redis:6379"
		redisPassword = ""
	)

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
	})
	err := rdb.Ping(ctx).Err()
	require.Nil(t, err)

	err = rdb.FlushDB(ctx).Err()
	require.Nil(t, err)

	sessionManager := session.NewMock(time.Hour)

	stream, err := stream.Init(ctx, rdb, "test")
	require.Nil(t, err)

	s := &Suite{
		Logger:   zap.NewNop(),
		Redis:    rdb,
		Stream:   stream,
		Sessions: sessionManager,
	}

	for _, option := range options {
		option(s)
	}

	return s
}

type Option func(*Suite)

func WithLogger(logger *zap.Logger) Option {
	return func(s *Suite) { s.Logger = logger }
}

type Suite struct {
	Logger   *zap.Logger
	Redis    *redis.Client
	Stream   *stream.Client
	Sessions *session.Mock
}

func (s Suite) NewSession(ctx context.Context, t *testing.T, u session.User) *session.Session {
	t.Helper()

	id, err := rand.GenerateString(16)
	require.Nil(t, err)

	sess := session.New(id, u, time.Minute)

	err = s.Sessions.CreateSession(ctx, *sess)
	require.Nil(t, err)

	return sess
}

func (s Suite) Request(
	ctx context.Context,
	t *testing.T,
	handler http.Handler,
	method string,
	target string,
	body interface{},
	sess ...*session.Session,
) *http.Response {
	t.Helper()

	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, target, nil)
	} else {
		buf := new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		assert.Nil(t, err)

		req = httptest.NewRequest(method, target, buf)
	}

	req = req.WithContext(ctx)

	if l := len(sess); l == 1 {
		req.AddCookie(ihttp.Cookie(sess[0].ID, ihttp.CookieOptions{}))
	} else if l > 1 {
		t.Fatalf("Suite.Request only accepts zero or one session")
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	return rr.Result()
}
