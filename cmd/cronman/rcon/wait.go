package rcon

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/logger"

	"go.uber.org/zap"
)

func NewWaiter(logger *zap.Logger) *Waiter {
	return &Waiter{
		logger: logger,
	}
}

// Waiter is responsible for functionality that allows clients to wait for
// Rcon servers to be in a particular state.
type Waiter struct {
	logger *zap.Logger
}

var (
	errDialing    = errors.New("error dialing websocket server")
	errReadyCheck = errors.New("error performing rcon ready check")
)

// WaitUntilReady waits until the specified URL is accepting connections. The
// wait arguement specifies the period of time to wait between retries. This
// process may be cancelled by cancelling context.Context. On success, a
// nil-error is returned.
func (w Waiter) UntilReady(
	ctx context.Context,
	url string,
	wait time.Duration,
) error {
	logger := w.logger.With(logger.ContextFields(ctx)...)

	readyCheck := func() error {
		// check if service may be dialed

		client, err := Dial(ctx, w.logger, url)
		if netErr, ok := err.(*net.OpError); ok && netErr.Op == "dial" {
			return errDialing
		}
		if err != nil {
			return err
		}
		defer client.Close()
		logger.Info("RCON server dialed")

		// check if rcon may be used
		info, err := client.ServerInfo(ctx)
		if err != nil {
			return fmt.Errorf("error performing ready check; %w", errReadyCheck)
		}
		logger.Info("RCON ready check passed", zap.Int("uptime", info.Uptime))

		return nil
	}

	logger.Info("waiting for RCON to be ready", zap.String("url", url), zap.Duration("retry", wait))
	for {
		time.Sleep(wait)
		err := readyCheck()
		if errors.Is(err, errDialing) {
			logger.Info("dialing...")
			continue
		}
		if errors.Is(err, errReadyCheck) {
			logger.Info("ready check...")
			continue
		}
		if err != nil {
			return fmt.Errorf("unable to wait for rcon to be ready; %w", err)
		}
		break
	}
	return nil
}
