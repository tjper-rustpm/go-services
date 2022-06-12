package lock

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const key = "mutex-key"

func TestLock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	redis := &redisMock{
		lock: new(sync.RWMutex),
	}

	lock := NewDistributed(zap.NewNop(), redis, key, 100*time.Millisecond)

	err := lock.Lock(ctx)
	require.Nil(t, err)
	lock.Unlock(ctx)

	require.Equal(t, 1, redis.Acquired())
}

func TestWaitForLock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	redis := &redisMock{
		lock: new(sync.RWMutex),
	}

	first := NewDistributed(zap.NewNop(), redis, key, 100*time.Millisecond)

	err := first.Lock(ctx)
	require.Nil(t, err)
	defer first.Unlock(ctx)

	second := NewDistributed(zap.NewNop(), redis, key, 100*time.Millisecond)

	err = second.Lock(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	require.Equal(t, 1, redis.Acquired())
	require.Equal(t, 3, redis.Attempted())
}

// --- mocks ---

type redisMock struct {
	lock       *sync.RWMutex
	val        interface{}
	expiration time.Time

	attempted  int32
	acquired   int32
	maintained int32
}

func (r *redisMock) SetNX(
	ctx context.Context,
	key string,
	val interface{},
	exp time.Duration,
) (bool, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.attempted++

	if time.Now().UnixNano() > r.expiration.UnixNano() {
		r.val = nil
	}
	if r.val != nil {
		return false, nil
	}

	r.val = val
	r.expiration = time.Now().Add(exp)
	r.acquired++

	return true, nil
}

func (r *redisMock) SetXX(
	ctx context.Context,
	key string,
	val interface{},
	exp time.Duration,
) (bool, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if time.Now().UnixNano() > r.expiration.UnixNano() {
		r.val = nil
	}
	if r.val == nil {
		return false, nil
	}
	r.val = val
	r.expiration = time.Now().Add(exp)

	r.maintained++
	return true, nil
}

func (r *redisMock) Acquired() int {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return int(r.acquired)
}

func (r *redisMock) Attempted() int {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return int(r.attempted)
}
