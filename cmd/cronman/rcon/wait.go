package rcon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/logger"

	"go.uber.org/zap"
)

func NewWaiter(logger *zap.Logger, interval time.Duration) *Waiter {
	return &Waiter{
		logger:   logger,
		interval: interval,
	}
}

// Waiter is responsible for functionality that allows clients to wait for
// Rcon servers to be in a particular state.
type Waiter struct {
	logger   *zap.Logger
	interval time.Duration
}

var (
	errDialing    = errors.New("error dialing websocket server")
	errReadyCheck = errors.New("error performing rcon ready check")
)

// UntilReady waits until the specified URL is accepting connections. The
// wait argument specifies the period of time to wait between retries. This
// process may be cancelled by cancelling context.Context. On success, a
// nil-error is returned.
func (w Waiter) UntilReady(
	ctx context.Context,
	url string,
) error {
	logger := w.logger.With(logger.ContextFields(ctx)...)

	readyCheck := func() error {
		// check if service may be dialed

		client, err := Dial(ctx, w.logger, url)
		netErr := new(net.OpError)
		if ok := errors.As(err, &netErr); ok && netErr.Op == "dial" {
			return errDialing
		}
		if err != nil {
			return err
		}
		defer client.Close()

		// check if rcon may be used
		info, err := client.ServerInfo(ctx)
		if err != nil {
			return fmt.Errorf("error performing ready check; %w", errReadyCheck)
		}
		logger.Info("RCON ready check passed", zap.Int("uptime", info.Uptime))

		return nil
	}

	logger.Info("waiting for RCON to be ready", zap.String("url", url), zap.Duration("retry", w.interval))
	for {
		err := readyCheck()
		if errors.Is(err, io.ErrUnexpectedEOF) {
			goto retry
		}
		if errors.Is(err, errDialing) {
			goto retry
		}
		if errors.Is(err, errReadyCheck) {
			goto retry
		}
		if err != nil {
			return fmt.Errorf("unable to wait for rcon to be ready; %w", err)
		}
		break

	retry:
		time.Sleep(w.interval)
		continue
	}
	return nil
}
