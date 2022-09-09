package director

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/controller"
	"github.com/tjper/rustcron/cmd/cronman/model"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// WatchAndDirect instructs the Controller to collect upcoming server events and
// pass them to the EventsProcessor.
func (dir Director) WatchAndDirect(ctx context.Context) error {
	// acquire distributed lock, only one instance runs the controller
	if err := dir.distributedLock.Lock(ctx); err != nil {
		return fmt.Errorf("acquire director lock; %w", err)
	}
	defer dir.distributedLock.Unlock(ctx)

	dir.logger.Info("subscribed to refresh subject")
	sub := dir.redis.Subscribe(ctx, refreshSubj)
	defer func() {
		if err := sub.Close(); err != nil {
			dir.logger.Error("failed to close refresh subscription")
		}
	}()

	for {
		events, err := dir.store.ListActiveServerEvents(ctx)
		if err != nil {
			return fmt.Errorf("failed to list events; %w", err)
		}

		err = dir.schedule(ctx, sub.Channel(), events)
		if errors.Is(err, errDirectorRefresh) {
			continue
		}
		if err != nil {
			return fmt.Errorf("while scheduling events: %w", err)
		}
	}
}

var errDirectorRefresh = errors.New("director refresh received")

func (dir Director) schedule(
	ctx context.Context,
	refresh <-chan *redis.Message,
	events model.Events,
) error {
	scheduler := cron.New()

	if _, err := scheduler.AddFunc(
		"* * * * *",
		func() {
			if err := dir.controller.LiveServerRconForEach(ctx, dir.controller.CaptureServerInfo); err != nil {
				dir.logger.Error("while capturing live server info", zap.Error(err))
			}
		},
	); err != nil {
		dir.logger.Error("while scheduling server info capture", zap.Error(err))
	}

	if _, err := scheduler.AddFunc(
		"*/15 * * * *",
		func() {
			if err := dir.controller.LiveServerRconForEach(ctx, dir.controller.SayServerTimeRemaining); err != nil {
				dir.logger.Error("while saying server time remaining", zap.Error(err))
			}
		},
	); err != nil {
		dir.logger.Error("while scheduling say server time remaining", zap.Error(err))
	}

	for _, event := range events {
		this := event

		if _, err := scheduler.AddFunc(
			this.Schedule,
			func() {
				if this.Weekday != nil && !this.IsWeekDay(time.Now().UTC()) {
					return
				}
				dir.Direct(ctx, this)
			},
		); err != nil {
			dir.logger.Error(
				"schedule event",
				zap.Stringer("event-id", this.ID),
				zap.Error(err),
			)
		}
	}

	scheduler.Start()
	defer func() {
		ctx := scheduler.Stop()
		<-ctx.Done()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-refresh:
		return errDirectorRefresh
	}
}

func (dir Director) Direct(ctx context.Context, event model.Event) {
	var err error
	switch event.Kind {
	case model.EventKindStart:
		err = dir.startServer(ctx, event.ServerID)
	case model.EventKindStop:
		err = dir.stopServer(ctx, event.ServerID)
	case model.EventKindLive:
		err = dir.serverLive(ctx, event.ServerID)
	case model.EventKindMapWipe:
		err = dir.mapWipeServer(ctx, event.ServerID)
	case model.EventKindFullWipe:
		err = dir.fullWipeServer(ctx, event.ServerID)
	}
	if err != nil {
		dir.logger.Error(
			"directing event",
			zap.Stringer("event-id", event.ID),
			zap.Stringer("server-id", event.ServerID),
			zap.Error(err),
		)
	}
}

func (dir Director) startServer(ctx context.Context, serverID uuid.UUID) error {
	if _, err := dir.controller.StartServer(ctx, serverID); err != nil {
		return fmt.Errorf("start server; id: %s, error: %w", serverID, err)
	}
	return nil
}

func (dir Director) serverLive(ctx context.Context, serverID uuid.UUID) error {
	if _, err := dir.controller.MakeServerLive(ctx, serverID); err != nil {
		return fmt.Errorf("make server live; id: %s, error: %w", serverID, err)
	}
	return nil
}

func (dir Director) stopServer(ctx context.Context, serverID uuid.UUID) error {
	if _, err := dir.controller.StopServer(ctx, serverID); err != nil {
		return fmt.Errorf("stop server; id: %s, error: %w", serverID, err)
	}
	return nil
}

func (dir Director) wipeServer(ctx context.Context, serverID uuid.UUID, option controller.WipeOption) error {
	server, err := dir.controller.GetServer(ctx, serverID)
	if err != nil {
		return err
	}

	_, isLive := server.(*model.LiveServer)

	if isLive {
		if _, err := dir.controller.StopServer(ctx, serverID); err != nil {
			return err
		}
		defer func() {
			if _, err := dir.controller.StartServer(ctx, serverID); err != nil {
				dir.logger.Error("while restarting a wiped server", zap.Error(err))
				return
			}

			if _, err := dir.controller.MakeServerLive(ctx, serverID); err != nil {
				dir.logger.Error("while make a wiped server live", zap.Error(err))
				return
			}
		}()
	}

	if err := dir.controller.WipeServer(ctx, serverID, option); err != nil {
		return fmt.Errorf("while wiping server: %w", err)
	}
	return nil
}

func (dir Director) fullWipeServer(ctx context.Context, serverID uuid.UUID) error {
	seed := model.GenerateSeed()
	salt := model.GenerateSalt()
	return dir.wipeServer(ctx, serverID, controller.WipeFull(seed, salt))
}

func (dir Director) mapWipeServer(ctx context.Context, serverID uuid.UUID) error {
	seed := model.GenerateSeed()
	salt := model.GenerateSalt()
	return dir.wipeServer(ctx, serverID, controller.WipeMap(seed, salt))
}
