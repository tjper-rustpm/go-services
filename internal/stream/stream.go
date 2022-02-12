package stream

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

var (
	ErrUnexpectedStreamCount  = errors.New("unexpected stream count")
	ErrUnexpectedMessageCount = errors.New("unexpected message count")
	ErrNoPending              = errors.New("no pending messages")
)

const (
	stream = "interserviceEventStream"
	start  = "0"
	maxlen = 20000
)

func Init(ctx context.Context, rdb *redis.Client, group string) (*Client, error) {
	if err := rdb.XGroupCreateMkStream(ctx, stream, group, start).Err(); err != nil {
		return nil, fmt.Errorf("initializing stream; error: %w", err)
	}
	return &Client{
		rdb:        rdb,
		group:      group,
		consumer:   uuid.New().String(),
		mutex:      new(sync.RWMutex),
		claimStart: "0-0",
	}, nil
}

type Client struct {
	rdb *redis.Client

	group    string
	consumer string

	mutex      *sync.RWMutex
	claimStart string
}

func (c Client) Write(ctx context.Context, kv map[string]interface{}) error {
	args := &redis.XAddArgs{
		Stream:       stream,
		MaxLenApprox: maxlen,
		ID:           "*",
		Values:       kv,
	}
	if err := c.rdb.XAdd(ctx, args).Err(); err != nil {
		return fmt.Errorf("write stream; error: %w", err)
	}

	return nil
}

func (c *Client) Claim(ctx context.Context, idle time.Duration) (*Message, error) {
	c.mutex.RLock()
	start := c.claimStart
	c.mutex.RUnlock()

	args := &redis.XAutoClaimArgs{
		Stream:   stream,
		Group:    c.group,
		Consumer: c.consumer,
		MinIdle:  idle,
		Count:    1,
		Start:    start,
	}
	messages, start, err := c.rdb.XAutoClaim(ctx, args).Result()
	if err != nil {
		return nil, fmt.Errorf("auto claim; error: %w", err)
	}

	c.mutex.Lock()
	c.claimStart = start
	c.mutex.Unlock()

	if len(messages) == 0 {
		return nil, ErrNoPending
	}

	return c.extractMessage(messages)
}

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

	return c.extractMessage(streams[0].Messages)
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

	return &Message{
		ID:     m.ID,
		Values: m.Values,
		ackFn: func(ctx context.Context) error {
			return c.rdb.XAck(ctx, stream, c.group, m.ID).Err()
		},
	}, nil
}

type Message struct {
	ID     string
	Values map[string]interface{}
	ackFn  func(context.Context) error
}

func (m Message) Ack(ctx context.Context) error {
	return m.ackFn(ctx)
}
