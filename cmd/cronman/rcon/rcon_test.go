// +build rconintegration

package rcon

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWaitUntilReady(t *testing.T) {
	type expected struct {
		err error
	}
	tests := map[string]struct {
		url  string
		wait time.Duration
		exp  expected
	}{
		"base": {
			url:  "ws://44.195.233.247:28016/omega-rcon-password",
			wait: time.Minute,
			exp: expected{
				err: nil,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()

			waiter := NewWaiter(zap.NewExample())

			err := waiter.UntilReady(ctx, test.url, test.wait)
			assert.Nil(t, err)
		})
	}
}

func TestServerInfo(t *testing.T) {
	type expected struct {
		err error
	}
	tests := map[string]struct {
		url string
		exp expected
	}{
		"hello": {
			url: "ws://44.195.233.247:28016/omega-rcon-password",
			exp: expected{
				err: nil,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			var wg sync.WaitGroup
			for i := 0; i < 3; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					client, err := Dial(ctx, zap.NewExample(), test.url)
					require.Nil(t, err)
					defer client.Close()

					res, err := client.ServerInfo(ctx)
					assert.Equal(t, test.exp.err, err)
					t.Logf("res: %v\n", res)
				}()
			}
			wg.Wait()
		})
	}
}

func TestSay(t *testing.T) {
	type expected struct {
		err error
	}
	tests := map[string]struct {
		url string
		msg string
		exp expected
	}{
		"hello": {
			url: "ws://52.207.55.232:28016/omega-rcon-password",
			msg: "hello server",
			exp: expected{
				err: nil,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			client, err := Dial(ctx, zap.NewExample(), test.url)
			require.Nil(t, err)
			defer client.Close()

			err = client.Say(ctx, test.msg)
			assert.Equal(t, test.exp.err, err)
		})
	}
}

func TestAddRemoveModerator(t *testing.T) {
	type expected struct {
		addErr    error
		existsErr error
		removeErr error
		dneErr    error
	}
	tests := map[string]struct {
		url string
		id  string
		exp expected
	}{
		"add-exists-remove": {
			url: "ws://52.207.55.232:28016/omega-rcon-password",
			id:  "76561197962911631",
			exp: expected{
				addErr:    nil,
				existsErr: ErrModeratorExists,
				removeErr: nil,
				dneErr:    ErrModeratorDNE,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			client, err := Dial(ctx, zap.NewExample(), test.url)
			require.Nil(t, err)
			defer client.Close()

			err = client.AddModerator(ctx, test.id)
			assert.Equal(t, test.exp.addErr, err)

			err = client.AddModerator(ctx, test.id)
			assert.Equal(t, test.exp.existsErr, err)

			err = client.RemoveModerator(ctx, test.id)
			assert.Equal(t, test.exp.removeErr, err)

			err = client.RemoveModerator(ctx, test.id)
			assert.Equal(t, test.exp.dneErr, err)
		})
	}
}

func TestGrantRevokePermission(t *testing.T) {
	type expected struct {
		grantErr          error
		alreadyGrantedErr error
		revokeErr         error
	}
	tests := map[string]struct {
		url        string
		steamId    string
		permission string
		exp        expected
	}{
		"grant-alreadyGranted-revoke": {
			url:        "ws://44.195.14.16:28016/omega-rcon-password",
			steamId:    "76561197962911631",
			permission: "bypassqueue.allow",
			exp: expected{
				grantErr:          nil,
				alreadyGrantedErr: ErrPermissionAlreadyGranted,
				revokeErr:         nil,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			client, err := Dial(ctx, zap.NewExample(), test.url)
			require.Nil(t, err)
			defer client.Close()

			err = client.GrantPermission(ctx, test.steamId, test.permission)
			assert.Equal(t, test.exp.grantErr, err)

			err = client.GrantPermission(ctx, test.steamId, test.permission)
			assert.Equal(t, test.exp.alreadyGrantedErr, err)

			err = client.RevokePermission(ctx, test.steamId, test.permission)
			assert.Equal(t, test.exp.revokeErr, err)
		})
	}
}

func TestQuit(t *testing.T) {
	type expected struct {
		err error
	}
	tests := map[string]struct {
		url string
		exp expected
	}{
		"quit": {
			url: "ws://18.212.149.82:28016/rustpmrconpass",
			exp: expected{
				err: nil,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			client, err := Dial(ctx, zap.NewExample(), test.url)
			require.Nil(t, err)
			defer client.Close()

			err = client.Quit(ctx)
			assert.Equal(t, test.exp.err, err)
		})
	}
}
