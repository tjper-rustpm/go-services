package stream

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrUnexpectedStreamCount  = errors.New("unexpected stream count")
	ErrUnexpectedMessageCount = errors.New("unexpected message count")
	ErrNoPending              = errors.New("no pending messages")

	errInvalidPayload = errors.New("invalid payload")
	errBusyGroup      = errors.New("BUSYGROUP Consumer Group name already exists")
)

const (
	stream = "interserviceEventStream"
	start  = "0"
	maxlen = 20000
)

// Init intializes a stream Client associated with the specified with group.
// Multiple Client instances with the same group will form a round-robin queue.
func Init(ctx context.Context, logger *zap.Logger, rdb *redis.Client, group string) (*Client, error) {
	err := rdb.XGroupCreateMkStream(ctx, stream, group, start).Err()
	if err != nil && !(err.Error() == errBusyGroup.Error()) {
		return nil, fmt.Errorf("initializing stream; error: %w", err)
	}

	return &Client{
		logger:     logger,
		rdb:        rdb,
		group:      group,
		consumer:   uuid.New().String(),
		mutex:      new(sync.RWMutex),
		claimStart: "0-0",
	}, nil
}

// Client is a persistent streaming client.
type Client struct {
	logger *zap.Logger
	rdb    *redis.Client

	group    string
	consumer string

	mutex      *sync.RWMutex
	claimStart string
}

// Write writes b to the Client's persistent stream.
func (c Client) Write(ctx context.Context, b []byte) error {
	c.logger.Debug("write stream", zap.ByteString("bytes", b))

	args := &redis.XAddArgs{
		Stream:       stream,
		MaxLenApprox: maxlen,
		ID:           "*",
		Values:       map[string]interface{}{"payload": b},
	}
	if err := c.rdb.XAdd(ctx, args).Err(); err != nil {
		return fmt.Errorf("write stream; error: %w", err)
	}

	return nil
}

// Claim checks if any messages exist on the Client's persistent stream
// that have not been acknowledged for the idle duration.
func (c *Client) Claim(ctx context.Context, idle time.Duration) (*Message, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	args := &redis.XAutoClaimArgs{
		Stream:   stream,
		Group:    c.group,
		Consumer: c.consumer,
		MinIdle:  idle,
		Count:    1,
		Start:    c.claimStart,
	}
	messages, start, err := c.rdb.XAutoClaim(ctx, args).Result()
	if err != nil {
		return nil, fmt.Errorf("auto claim; error: %w", err)
	}

	c.claimStart = start

	if len(messages) == 0 {
		return nil, ErrNoPending
	}

	return c.extractMessage(messages)
}

// Read reads a message from the persistent stream.
func (c Client) Read(ctx context.Context) (*Message, error) {
	args := &redis.XReadGroupArgs{
		Group:    c.group,
		Consumer: c.consumer,
		Streams:  []string{stream, ">"},
		Count:    1,
		Block:    24 * time.Hour,
		NoAck:    false,
	}

read:
	streams, err := c.rdb.XReadGroup(ctx, args).Result()
	if errors.Is(err, redis.Nil) {
		goto read
	}
	if err != nil {
		return nil, fmt.Errorf("read stream; error: %w", err)
	}

	if len(streams) != 1 {
		return nil, fmt.Errorf(
			"read stream; n: %d, error: %w",
			len(streams),
			ErrUnexpectedStreamCount,
		)
	}

	m, err := c.extractMessage(streams[0].Messages)
	if err != nil {
		return nil, err
	}

	c.logger.Debug(
		"read stream",
		zap.String("message-id", m.ID),
		zap.String("group", c.group),
		zap.String("consumer", c.consumer),
		zap.ByteString("payload", m.Payload),
	)
	return m, nil
}

// Ack acknowledges the passed Message. A Message should be acknowledged when
// it has been processed , and it is acceptable for the persistent stream to
// discard the contents.
func (c Client) Ack(ctx context.Context, m *Message) error {
	return c.rdb.XAck(ctx, stream, c.group, m.ID).Err()
}

func (c Client) extractMessage(messages []redis.XMessage) (*Message, error) {
	if len(messages) != 1 {
		return nil, fmt.Errorf(
			"unexpected stream message count; n: %d, error: %w",
			len(messages),
			ErrUnexpectedMessageCount,
		)
	}

	m := messages[0]

	str, ok := m.Values["payload"].(string)
	if !ok {
		return nil, errInvalidPayload
	}

	return &Message{
		ID:      m.ID,
		Payload: []byte(str),
	}, nil
}

type Message struct {
	ID      string
	Payload []byte
}
