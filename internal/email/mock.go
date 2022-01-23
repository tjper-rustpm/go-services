package email

import (
	"context"
	"sync"
)

func NewMock() *Mock {
	return &Mock{
		mutex:               new(sync.RWMutex),
		passwordResetEmails: make(map[string]string),
		sendVerifyEmails:    make(map[string]string),
	}
}

type Mock struct {
	mutex               *sync.RWMutex
	passwordResetEmails map[string]string
	sendVerifyEmails    map[string]string
}

func (m *Mock) SendPasswordReset(ctx context.Context, to, hash string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.passwordResetEmails[to] = hash
	return nil
}

func (m Mock) SendVerifyEmail(ctx context.Context, to, hash string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.sendVerifyEmails[to] = hash
	return nil
}
