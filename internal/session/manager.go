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

// Creates creates a new Session. This session should not already exist. If
// it does, an error will be thrown.
func (m Manager) Create(
	ctx context.Context,
	sess Session,
	exp time.Duration,
) error {
	if err := m.setnx(ctx, keygen(sessionPrefix, sess.ID), sess, exp); err != nil {
		return err
	}

	return m.setnx(ctx, keygen(lastActivityAtPrefix, sess.ID), sess.LastActivityAt, exp)
}

// Retrieve gets the Session related to the sessionID passed.
func (m Manager) Retrieve(
	ctx context.Context,
	sessionID string,
) (*Session, error) {
	var sess Session
	if err := m.get(ctx, keygen(sessionPrefix, sessionID), &sess); err != nil {
		return nil, err
	}

	var lastActivityAt time.Time
	if err := m.get(ctx, keygen(lastActivityAtPrefix, sessionID), &lastActivityAt); err != nil {
		return nil, err
	}

	sess.LastActivityAt = lastActivityAt

	return &sess, nil
}

// Touch updates the LastActivityAt field of the session identified by
// sessionID. This session must already exist. If it does not exist, an error
// will be thrown.
func (m Manager) Touch(
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

// Delete deletes the specified Session.
func (m Manager) Delete(ctx context.Context, sessionID string) error {
	if _, err := m.redis.Del(ctx, keygen(sessionPrefix, sessionID)).Result(); err != nil {
		return err
	}

	if _, err := m.redis.Del(ctx, keygen(lastActivityAtPrefix, sessionID)).Result(); err != nil {
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
	sessionPrefix        = "rustpm-session-"
	lastActivityAtPrefix = "last-activity-at-"
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
