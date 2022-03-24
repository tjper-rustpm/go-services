//go:build awsintegration
// +build awsintegration

package server

import (
	"context"
	"testing"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/model"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	suite := setup(ctx, t)

	var createInstanceOutput CreateInstanceOutput
	t.Run("create instance", func(t *testing.T) {
		out, err := suite.manager.CreateInstance(ctx, model.InstanceKindStandard)
		require.Nil(t, err)
		createInstanceOutput = *out
	})

	t.Run("start instance", func(t *testing.T) {
		err := suite.manager.StartInstance(ctx, *createInstanceOutput.Instance.InstanceId, "")
		require.Nil(t, err)
	})

	var associationOutput AssociationOutput
	t.Run("make instance available", func(t *testing.T) {
		out, err := suite.manager.MakeInstanceAvailable(
			ctx,
			*createInstanceOutput.Instance.InstanceId,
			*createInstanceOutput.Address.AllocationId,
		)
		require.Nil(t, err)
		associationOutput = *out
	})

	t.Run("make instance unavailable", func(t *testing.T) {
		err := suite.manager.MakeInstanceUnavailable(ctx, *associationOutput.AssociationId)
		require.Nil(t, err)
	})
	t.Run("stop instance", func(t *testing.T) {
		err := suite.manager.StopInstance(ctx, *createInstanceOutput.Instance.InstanceId)
		require.Nil(t, err)
	})
	t.Run("terminate instance", func(t *testing.T) {
		err := suite.manager.TerminateInstance(
			ctx,
			*createInstanceOutput.Instance.InstanceId,
			*createInstanceOutput.Address.AllocationId,
		)
		require.Nil(t, err)
	})
}

func setup(ctx context.Context, t *testing.T) *suite {
	awscfg, err := config.LoadDefaultConfig(ctx)
	require.Nil(t, err)

	usEastEC2 := ec2.NewFromConfig(awscfg, func(opts *ec2.Options) {
		opts.Region = "us-east-1"
	})

	return &suite{
		manager: NewManager(zap.NewNop(), usEastEC2),
	}
}

type suite struct {
	manager *Manager
}
