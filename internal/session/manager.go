package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/vmihailenco/msgpack/v5"
	"go.uber.org/zap"
)

var (
	// ErrSessionIDNotUnique indicates that the session ID used to create a
	// session is already being used by another session.
	ErrSessionIDNotUnique = errors.New("session ID is not unique")

	// ErrSessionDNE indicates that an interaction was attempted against a
	// session that does not exist.
	ErrSessionDNE = errors.New("session does not exist")

	// ErrSessionStale indicates that an interaction was attempted against a
	// session that has been marked as stale and therefore outdated.
	ErrSessionStale = errors.New("session stale")

	// ErrMaxAttemptsReached indicates that an operation was unable to complete
	// despite having been attempted the maximum allowed number of times.
	ErrMaxAttemptsReached = errors.New("maximum attempts reached")
)

func NewManager(
	logger *zap.Logger,
	redis *redis.Client,
	sessionExpiration time.Duration,
) *Manager {
	return &Manager{
		logger:     logger,
		redis:      redis,
		expiration: sessionExpiration,
	}
}

// Manager manages Session interactions.
type Manager struct {
	logger *zap.Logger
	redis  *redis.Client

	expiration time.Duration
}

// CreateSession creates a new Session. This session should not already exist.
// If it does, an error will be thrown.
func (m Manager) CreateSession(
	ctx context.Context,
	sess Session,
) error {
	return m.setnx(ctx, keygen(sessionPrefix, sess.ID), sess, m.expiration)
}

// UpdateSession updated the Session related to the sessionID passed using the
// updateFn. The Session will be updated to equal the state of the updateFn's
// *Session argument.
func (m Manager) UpdateSession(
	ctx context.Context,
	sessionID string,
	updateFn func(*Session),
) (*Session, error) {
	return m.update(ctx, sessionID, updateFn, m.expiration)
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

	ok, err := m.isSessionValid(ctx, sess)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrSessionDNE
	}

	ok, err = m.isSessionFresh(ctx, sess)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &sess, ErrSessionStale
	}

	return &sess, nil
}

// TouchSession updates the LastActivityAt field of the session identified by
// sessionID. This session must already exist. If it does not exist, an error
// will be thrown.
func (m Manager) TouchSession(
	ctx context.Context,
	sessionID string,
) (*Session, error) {
	return m.update(
		ctx,
		sessionID,
		func(sess *Session) { sess.LastActivityAt = time.Now() },
		m.expiration,
	)
}

// DeleteSession deletes the specified Session.
func (m Manager) DeleteSession(ctx context.Context, sess Session) error {
	if _, err := m.redis.Del(ctx, keygen(sessionPrefix, sess.ID)).Result(); err != nil {
		return err
	}

	return nil
}

func (m Manager) InvalidateUserSessionsBefore(
	ctx context.Context,
	userID fmt.Stringer,
	dt time.Time,
) error {
	b, err := encode(dt)
	if err != nil {
		return err
	}
	if _, err := m.redis.Set(
		ctx,
		keygen(invalidateUserSessionsPrefix, userID.String()),
		b,
		0,
	).Result(); err != nil {
		return err
	}

	return nil
}

func (m Manager) MarkStaleUserSessionsBefore(
	ctx context.Context,
	userID fmt.Stringer,
	dt time.Time,
) error {
	b, err := encode(dt)
	if err != nil {
		return err
	}
	if _, err := m.redis.Set(
		ctx,
		keygen(markStaleUserSessionsPrefix, userID.String()),
		b,
		0,
	).Result(); err != nil {
		return err
	}

	return nil
}

func (m Manager) isSessionValid(
	ctx context.Context,
	sess Session,
) (bool, error) {
	var invalidAt time.Time

	res, err := m.redis.Get(
		ctx,
		keygen(invalidateUserSessionsPrefix, sess.User.ID.String()),
	).Result()
	if errors.Is(err, redis.Nil) {
		goto valid
	}
	if err != nil {
		return false, err
	}

	if err := decode([]byte(res), &invalidAt); err != nil {
		return false, err
	}

	if sess.CreatedAt.After(invalidAt) {
		goto valid
	}

	if err := m.DeleteSession(ctx, sess); err != nil {
		return false, err
	}

	return false, nil

valid:
	return true, nil
}

func (m Manager) isSessionFresh(
	ctx context.Context,
	sess Session,
) (bool, error) {
	var staleAt time.Time

	res, err := m.redis.Get(
		ctx,
		keygen(markStaleUserSessionsPrefix, sess.User.ID.String()),
	).Result()
	if errors.Is(err, redis.Nil) {
		goto fresh
	}
	if err != nil {
		return false, err
	}

	if err := decode([]byte(res), &staleAt); err != nil {
		return false, err
	}

	if sess.RefreshedAt.After(staleAt) {
		goto fresh
	}

	return false, nil

fresh:
	return true, nil
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

func (m Manager) update(
	ctx context.Context,
	sessionID string,
	updateFn func(*Session),
	exp time.Duration,
) (*Session, error) {
	const maxRetries = 10

	var updatedSess Session
	optimisticTransaction := func(key string, updateFn func(*Session)) error {
		attempt := func(tx *redis.Tx) error {
			res, err := tx.Get(ctx, key).Result()
			if err != nil {
				return err
			}

			var sess Session
			if err := decode([]byte(res), &sess); err != nil {
				return err
			}

			updateFn(&sess)

			b, err := encode(sess)
			if err != nil {
				return err
			}

			// Operation is committed only if the watched keys remain unchanged.
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				if err := pipe.SetXX(ctx, key, b, exp).Err(); err != nil {
					m.logger.Error("update setxx", zap.Error(err))
				}
				return nil
			})
			if err != nil {
				m.logger.Error("update", zap.Error(err))
			}
			updatedSess = sess
			return err
		}

		for i := 0; i < maxRetries; i++ {
			err := m.redis.Watch(ctx, attempt, key)
			if err == nil {
				// Success.
				return nil
			}
			if errors.Is(err, redis.TxFailedErr) {
				// Optimistic lock lost. Retry.
				continue
			}
			// Return any other error.
			return err
		}

		return ErrMaxAttemptsReached
	}

	err := optimisticTransaction(keygen(sessionPrefix, sessionID), updateFn)
	if errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("touch session; id: %s, error: %w", sessionID, ErrSessionDNE)
	}
	if err != nil {
		return nil, fmt.Errorf("touch session; id: %s, error: %w", sessionID, err)
	}

	return &updatedSess, nil
}

// --- helpers ---

const (
	sessionPrefix                = "rustpm-session-"
	invalidateUserSessionsPrefix = "invalidate-user-sessions-"
	markStaleUserSessionsPrefix  = "stale-user-sessions-"
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
