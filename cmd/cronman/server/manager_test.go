// +build awsintegration

package server

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/db"
	"github.com/tjper/rustcron/cmd/cronman/userdata"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestManagerCreateInstance(t *testing.T) {
	type expected struct {
		err error
	}
	tests := []struct {
		template db.RustpmInstanceType
		exp      expected
	}{
		0: {template: db.RustpmInstanceTypeSTANDARD, exp: expected{err: nil}},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			m := newManager(t)
			_, err := m.CreateInstance(ctx, test.template)
			assert.Equal(t, test.exp.err, err)
		})
	}
}

func TestManagerStartInstance(t *testing.T) {
	type expected struct {
		err error
	}
	tests := []struct {
		id       string
		userdata string
		exp      expected
	}{
		0: {
			id:       "i-03b2c52e8ea2af712",
			userdata: generateUserData(),
			exp: expected{
				err: nil,
			},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			m := newManager(t)
			err := m.StartInstance(ctx, test.id, test.userdata)
			assert.Equal(t, test.exp.err, err)
		})
	}
}

func TestManagerStopInstance(t *testing.T) {
	type expected struct {
		err error
	}
	tests := []struct {
		id  string
		exp expected
	}{
		0: {
			id: "i-03b2c52e8ea2af712",
			exp: expected{
				err: nil,
			},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			m := newManager(t)
			err := m.StopInstance(ctx, test.id)
			assert.Equal(t, test.exp.err, err)
		})
	}
}

// --- helpers ---

func newManager(t *testing.T) *Manager {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	require.Nil(t, err)

	return NewManager(
		zap.NewExample(),
		ec2.NewFromConfig(cfg),
	)
}

func generateUserData() string {
	return userdata.Generate(
		"rustpm",
		"rustpmrconpass",
		100,
		2000,
		100,
		1,
		30,
		userdata.WithQueueBypassPlugin(),
	)
}
