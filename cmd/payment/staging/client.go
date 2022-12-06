package staging

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

// NewClient creates a new Client instance.
func NewClient(redis *redis.Client) *Client {
	return &Client{redis: redis}
}

// Client manages the cache client and provides an API for interacting with
// staged checkouts.
type Client struct {
	redis *redis.Client
}

// errUnrecognizedCheckout indicates that a type other than UserCheckout or
// Checkout was passed where it is not handled.
var errUnrecognizedCheckout = errors.New("unrecognized checkout type")

// StageCheckout stores the checkout in staging store. The checkout will expire
// at the time passed via expiresAt. An identifier unique to staged checkout is
// returned as the first return value. If the checkout passed in not of type
// UserCheckout or Checkout, an error is returned.
func (c Client) StageCheckout(
	ctx context.Context,
	checkout interface{},
	expiresAt time.Time,
) (string, error) {
	switch checkout.(type) {
	case *Checkout:
		break
	case *UserCheckout:
		break
	default:
		return "", errUnrecognizedCheckout
	}

	b, err := encode(checkout)
	if err != nil {
		return "", fmt.Errorf("encode checkout; error: %w", err)
	}

	// NOTE: In the event there is a collision in Redis because the same UUID is
	// generated twice, retry. Retry upto ten times. The number of retries is
	// arbitrary and if collisions are still occurring something funny is going
	// on.
	const attempts = 10

	var id uuid.UUID
	for i := 0; i <= attempts; i++ {
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

// FetchCheckout retrieves a checkout specific to the passed id from the
// staging store.
func (c Client) FetchCheckout(
	ctx context.Context,
	id string,
) (interface{}, error) {
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

// Checkout is a Rustpm checkout associating a server with a Steam ID.
type Checkout struct {
	ServerID uuid.UUID
	SteamID  string
	PriceID  string
}

// UserCheckout is a Rustpm checkout associating a server, user, and a
// Steam ID.
type UserCheckout struct {
	Checkout
	UserID uuid.UUID
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
