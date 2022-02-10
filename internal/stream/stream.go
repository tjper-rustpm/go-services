package stream

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

var (
	ErrUnexpectedStreamCount  = errors.New("unexpected stream count")
	ErrUnexpectedMessageCount = errors.New("unexpected message count")
)

const (
	stream = "interserviceEventStream"
	start  = "0"
	maxlen = 1000
)

func Init(ctx context.Context, rdb *redis.Client, group string) (*Client, error) {
	if err := rdb.XGroupCreateMkStream(ctx, stream, group, start).Err(); err != nil {
		return nil, fmt.Errorf("initializing stream; error: %w", err)
	}
	return &Client{
		rdb:      rdb,
		group:    group,
		consumer: uuid.New().String(),
	}, nil
}

type Client struct {
	rdb *redis.Client

	group    string
	consumer string
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

func (c Client) Read(ctx context.Context) (*Message, error) {
	args := &redis.XReadGroupArgs{
		Group:    c.group,
		Consumer: c.consumer,
		Streams:  []string{stream},
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
	if len(streams[0].Messages) != 1 {
		return nil, fmt.Errorf(
			"unexpected steam message count; n: %d, error: %w",
			len(streams[0].Messages),
			ErrUnexpectedMessageCount,
		)
	}

	m := streams[0].Messages[0]

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
