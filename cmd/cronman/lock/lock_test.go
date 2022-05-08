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
	type expected struct {
		attempted  int32
		acquired   int32
		maintained int32
	}
	tests := map[string]struct {
		timeout    time.Duration
		expiration time.Duration
		processes  int
		exp        expected
	}{
		"1 process": {
			timeout:    time.Second,
			expiration: 90 * time.Millisecond,
			processes:  1,
			exp: expected{
				attempted:  1,
				acquired:   1,
				maintained: 21,
			},
		},
		"2 processes": {
			timeout:    time.Second,
			expiration: 90 * time.Millisecond,
			processes:  2,
			exp: expected{
				attempted:  23,
				acquired:   1,
				maintained: 21,
			},
		},
		"10 processes": {
			timeout:    time.Second,
			expiration: 90 * time.Millisecond,
			processes:  10,
			exp: expected{
				attempted:  199,
				acquired:   1,
				maintained: 21,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), test.timeout)
			defer cancel()

			redis := &redisMock{
				lock: new(sync.RWMutex),
			}

			locks := make([]*Distributed, 0, test.processes)
			for i := 0; i < test.processes; i++ {
				locks = append(locks, NewDistributed(zap.NewNop(), redis, key, test.expiration))
			}

			var wg sync.WaitGroup
			for _, lock := range locks {
				wg.Add(1)
				go func(lock *Distributed) {
					defer wg.Done()
					err := lock.Lock(ctx)
					if err != nil {
						require.Equal(t, context.DeadlineExceeded, err)
					}
					if err == nil {
						defer lock.Unlock(ctx)
					}
					<-ctx.Done()
					require.Equal(t, context.DeadlineExceeded, ctx.Err())
				}(lock)
			}
			wg.Wait()

			redis.lock.RLock()
			require.Equal(t, test.exp.attempted, redis.attempted, "attempted")
			require.Equal(t, test.exp.acquired, redis.acquired, "acquired")
			require.Equal(t, test.exp.maintained, redis.maintained, "maintained")
			redis.lock.RUnlock()
		})
	}
}

func TestUnlock(t *testing.T) {
	type expected struct {
		acquired int32
	}
	tests := map[string]struct {
		timeout    time.Duration
		expiration time.Duration
		locks      int
		wait       time.Duration
		exp        expected
	}{
		"1 lock": {
			timeout:    time.Second,
			expiration: 90 * time.Millisecond,
			locks:      2,
			wait:       200 * time.Millisecond,
			exp: expected{
				acquired: 2,
			},
		},
		"2 locks": {
			timeout:    time.Second,
			expiration: 90 * time.Millisecond,
			locks:      3,
			wait:       200 * time.Millisecond,
			exp: expected{
				acquired: 3,
			},
		},
		"10 locks": {
			timeout:    5 * time.Second,
			expiration: 90 * time.Millisecond,
			locks:      10,
			wait:       200 * time.Millisecond,
			exp: expected{
				acquired: 10,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), test.timeout)
			defer cancel()

			redis := &redisMock{
				lock: new(sync.RWMutex),
			}

			for i := 0; i < test.locks; i++ {
				lock := NewDistributed(zap.NewNop(), redis, key, test.expiration)
				err := lock.Lock(ctx)
				require.Nil(t, err)
				time.AfterFunc(test.wait, func() {
					lock.Unlock(ctx)
				})
			}

			<-ctx.Done()
			require.Equal(t, context.DeadlineExceeded, ctx.Err())

			redis.lock.RLock()
			require.Equal(t, test.exp.acquired, redis.acquired, "acquired")
			redis.lock.RUnlock()
		})
	}
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
