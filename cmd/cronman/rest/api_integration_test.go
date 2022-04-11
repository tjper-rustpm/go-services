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
	"github.com/tjper/rustcron/cmd/cronman/rcon"
	"github.com/tjper/rustcron/cmd/cronman/server"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/integration"
	"github.com/tjper/rustcron/internal/rand"
	"github.com/tjper/rustcron/internal/redis"
	"github.com/tjper/rustcron/internal/session"

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
		instanceID, err := rand.GenerateString(16)
		require.Nil(t, err)

		allocationID, err := rand.GenerateString(16)
		require.Nil(t, err)

		suite.serverManager.SetCreateInstanceOutput(
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

		suite.postCreateServer(ctx, t, sess)
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
		s.Logger,
		db.NewStore(s.Logger, dbconn),
		controller.NewServerDirector(
			serverManager,
			serverManager,
			serverManager,
		),
		controller.NewRconHubMock(),
		rcon.NewWaiterMock(time.Millisecond),
		director.NewNotifier(s.Logger, redis.Redis),
	)

	api := NewAPI(
		s.Logger,
		ctrl,
		ihttp.NewSessionMiddleware(s.Logger, sessions.Manager),
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

func (s suite) postCreateServer(ctx context.Context, t *testing.T, sess *session.Session) {
	t.Helper()

	fd, err := os.Open(fmt.Sprintf("testdata/%s.json", t.Name()))
	require.Nil(t, err)
	defer fd.Close()

	body := make(map[string]interface{})
	err = json.NewDecoder(fd).Decode(&body)
	require.Nil(t, err)

	resp := s.Request(ctx, t, s.api, http.MethodPost, "/v1/server", body, sess)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)
}
