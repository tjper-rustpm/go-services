//go:build integration
// +build integration

package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/controller"
	"github.com/tjper/rustcron/cmd/cronman/db"
	"github.com/tjper/rustcron/cmd/cronman/director"
	"github.com/tjper/rustcron/cmd/cronman/model"
	"github.com/tjper/rustcron/cmd/cronman/rcon"
	"github.com/tjper/rustcron/cmd/cronman/server"
	"github.com/tjper/rustcron/internal/gorm"
	"github.com/tjper/rustcron/internal/healthz"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/integration"
	"github.com/tjper/rustcron/internal/rand"
	"github.com/tjper/rustcron/internal/redis"
	"github.com/tjper/rustcron/internal/session"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCreateServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.sessions.CreateSession(ctx, t, "rustcron@gmail.com", session.RoleAdmin)

	t.Run("create server with admin user", func(t *testing.T) {
		instanceID, err := rand.GenerateString(16)
		require.Nil(t, err)

		allocationID, err := rand.GenerateString(16)
		require.Nil(t, err)

		suite.serverManager.SetCreateInstanceHandler(func(context.Context, model.InstanceKind) (*server.CreateInstanceOutput, error) {
			return &server.CreateInstanceOutput{
					Instance: types.Instance{
						InstanceId: aws.String(instanceID),
					},
					Address: ec2.AllocateAddressOutput{
						AllocationId: aws.String(allocationID),
						PublicIp:     aws.String("127.0.0.1"),
					},
				},
				nil
		})

		resp := suite.postCreateServer(ctx, t, sess, "testdata/default-body.json")
		defer resp.Body.Close()

		require.Equal(t, http.StatusAccepted, resp.StatusCode)
	})

	t.Run("create server that has options with admin user", func(t *testing.T) {
		instanceID, err := rand.GenerateString(16)
		require.Nil(t, err)

		allocationID, err := rand.GenerateString(16)
		require.Nil(t, err)

		suite.serverManager.SetCreateInstanceHandler(func(context.Context, model.InstanceKind) (*server.CreateInstanceOutput, error) {
			return &server.CreateInstanceOutput{
					Instance: types.Instance{
						InstanceId: aws.String(instanceID),
					},
					Address: ec2.AllocateAddressOutput{
						AllocationId: aws.String(allocationID),
						PublicIp:     aws.String("127.0.0.1"),
					},
				},
				nil
		})

		resp := suite.postCreateServer(ctx, t, sess, "testdata/options-body.json")
		defer resp.Body.Close()

		require.Equal(t, http.StatusAccepted, resp.StatusCode)
	})

	sess = suite.sessions.CreateSession(ctx, t, "rustcron@gmail.com", session.RoleStandard)

	t.Run("create server with standard user", func(t *testing.T) {
		instanceID, err := rand.GenerateString(16)
		require.Nil(t, err)

		allocationID, err := rand.GenerateString(16)
		require.Nil(t, err)

		suite.serverManager.SetCreateInstanceHandler(func(context.Context, model.InstanceKind) (*server.CreateInstanceOutput, error) {
			return &server.CreateInstanceOutput{
					Instance: types.Instance{
						InstanceId: aws.String(instanceID),
					},
					Address: ec2.AllocateAddressOutput{
						AllocationId: aws.String(allocationID),
						PublicIp:     aws.String("127.0.0.1"),
					},
				},
				nil
		})

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
		instanceID, err := rand.GenerateString(16)
		require.Nil(t, err)

		allocationID, err := rand.GenerateString(16)
		require.Nil(t, err)

		suite.serverManager.SetCreateInstanceHandler(func(context.Context, model.InstanceKind) (*server.CreateInstanceOutput, error) {
			return &server.CreateInstanceOutput{
					Instance: types.Instance{
						InstanceId: aws.String(instanceID),
					},
					Address: ec2.AllocateAddressOutput{
						AllocationId: aws.String(allocationID),
						PublicIp:     aws.String("127.0.0.1"),
					},
				},
				nil
		})

		createResp := suite.postCreateServer(ctx, t, sess, "testdata/default-body.json")
		defer createResp.Body.Close()

		require.Equal(t, http.StatusAccepted, createResp.StatusCode)

		var server map[string]interface{}
		err = json.NewDecoder(createResp.Body).Decode(&server)
		require.Nil(t, err)

		iServerID, ok := server["id"]
		require.True(t, ok)
		serverID, ok = iServerID.(string)
		require.True(t, ok)
	})

	t.Run("wait until server is dormant", func(t *testing.T) {
		isDormant := func(server map[string]interface{}) bool {
			return server["kind"] == "dormant"
		}

		suite.waitUntilServer(ctx, t, sess, serverID, isDormant)
	})

	t.Run("start server with admin user", func(t *testing.T) {
		resp := suite.postStartServer(ctx, t, sess, serverID)
		defer resp.Body.Close()

		require.Equal(t, http.StatusAccepted, resp.StatusCode)
	})

	t.Run("wait until server is live", func(t *testing.T) {
		isLive := func(server map[string]interface{}) bool {
			return server["kind"] == "live"
		}

		suite.waitUntilServer(ctx, t, sess, serverID, isLive)
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
		instanceID, err := rand.GenerateString(16)
		require.Nil(t, err)

		allocationID, err := rand.GenerateString(16)
		require.Nil(t, err)

		suite.serverManager.SetCreateInstanceHandler(func(context.Context, model.InstanceKind) (*server.CreateInstanceOutput, error) {
			return &server.CreateInstanceOutput{
					Instance: types.Instance{
						InstanceId: aws.String(instanceID),
					},
					Address: ec2.AllocateAddressOutput{
						AllocationId: aws.String(allocationID),
						PublicIp:     aws.String("127.0.0.1"),
					},
				},
				nil
		})

		createResp := suite.postCreateServer(ctx, t, sess, "testdata/default-body.json")
		defer createResp.Body.Close()

		require.Equal(t, http.StatusAccepted, createResp.StatusCode)

		var server map[string]interface{}
		err = json.NewDecoder(createResp.Body).Decode(&server)
		require.Nil(t, err)

		iServerID, ok := server["id"]
		require.True(t, ok)
		serverID, ok = iServerID.(string)
		require.True(t, ok)
	})

	t.Run("wait until server is dormant", func(t *testing.T) {
		isDormant := func(server map[string]interface{}) bool {
			return server["kind"] == "dormant"
		}

		suite.waitUntilServer(ctx, t, sess, serverID, isDormant)
	})

	t.Run("start server with admin user", func(t *testing.T) {
		resp := suite.postStartServer(ctx, t, sess, serverID)
		defer resp.Body.Close()

		require.Equal(t, http.StatusAccepted, resp.StatusCode)
	})

	t.Run("wait until server is live", func(t *testing.T) {
		isLive := func(server map[string]interface{}) bool {
			return server["kind"] == "live"
		}

		suite.waitUntilServer(ctx, t, sess, serverID, isLive)
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

		require.Equal(t, http.StatusAccepted, resp.StatusCode)
	})

	t.Run("wait until server is dormant", func(t *testing.T) {
		isDormant := func(server map[string]interface{}) bool {
			return server["kind"] == "dormant"
		}

		suite.waitUntilServer(ctx, t, sess, serverID, isDormant)
	})

	t.Run("stop server that is dormant", func(t *testing.T) {
		resp := suite.postStopServer(ctx, t, sess, serverID)
		defer resp.Body.Close()

		require.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}

func TestWipeServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	suite := setup(ctx, t)

	sess := suite.sessions.CreateSession(ctx, t, "rustcron@gmail.com", session.RoleAdmin)

	var serverID string
	var instanceID string
	t.Run("create server with admin user", func(t *testing.T) {
		var err error
		instanceID, err = rand.GenerateString(16)
		require.Nil(t, err)

		allocationID, err := rand.GenerateString(16)
		require.Nil(t, err)

		suite.serverManager.SetCreateInstanceHandler(func(context.Context, model.InstanceKind) (*server.CreateInstanceOutput, error) {
			return &server.CreateInstanceOutput{
					Instance: types.Instance{
						InstanceId: aws.String(instanceID),
					},
					Address: ec2.AllocateAddressOutput{
						AllocationId: aws.String(allocationID),
						PublicIp:     aws.String("127.0.0.1"),
					},
				},
				nil
		})

		createResp := suite.postCreateServer(ctx, t, sess, "testdata/default-body.json")
		defer createResp.Body.Close()

		require.Equal(t, http.StatusAccepted, createResp.StatusCode)

		var server map[string]interface{}
		err = json.NewDecoder(createResp.Body).Decode(&server)
		require.Nil(t, err)

		iServerID, ok := server["id"]
		require.True(t, ok)
		serverID, ok = iServerID.(string)
		require.True(t, ok)
	})

	t.Run("wait until server is dormant", func(t *testing.T) {
		isDormant := func(server map[string]interface{}) bool {
			return server["kind"] == "dormant"
		}

		suite.waitUntilServer(ctx, t, sess, serverID, isDormant)
	})

	var seed uint16
	var salt uint16
	t.Run("require that server map seed and salt are set", func(t *testing.T) {
		resp := suite.getServer(ctx, t, sess, serverID)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var server DormantServer
		err := json.NewDecoder(resp.Body).Decode(&server)
		require.Nil(t, err)

		require.NotEmpty(t, server.MapSeed)
		require.NotEmpty(t, server.MapSalt)

		seed = server.MapSeed
		salt = server.MapSalt
	})

	t.Run("wipe server w/ random seed and salt", func(t *testing.T) {
		body := map[string]interface{}{
			"serverId": serverID,
			"kind":     model.WipeKindMap,
		}

		resp := suite.wipeServer(ctx, t, sess, body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("require that server map seed and salt have changed", func(t *testing.T) {
		resp := suite.getServer(ctx, t, sess, serverID)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var server DormantServer
		err := json.NewDecoder(resp.Body).Decode(&server)
		require.Nil(t, err)

		require.NotEqual(t, seed, server.MapSeed, "previous seed is %d and new seed is %d", seed, server.MapSeed)
		require.NotEqual(t, salt, server.MapSalt, "previous salt is %d and new salt is %d", salt, server.MapSalt)

		seed = server.MapSeed
		salt = server.MapSalt
	})

	t.Run("start server recently wiped server", func(t *testing.T) {
		check := func(_ context.Context, id string, userdata string) error {
			require.Equal(t, instanceID, id, "expected instance ID: \"%s\", actual: \"%s\"", instanceID, id)
			require.Regexp(t, fmt.Sprintf("server\\.seed %d", seed), userdata)
			require.Regexp(t, fmt.Sprintf("server\\.salt %d", salt), userdata)
			require.Regexp(t, `"proceduralmap\\\.\*\\\.\*\\\.\*\\\.map" \| xargs rm`, userdata)
			return nil
		}
		suite.serverManager.SetStartInstanceHandler(check)

		resp := suite.postStartServer(ctx, t, sess, serverID)
		defer resp.Body.Close()

		require.Equal(t, http.StatusAccepted, resp.StatusCode)

		isLive := func(server map[string]interface{}) bool {
			return server["kind"] == "live"
		}
		suite.waitUntilServer(ctx, t, sess, serverID, isLive)

		// Necessary to not re-use instance handler function in future executions
		// of StartInstance.
		suite.serverManager.SetStartInstanceHandler(nil)
	})

	t.Run("start server w/ no wipe to apply", func(t *testing.T) {
		stopResp := suite.postStopServer(ctx, t, sess, serverID)
		defer stopResp.Body.Close()

		require.Equal(t, http.StatusAccepted, stopResp.StatusCode)

		isDormant := func(server map[string]interface{}) bool {
			return server["kind"] == "dormant"
		}
		suite.waitUntilServer(ctx, t, sess, serverID, isDormant)

		check := func(_ context.Context, id string, userdata string) error {
			require.Equal(t, instanceID, id, "expected instance ID: \"%s\", actual: \"%s\"", instanceID, id)
			require.Regexp(t, fmt.Sprintf("server\\.seed %d", seed), userdata)
			require.Regexp(t, fmt.Sprintf("server\\.salt %d", salt), userdata)
			require.NotRegexp(t, `"proceduralmap\\\.\*\\\.\*\\\.\*\\\.map" \| xargs rm`, userdata)
			return nil
		}
		suite.serverManager.SetStartInstanceHandler(check)

		startResp := suite.postStartServer(ctx, t, sess, serverID)
		defer startResp.Body.Close()

		require.Equal(t, http.StatusAccepted, startResp.StatusCode)

		isLive := func(server map[string]interface{}) bool {
			return server["kind"] == "live"
		}
		suite.waitUntilServer(ctx, t, sess, serverID, isLive)

		// Necessary to not re-use instance handler function in future executions
		// of StartInstance.
		suite.serverManager.SetStartInstanceHandler(nil)
	})

	t.Run("full wipe live server", func(t *testing.T) {
		check := func(_ context.Context, id string, userdata string) error {
			require.Equal(t, instanceID, id, "expected instance ID: \"%s\", actual: \"%s\"", instanceID, id)
			require.Regexp(t, `"proceduralmap\\\.\*\\\.\*\\\.\*\\\.map" \| xargs rm`, userdata)
			require.Regexp(t, `"player\\\.blueprints\\\.\*\\\.db" \| xargs rm`, userdata)
			return nil
		}
		suite.serverManager.SetStartInstanceHandler(check)

		body := map[string]interface{}{
			"serverId": serverID,
			"kind":     model.WipeKindFull,
		}

		resp := suite.wipeServer(ctx, t, sess, body)
		defer resp.Body.Close()

		require.Equal(t, http.StatusAccepted, resp.StatusCode)

		hasUpdated := func(server map[string]interface{}) bool {
			newSeed := uint16(server["mapSeed"].(float64))
			newSalt := uint16(server["mapSalt"].(float64))

			return newSeed != seed && newSalt != salt
		}
		suite.waitUntilServer(ctx, t, sess, serverID, hasUpdated)

		isLive := func(server map[string]interface{}) bool {
			return server["kind"] == "live"
		}
		suite.waitUntilServer(ctx, t, sess, serverID, isLive)

		// Necessary to not re-use instance handler function in future executions
		// of StartInstance.
		suite.serverManager.SetStartInstanceHandler(nil)
	})
}

func setup(ctx context.Context, t *testing.T) *suite {
	t.Helper()

	redis := redis.InitSuite(ctx, t)
	err := redis.Redis.FlushAll(ctx).Err()
	require.Nil(t, err)

	s := integration.InitSuite(ctx, t)
	sessions := session.InitSuite(ctx, t)

	logger := zap.NewNop()

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
		gorm.NewStore(dbconn),
		controller.NewServerDirector(
			serverManager,
			serverManager,
			serverManager,
		),
		rcon.NewHubMock(),
		rcon.NewWaiterMock(time.Millisecond),
		director.NewNotifier(logger, redis.Redis),
	)

	healthz := healthz.NewHTTP()

	api := NewAPI(
		logger,
		ctrl,
		ihttp.NewSessionMiddleware(logger, sessions.Manager),
		healthz,
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

	fd, err := os.Open(path)
	require.Nil(t, err)
	defer fd.Close()

	body := make(map[string]interface{})
	err = json.NewDecoder(fd).Decode(&body)
	require.Nil(t, err)

	return s.Request(ctx, t, s.api, http.MethodPost, "/v1/server", body, sess)
}

func (s suite) waitUntilServer(
	ctx context.Context,
	t *testing.T,
	sess *session.Session,
	serverID string,
	is func(server map[string]interface{}) bool,
) error {
	t.Helper()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			resp := s.Request(ctx, t, s.api, http.MethodGet, fmt.Sprintf("/v1/server/%s", serverID), nil, sess)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				continue
			}

			server := make(map[string]interface{})
			err := json.NewDecoder(resp.Body).Decode(&server)
			require.Nil(t, err)

			if is(server) {
				return nil
			}
		}
	}
}

func (s suite) getServer(ctx context.Context, t *testing.T, sess *session.Session, serverID string) *http.Response {
	t.Helper()

	return s.Request(ctx, t, s.api, http.MethodGet, fmt.Sprintf("/v1/server/%s", serverID), nil, sess)
}

func (s suite) wipeServer(ctx context.Context, t *testing.T, sess *session.Session, body map[string]interface{}) *http.Response {
	t.Helper()

	return s.Request(ctx, t, s.api, http.MethodPost, "/v1/server/wipe", body, sess)
}

func (s suite) postStartServer(ctx context.Context, t *testing.T, sess *session.Session, serverID string) *http.Response {
	t.Helper()

	associationID, err := rand.GenerateString(16)
	require.Nil(t, err)

	s.serverManager.SetMakeInstanceAvailableHandler(func(context.Context, string, string) (*server.AssociationOutput, error) {
		return &server.AssociationOutput{
				AssociateAddressOutput: ec2.AssociateAddressOutput{
					AssociationId: aws.String(associationID),
				},
			},
			nil
	})

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
