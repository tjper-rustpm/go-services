package server

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/model"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"go.uber.org/zap"
)

var (
	errUnexpectedNumberOfInstances = errors.New("unexpected number of EC2 instances")
)

func NewManager(
	logger *zap.Logger,
	ec2 *ec2.Client,
) *Manager {
	return &Manager{
		logger: logger,
		ec2:    ec2,
	}
}

// Manager provides an API by which to manage Rust server intances.
type Manager struct {
	logger *zap.Logger
	ec2    *ec2.Client
}

// CreateInstanceOutput is the return value of CreateInstance.
type CreateInstanceOutput struct {
	Instance types.Instance
	Address  ec2.AllocateAddressOutput
}

// CreateInstance creates a Rust server based on the template provided. The
// creation may be terminated via the ctx. Cancelling the context does not
// necessarily terminate resources created.
func (m Manager) CreateInstance(
	ctx context.Context,
	template model.InstanceKind,
) (*CreateInstanceOutput, error) {
	tmpl := fmt.Sprintf("rustpm-%s", strings.ToLower(string(template)))
	m.logger.Info("creating instance", zap.String("template", tmpl))

	var instance types.Instance
	{ // launch EC2 instance
		input := &ec2.RunInstancesInput{
			MinCount: 1,
			MaxCount: 1,
			LaunchTemplate: &types.LaunchTemplateSpecification{
				LaunchTemplateName: aws.String(tmpl),
			},
		}

		reservation, err := m.ec2.RunInstances(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("error launching EC2 instance; %w", err)
		}
		if len(reservation.Instances) != 1 {
			return nil, errUnexpectedNumberOfInstances
		}
		instance = reservation.Instances[0]
	}

	if err := m.WaitUntilInstanceStatusOk(ctx, *instance.InstanceId); err != nil {
		return nil, fmt.Errorf(
			"error waiting for launched EC2 instance \"%s\"; %w",
			*instance.InstanceId,
			err,
		)
	}

	if err := m.StopInstance(ctx, *instance.InstanceId); err != nil {
		return nil, fmt.Errorf(
			"error stopping launched EC2 instance \"%s\"; %w",
			*instance.InstanceId,
			err,
		)
	}

	var address *ec2.AllocateAddressOutput
	{ // allocate elastic IP address
		input := &ec2.AllocateAddressInput{
			Domain: types.DomainType("vpc"),
		}

		addr, err := m.ec2.AllocateAddress(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("error allocating elastic IP address; %w", err)
		}
		address = addr
	}

	return &CreateInstanceOutput{
		Instance: instance,
		Address:  *address,
	}, nil
}

// WaitUntilInstanceStatusOk waits until the specified instance id's status is
// "ok". An "ok" state indicates the instance has initialized and is reachable.
func (m Manager) WaitUntilInstanceStatusOk(ctx context.Context, id string) error {
	m.logger.Info("waiting for instance system status ok", zap.String("instance-id", id))
	input := &ec2.DescribeInstanceStatusInput{
		InstanceIds: []string{id},
	}

	waiter := ec2.NewSystemStatusOkWaiter(m.ec2)
	if err := waiter.Wait(ctx, input, 10*time.Minute); err != nil {
		return fmt.Errorf("error waiting for launched EC2 instance; %w", err)
	}
	return nil
}

// StartInstance updates the Rust server instance with the specified userdata
// and starts the server.
func (m Manager) StartInstance(
	ctx context.Context,
	id,
	userdata string,
) error {
	m.logger.Info("updating instance userdata", zap.String("instance-id", id))
	{ // update instance userdata
		input := &ec2.ModifyInstanceAttributeInput{
			InstanceId: aws.String(id),
			UserData:   &types.BlobAttributeValue{Value: []byte(userdata)},
		}

		_, err := m.ec2.ModifyInstanceAttribute(ctx, input)
		if err != nil {
			return fmt.Errorf("error modifying EC2 instance \"%s\" userdata; %w", id, err)
		}
	}

	m.logger.Info("starting instance", zap.String("instance-id", id))
	{ // start EC2 instance
		input := &ec2.StartInstancesInput{
			InstanceIds: []string{id},
		}

		_, err := m.ec2.StartInstances(ctx, input)
		if err != nil {
			return fmt.Errorf("error starting EC2 instance; %w", err)
		}
	}

	if err := m.WaitUntilInstanceStatusOk(ctx, id); err != nil {
		return fmt.Errorf("error starting EC2 instance \"%s\"; %w", id, err)
	}

	return nil
}

// StopInstance stops the specified Rust server.
func (m Manager) StopInstance(
	ctx context.Context,
	id string,
) error {
	m.logger.Info("stopping instance", zap.String("instance-id", id))

	{
		input := &ec2.StopInstancesInput{
			InstanceIds: []string{id},
		}

		_, err := m.ec2.StopInstances(ctx, input)
		if err != nil {
			return fmt.Errorf("error stopping EC2 instance; %w", err)
		}
	}

	{
		input := &ec2.DescribeInstancesInput{
			InstanceIds: []string{id},
		}

		m.logger.Info("waiting for instance to stop", zap.String("instance-id", id))
		waiter := ec2.NewInstanceStoppedWaiter(m.ec2)
		if err := waiter.Wait(ctx, input, 10*time.Minute); err != nil {
			return fmt.Errorf("error waiting for EC2 instance to stop; %w", err)
		}
	}
	return nil
}

// MakeInstanceAvailable associates the Rust server instance with the specified
// allocationId. The allocationId is associated with a elastic IP.
func (m Manager) MakeInstanceAvailable(
	ctx context.Context,
	instanceId,
	allocationId string,
) (*AssociationOutput, error) {
	m.logger.Info(
		"making instance available",
		zap.String("instance-id", instanceId),
		zap.String("allocation-id", allocationId),
	)

	input := &ec2.AssociateAddressInput{
		InstanceId:   aws.String(instanceId),
		AllocationId: aws.String(allocationId),
	}

	association, err := m.ec2.AssociateAddress(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error making instance available; %w", err)
	}
	return &AssociationOutput{
		AssociateAddressOutput: *association,
	}, nil
}

// AssociationOutput is the return value of MakeInstanceAvailable.
type AssociationOutput struct {
	ec2.AssociateAddressOutput
}

// MakeInstanceUnavailable disassociates the specified associationId. Making
// the related Rust server instance unavailable.
func (m Manager) MakeInstanceUnavailable(
	ctx context.Context,
	associationId string,
) error {
	m.logger.Info(
		"making instance unavailable",
		zap.String("association-id", associationId),
	)

	input := &ec2.DisassociateAddressInput{
		AssociationId: aws.String(associationId),
	}

	if _, err := m.ec2.DisassociateAddress(ctx, input); err != nil {
		return fmt.Errorf("error making instance unavailable; %w", err)
	}
	return nil
}

// TerminateInstance permanently deletes the instance and it's allocated
// address.
func (m Manager) TerminateInstance(
	ctx context.Context,
	instanceId string,
	allocationId string,
) error {
	{ // terminate instance
		input := &ec2.TerminateInstancesInput{
			InstanceIds: []string{instanceId},
		}
		if _, err := m.ec2.TerminateInstances(ctx, input); err != nil {
			return fmt.Errorf("terminate instances; id: %s, error: %w", instanceId, err)
		}
	}
	{ // release address allocation
		input := &ec2.ReleaseAddressInput{
			AllocationId: aws.String(allocationId),
		}
		if _, err := m.ec2.ReleaseAddress(ctx, input); err != nil {
			return fmt.Errorf("release address; id: %s, error: %w", allocationId, err)
		}
	}
	return nil
}
