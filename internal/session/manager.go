package session

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/vmihailenco/msgpack/v5"
)

const (
	prefix = "rustpm-session-"
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
func (m Manager) Create(ctx context.Context, sess Session) error {
	val, err := encode(sess)
	if err != nil {
		return err
	}
	set, err := m.redis.SetNX(ctx, key(sess.ID), val, 0).Result()
	if err != nil {
		return err
	}
	if !set {
		return ErrSessionIDNotUnique
	}
	return nil
}

// Retrieve gets the Session related to the sessionID passed.
func (m Manager) Retrieve(ctx context.Context, sessionID string) (*Session, error) {
	val, err := m.redis.Get(ctx, key(sessionID)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, ErrSessionDNE
	}
	if err != nil {
		return nil, err
	}
	sess := new(Session)
	return sess, decode([]byte(val), sess)
}

// Save saves the passed Session. This session must already exist. If it does
// not exist, and error will be thrown.
func (m Manager) Save(ctx context.Context, sess Session) error {
	val, err := encode(sess)
	if err != nil {
		return err
	}
	set, err := m.redis.SetXX(
		ctx,
		key(sess.ID),
		val,
		0,
	).Result()
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
	if _, err := m.redis.Del(ctx, key(sessionID)).Result(); err != nil {
		return err
	}
	return nil
}

// --- helpers ---

func key(id string) string {
	return fmt.Sprintf("%s%s", prefix, id)
}

func encode(obj interface{}) ([]byte, error) {
	return msgpack.Marshal(obj)
}

func decode(b []byte, obj interface{}) error {
	return msgpack.Unmarshal(b, obj)
}
