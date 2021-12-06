package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/vmihailenco/msgpack/v5"
)

var (
	// ErrSessionIDNotUnique indicates that the session ID used to create a
	// session is already being used by another session.
	ErrSessionIDNotUnique = errors.New("session ID is not unique")

	// ErrSessionDNE indicates that an interaction was attempted against a
	// session that does not exist.
	ErrSessionDNE = errors.New("session does not exist")
)

func NewManager(redis *redis.Client) *Manager {
	return &Manager{redis: redis}
}

// Manager manages Session interactions.
type Manager struct {
	redis *redis.Client
}

// CreateSession creates a new Session. This session should not already exist.
// If it does, an error will be thrown.
func (m Manager) CreateSession(
	ctx context.Context,
	sess Session,
	exp time.Duration,
) error {
	if err := m.setnx(ctx, keygen(sessionPrefix, sess.ID), sess, exp); err != nil {
		return err
	}

	return m.setnx(ctx, keygen(lastActivityAtPrefix, sess.ID), sess.LastActivityAt, exp)
}

// RetrieveSession gets the Session related to the sessionID passed.
func (m Manager) RetrieveSession(
	ctx context.Context,
	sessionID string,
) (*Session, error) {
	var sess Session
	if err := m.get(ctx, keygen(sessionPrefix, sessionID), &sess); err != nil {
		return nil, err
	}

	res, err := m.redis.Get(ctx, keygen(invalidateUserSessionsPrefix, sessionID)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	var invalidAt time.Time
	if err := decode([]byte(res), &invalidAt); err != nil {
		return nil, err
	}

	if sess.CreatedAt.Before(invalidAt) {
		return nil, m.DeleteSession(ctx, sess)
	}

	var lastActivityAt time.Time
	if err := m.get(ctx, keygen(lastActivityAtPrefix, sessionID), &lastActivityAt); err != nil {
		return nil, err
	}

	sess.LastActivityAt = lastActivityAt

	return &sess, nil
}

// TouchSession updates the LastActivityAt field of the session identified by
// sessionID. This session must already exist. If it does not exist, an error
// will be thrown.
func (m Manager) TouchSession(
	ctx context.Context,
	sessionID string,
	exp time.Duration,
) error {
	if err := m.setxx(
		ctx,
		keygen(lastActivityAtPrefix, sessionID),
		time.Now(),
		exp,
	); err != nil {
		return err
	}

	set, err := m.redis.Expire(ctx, keygen(sessionPrefix, sessionID), exp).Result()
	if err != nil {
		return err
	}
	if !set {
		return ErrSessionDNE
	}

	return nil
}

// DeleteSession deletes the specified Session.
func (m Manager) DeleteSession(ctx context.Context, sess Session) error {
	if _, err := m.redis.Del(ctx, keygen(sessionPrefix, sess.ID)).Result(); err != nil {
		return err
	}

	if _, err := m.redis.Del(ctx, keygen(lastActivityAtPrefix, sess.ID)).Result(); err != nil {
		return err
	}

	return nil
}

func (m Manager) InvalidateUserSessionsBefore(
	ctx context.Context,
	userID fmt.Stringer,
	dt time.Time,
) error {
	if _, err := m.redis.Set(
		ctx,
		keygen(invalidateUserSessionsPrefix, userID.String()),
		time.Now(),
		0,
	).Result(); err != nil {
		return err
	}

	return nil
}

func (m Manager) setxx(
	ctx context.Context,
	key string,
	val interface{},
	exp time.Duration,
) error {
	b, err := encode(val)
	if err != nil {
		return err
	}

	set, err := m.redis.SetXX(ctx, key, b, exp).Result()
	if err != nil {
		return err
	}
	if !set {
		return ErrSessionDNE
	}

	return nil
}

func (m Manager) setnx(
	ctx context.Context,
	key string,
	val interface{},
	exp time.Duration,
) error {
	b, err := encode(val)
	if err != nil {
		return err
	}

	set, err := m.redis.SetNX(ctx, key, b, exp).Result()
	if err != nil {
		return err
	}
	if !set {
		return ErrSessionIDNotUnique
	}
	return nil
}

func (m Manager) get(ctx context.Context, key string, dst interface{}) error {
	res, err := m.redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return ErrSessionDNE
	}
	if err != nil {
		return err
	}

	return decode([]byte(res), dst)
}

// --- helpers ---

const (
	sessionPrefix                = "rustpm-session-"
	lastActivityAtPrefix         = "last-activity-at-"
	invalidateUserSessionsPrefix = "invalidate-user-sessions-"
)

func keygen(prefix, id string) string {
	return fmt.Sprintf("%s%s", prefix, id)
}

func encode(obj interface{}) ([]byte, error) {
	return msgpack.Marshal(obj)
}

func decode(b []byte, obj interface{}) error {
	return msgpack.Unmarshal(b, obj)
}
