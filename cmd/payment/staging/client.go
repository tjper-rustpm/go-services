package staging

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

func NewClient(redis *redis.Client) *Client {
	return &Client{redis: redis}
}

type Client struct {
	redis    *redis.Client
	attempts int
}

func (c Client) StageCheckout(
	ctx context.Context,
	input Checkout,
	expiresAt time.Time,
) (string, error) {
	b, err := encode(input)
	if err != nil {
		return "", fmt.Errorf("encode checkout; error: %w", err)
	}

	var id uuid.UUID
	for i := 0; i <= c.attempts; i++ {
		id, err = uuid.NewRandom()
		if err != nil {
			return "", fmt.Errorf("random id; error: %w", err)
		}

		ok, err := c.redis.SetNX(ctx, keygen(id.String()), b, time.Until(expiresAt)).Result()
		if err != nil {
			return "", fmt.Errorf("stage checkout; id: %s, error: %w", id.String(), err)
		}
		if ok {
			break
		}
	}

	return id.String(), nil
}

func (c Client) FetchCheckout(
	ctx context.Context,
	id string,
) (*Checkout, error) {
	res, err := c.redis.Get(ctx, keygen(id)).Result()
	if err != nil {
		return nil, fmt.Errorf("fetch checkout; id: %s, error: %w", id, err)
	}

	var checkout Checkout
	if err := decode([]byte(res), &checkout); err != nil {
		return nil, fmt.Errorf("decode checkout; id: %s, error: %w", id, err)
	}

	return &checkout, nil
}

type Checkout struct {
	ServerID uuid.UUID
	UserID   uuid.UUID
}

// --- helpers ---

const (
	prefix = "rustpm-checkout-"
)

func keygen(id string) string {
	return fmt.Sprintf("%s%s", prefix, id)
}

func encode(obj interface{}) ([]byte, error) {
	return msgpack.Marshal(obj)
}

func decode(b []byte, obj interface{}) error {
	return msgpack.Unmarshal(b, obj)
}
