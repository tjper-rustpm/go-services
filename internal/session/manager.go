package session

import (
	"context"
	"errors"
	"fmt"
	"sync"
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

	// ErrMaxAttemptsReached indicates that an operation was unable to complete
	// despite having been attempted the maximum allowed number of times.
	ErrMaxAttemptsReached = errors.New("maximum attempts reached")
)

func NewManager(redis *redis.Client) *Manager {
	return &Manager{redis: redis}
}

// Manager manages Session interactions.
type Manager struct {
	logger *zap.Logger
	redis  *redis.Client
}

// CreateSession creates a new Session. This session should not already exist.
// If it does, an error will be thrown.
func (m Manager) CreateSession(
	ctx context.Context,
	sess Session,
	exp time.Duration,
) error {
	return m.setnx(ctx, keygen(sessionPrefix, sess.ID), sess, exp)
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

	return &sess, nil
}

// TouchSession updates the LastActivityAt field of the session identified by
// sessionID. This session must already exist. If it does not exist, an error
// will be thrown.
func (m Manager) TouchSession(
	ctx context.Context,
	sess Session,
	exp time.Duration,
) error {
	const maxRetries = 10

	touch := func(key string) error {
		// Transactional function.
		fn := func(tx *redis.Tx) error {
			res, err := tx.Get(ctx, key).Result()
			if err != nil {
				return err
			}

			var sess Session
			if err := decode([]byte(res), &sess); err != nil {
				return err
			}

			sess.LastActivityAt = time.Now()

			// Operation is committed only if the watched keys remain unchanged.
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				return pipe.SetXX(ctx, key, sess, exp).Err()
			})
			return err
		}

		for i := 0; i < maxRetries; i++ {
			err := m.redis.Watch(ctx, fn, key)
			if err == nil {
				// Success.
				return nil
			}
			if err == redis.TxFailedErr {
				// Optimistic lock lost. Retry.
				continue
			}
			// Return any other error.
			return err
		}

		return ErrMaxAttemptsReached
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := touch(keygen(sessionPrefix, sess.ID))
			if errors.Is(err, redis.Nil) {
				return
			}
			if err != nil {
				m.logger.Error("error touching session", zap.Error(err))
			}
		}()
	}
	wg.Wait()

	return nil
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
