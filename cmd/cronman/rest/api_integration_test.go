//go:build integration
// +build integration

package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/controller"
	"github.com/tjper/rustcron/cmd/cronman/db"
	"github.com/tjper/rustcron/cmd/cronman/director"
	"github.com/tjper/rustcron/cmd/cronman/rcon"
	"github.com/tjper/rustcron/cmd/cronman/server"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/integration"
	"github.com/tjper/rustcron/internal/rand"
	"github.com/tjper/rustcron/internal/redis"
	"github.com/tjper/rustcron/internal/session"
	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/require"
)

func TestCreateServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.sessions.CreateSession(ctx, t, "rustcron@gmail.com", session.RoleAdmin)

	t.Run("create server with admin user", func(t *testing.T) {
		resp := suite.postCreateServer(ctx, t, sess, "testdata/default-body.json")
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	sess = suite.sessions.CreateSession(ctx, t, "rustcron@gmail.com", session.RoleStandard)

	t.Run("create server with standard user", func(t *testing.T) {
		resp := suite.postCreateServer(ctx, t, sess, "testdata/default-body.json")
		defer resp.Body.Close()

		require.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

func TestStartServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.sessions.CreateSession(ctx, t, "rustcron@gmail.com", session.RoleAdmin)

	var serverID string
	t.Run("create server with admin user", func(t *testing.T) {
		createResp := suite.postCreateServer(ctx, t, sess, "testdata/default-body.json")
		defer createResp.Body.Close()

		require.Equal(t, http.StatusCreated, createResp.StatusCode)

		var server map[string]interface{}
		err := json.NewDecoder(createResp.Body).Decode(&server)
		require.Nil(t, err)

		iServerID, ok := server["id"]
		require.True(t, ok)
		serverID, ok = iServerID.(string)
		require.True(t, ok)
	})

	t.Run("start server with admin user", func(t *testing.T) {
		resp := suite.postStartServer(ctx, t, sess, serverID)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("start server that is live", func(t *testing.T) {
		resp := suite.postStartServer(ctx, t, sess, serverID)
		defer resp.Body.Close()

		require.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	sess = suite.sessions.CreateSession(ctx, t, "rustcron@gmail.com", session.RoleStandard)

	t.Run("start server with standard user", func(t *testing.T) {
		resp := suite.postStartServer(ctx, t, sess, serverID)
		defer resp.Body.Close()

		require.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

func TestStopServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.sessions.CreateSession(ctx, t, "rustcron@gmail.com", session.RoleAdmin)

	var serverID string
	t.Run("create server with admin user", func(t *testing.T) {
		createResp := suite.postCreateServer(ctx, t, sess, "testdata/default-body.json")
		defer createResp.Body.Close()

		require.Equal(t, http.StatusCreated, createResp.StatusCode)

		var server map[string]interface{}
		err := json.NewDecoder(createResp.Body).Decode(&server)
		require.Nil(t, err)

		iServerID, ok := server["id"]
		require.True(t, ok)
		serverID, ok = iServerID.(string)
		require.True(t, ok)
	})

	t.Run("start server with admin user", func(t *testing.T) {
		resp := suite.postStartServer(ctx, t, sess, serverID)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	standardSess := suite.sessions.CreateSession(ctx, t, "rustcron@gmail.com", session.RoleStandard)

	t.Run("stop server with standard user", func(t *testing.T) {
		resp := suite.postStopServer(ctx, t, standardSess, serverID)
		defer resp.Body.Close()

		require.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("stop server with admin user", func(t *testing.T) {
		resp := suite.postStopServer(ctx, t, sess, serverID)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("stop server that is dormant", func(t *testing.T) {
		resp := suite.postStopServer(ctx, t, sess, serverID)
		defer resp.Body.Close()

		require.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}

func setup(
	ctx context.Context,
	t *testing.T,
) *suite {
	t.Helper()

	redis := redis.InitSuite(ctx, t)
	err := redis.Redis.FlushAll(ctx).Err()
	require.Nil(t, err)

	s := integration.InitSuite(ctx, t)
	sessions := session.InitSuite(ctx, t)

	logger := zap.NewExample()

	const (
		dsn        = "host=db user=postgres password=password dbname=postgres port=5432 sslmode=disable TimeZone=UTC"
		migrations = "file://../db/migrations"
	)

	dbconn, err := db.Open(dsn)
	require.Nil(t, err)

	err = db.Migrate(dbconn, migrations)
	require.Nil(t, err)

	serverManager := server.NewMockManager()

	ctrl := controller.New(
		logger,
		db.NewStore(logger, dbconn),
		controller.NewServerDirector(
			serverManager,
			serverManager,
			serverManager,
		),
		controller.NewRconHubMock(),
		rcon.NewWaiterMock(time.Millisecond),
		director.NewNotifier(logger, redis.Redis),
	)

	api := NewAPI(
		s.Logger,
		ctrl,
		ihttp.NewSessionMiddleware(logger, sessions.Manager),
	)

	return &suite{
		Suite:         *s,
		sessions:      sessions,
		api:           api.Mux,
		serverManager: serverManager,
	}
}

type suite struct {
	integration.Suite
	sessions *session.Suite

	api           http.Handler
	serverManager *server.MockManager
}

func (s suite) postCreateServer(ctx context.Context, t *testing.T, sess *session.Session, path string) *http.Response {
	t.Helper()

	instanceID, err := rand.GenerateString(16)
	require.Nil(t, err)

	allocationID, err := rand.GenerateString(16)
	require.Nil(t, err)

	s.serverManager.SetCreateInstanceOutput(
		&server.CreateInstanceOutput{
			Instance: types.Instance{
				InstanceId: aws.String(instanceID),
			},
			Address: ec2.AllocateAddressOutput{
				AllocationId: aws.String(allocationID),
				PublicIp:     aws.String("127.0.0.1"),
			},
		},
		nil,
	)

	fd, err := os.Open(path)
	require.Nil(t, err)
	defer fd.Close()

	body := make(map[string]interface{})
	err = json.NewDecoder(fd).Decode(&body)
	require.Nil(t, err)

	return s.Request(ctx, t, s.api, http.MethodPost, "/v1/server", body, sess)
}

func (s suite) postStartServer(ctx context.Context, t *testing.T, sess *session.Session, serverID string) *http.Response {
	t.Helper()

	associationID, err := rand.GenerateString(16)
	require.Nil(t, err)

	s.serverManager.SetMakeInstanceAvailableOutput(
		&server.AssociationOutput{
			AssociateAddressOutput: ec2.AssociateAddressOutput{
				AssociationId: aws.String(associationID),
			},
		},
		nil,
	)

	body := map[string]interface{}{
		"serverId": serverID,
	}
	return s.Request(ctx, t, s.api, http.MethodPost, "/v1/server/start", body, sess)
}

func (s suite) postStopServer(ctx context.Context, t *testing.T, sess *session.Session, serverID string) *http.Response {
	t.Helper()

	body := map[string]interface{}{
		"serverId": serverID,
	}
	return s.Request(ctx, t, s.api, http.MethodPost, "/v1/server/stop", body, sess)
}
