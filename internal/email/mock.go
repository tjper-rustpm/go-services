package email

import (
	"context"
	"sync"
)

func NewMock() *Mock {
	return &Mock{
		mutex:               new(sync.RWMutex),
		passwordResetEmails: make(map[string]string),
		verifyEmails:        make(map[string]string),
	}
}

type Mock struct {
	mutex               *sync.RWMutex
	passwordResetEmails map[string]string
	verifyEmails        map[string]string
}

func (m *Mock) SendPasswordReset(ctx context.Context, to, hash string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.passwordResetEmails[to] = hash
	return nil
}

func (m *Mock) PasswordResetHash(to string) string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.passwordResetEmails[to]
}

func (m *Mock) SendVerifyEmail(ctx context.Context, to, hash string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.verifyEmails[to] = hash
	return nil
}

func (m *Mock) VerifyEmailHash(to string) string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.verifyEmails[to]
}
