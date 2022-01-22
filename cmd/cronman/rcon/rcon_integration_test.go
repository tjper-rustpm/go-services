// +build rconintegration

package rcon

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var url = flag.String(
	"url",
	"ws://0.0.0.0:28016/docker",
	"websocket url to run rcon integration tests against",
)

func TestIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	suite := setup(ctx, t)
	defer suite.cleanup()

	t.Run("server info", func(t *testing.T) {
		_, err := suite.client.ServerInfo(ctx)
		assert.Nil(t, err)
	})
	t.Run("say", func(t *testing.T) {
		err := suite.client.Say(ctx, "hello rust world")
		assert.Nil(t, err)
	})
	t.Run("add moderator", func(t *testing.T) {
		err := suite.client.AddModerator(ctx, "76561197962911631")
		assert.Nil(t, err)
	})
	t.Run("add existing moderator", func(t *testing.T) {
		err := suite.client.AddModerator(ctx, "76561197962911631")
		assert.ErrorIs(t, err, ErrModeratorExists)
	})
	t.Run("remove moderator", func(t *testing.T) {
		err := suite.client.RemoveModerator(ctx, "76561197962911631")
		assert.Nil(t, err)
	})
	t.Run("remove none-existent moderator", func(t *testing.T) {
		err := suite.client.RemoveModerator(ctx, "76561197962911631")
		assert.ErrorIs(t, err, ErrModeratorDNE)
	})
	t.Run("grant bypass queue", func(t *testing.T) {
		err := suite.client.GrantPermission(ctx, "76561197962911631", "bypassqueue.allow")
		assert.Nil(t, err)
	})
	t.Run("grant bypass queue to already granted", func(t *testing.T) {
		err := suite.client.GrantPermission(ctx, "76561197962911631", "bypassqueue.allow")
		assert.ErrorIs(t, err, ErrPermissionAlreadyGranted)
	})
	t.Run("revoke bypass queue", func(t *testing.T) {
		err := suite.client.RevokePermission(ctx, "76561197962911631", "bypassqueue.allow")
		assert.Nil(t, err)
	})
	t.Run("quit", func(t *testing.T) {
		err := suite.client.Quit(ctx)
		assert.Nil(t, err)
	})
}

func setup(ctx context.Context, t *testing.T) *suite {
	waiter := NewWaiter(zap.NewNop())
	err := waiter.UntilReady(ctx, *url, 10*time.Second)
	assert.Nil(t, err)

	client, err := Dial(ctx, zap.NewNop(), *url)
	require.Nil(t, err)

	return &suite{
		client: client,
	}
}

type suite struct {
	client *Client
}

func (s suite) cleanup() {
	s.client.Close()
}
