package context

import (
	"context"
	"os"
	"os/signal"
)

// WithSignal creates a new context that may be cancelled by signalling to one
// of the passed signals. The cancel return value should be called to release
// this function's resources once it is no longer in use.
func WithSignal(ctx context.Context, signals ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, signals...)
	go func() {
		defer signal.Stop(ch)

		select {
		case <-ctx.Done():
			return
		case <-ch:
			cancel()
		}
	}()
	return ctx, cancel
}
