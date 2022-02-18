package session

import (
	"context"
	"fmt"
	"sync"
	"time"
)

func NewMock(expiration time.Duration) *Mock {
	return &Mock{
		mutex:         new(sync.Mutex),
		sessions:      make(map[string]MockSession),
		invalidations: make(map[string]time.Time),
		expiration:    expiration,
	}
}

type Mock struct {
	mutex         *sync.Mutex
	sessions      map[string]MockSession
	invalidations map[string]time.Time
	expiration    time.Duration
}

type MockSession struct {
	Session
	ExpiresAt time.Time
}

func (m *Mock) CreateSession(_ context.Context, sess Session) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.sessions[keygen(sessionPrefix, sess.ID)]; ok {
		return ErrSessionIDNotUnique
	}

	m.sessions[keygen(sessionPrefix, sess.ID)] = MockSession{
		Session:   sess,
		ExpiresAt: time.Now().Add(m.expiration),
	}
	return nil
}

func (m *Mock) UpdateSession(
	_ context.Context,
	sessionID string,
	updateFn func(*Session),
) (*Session, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	fetched, ok := m.sessions[keygen(sessionPrefix, sessionID)]
	if !ok {
		return nil, ErrSessionDNE
	}

	updateFn(&fetched.Session)

	m.sessions[keygen(sessionPrefix, sessionID)] = fetched

	return &fetched.Session, nil
}

func (m *Mock) RetrieveSession(_ context.Context, id string) (*Session, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	sess, ok := m.sessions[keygen(sessionPrefix, id)]
	if !ok {
		return nil, ErrSessionDNE
	}

	if sess.ExpiresAt.Before(time.Now()) {
		delete(m.sessions, keygen(sessionPrefix, id))
		return nil, ErrSessionDNE
	}

	invalidAt, ok := m.invalidations[keygen(invalidateUserSessionsPrefix, sess.User.ID.String())]
	if !ok {
		return &sess.Session, nil
	}

	if sess.CreatedAt.Before(invalidAt) {
		delete(m.sessions, keygen(sessionPrefix, id))
		return nil, ErrSessionDNE
	}

	return &sess.Session, nil
}

func (m *Mock) TouchSession(ctx context.Context, sessionID string) (*Session, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	fetched, ok := m.sessions[keygen(sessionPrefix, sessionID)]
	if !ok {
		return nil, ErrSessionDNE
	}

	fetched.LastActivityAt = time.Now()
	fetched.ExpiresAt = time.Now().Add(m.expiration)

	return &fetched.Session, nil
}

func (m *Mock) DeleteSession(_ context.Context, sess Session) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.sessions, keygen(sessionPrefix, sess.ID))

	return nil
}

func (m *Mock) InvalidateUserSessionsBefore(
	_ context.Context,
	userID fmt.Stringer,
	dt time.Time,
) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.invalidations[keygen(invalidateUserSessionsPrefix, userID.String())] = dt

	return nil
}

// func (m *Mock) MarkStaleUserSessionsBefore(
// 	ctx context.Context,
// 	userID fmt.Stringer,
// 	dt time.Time,
// ) error {

// }
