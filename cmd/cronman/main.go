package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/config"
	"github.com/tjper/rustcron/cmd/cronman/controller"
	"github.com/tjper/rustcron/cmd/cronman/db"
	"github.com/tjper/rustcron/cmd/cronman/director"
	"github.com/tjper/rustcron/cmd/cronman/rcon"
	"github.com/tjper/rustcron/cmd/cronman/redis"
	"github.com/tjper/rustcron/cmd/cronman/rest"
	"github.com/tjper/rustcron/cmd/cronman/server"
	"github.com/tjper/rustcron/cmd/cronman/stream"
	"github.com/tjper/rustcron/internal/healthz"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"
	istream "github.com/tjper/rustcron/internal/stream"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	redisv8 "github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
	"gorm.io/gorm"
)

func main() {
	logger := newLogger()
	defer func() { _ = logger.Sync() }()

	store := newDBConnection(logger)
	migrateDB(logger, store)

	serverDirector := newServerDirector(context.Background(), logger)

	redisClient := newRedisClient(context.Background(), logger)
	streamClient := newStreamClient(context.Background(), logger, redisClient)
	sessionManager := session.NewManager(logger, redisClient, 48*time.Hour)
	rconHub := rcon.NewHub(logger)
	rconWaiter := rcon.NewWaiter(logger, time.Minute)
	directorNotifier := director.NewNotifier(logger, redisClient)
	streamHandler := stream.NewHandler(logger, store, streamClient, rconHub)

	ctrl := controller.New(
		logger,
		store,
		serverDirector,
		rconHub,
		rconWaiter,
		directorNotifier,
	)

	healthz := healthz.NewHTTP()
	sessionMiddleware := ihttp.NewSessionMiddleware(logger, sessionManager)
	api := rest.NewAPI(logger, ctrl, sessionMiddleware, healthz)

	srv := http.Server{
		Handler:      api.Mux,
		Addr:         fmt.Sprintf(":%d", config.Port()),
		ReadTimeout:  config.HTTPReadTimeout(),
		WriteTimeout: config.HTTPWriteTimeout(),
	}

	// Waitgroup to ensure all supporting goroutines close properly on
	// application close.
	var wg sync.WaitGroup
	defer wg.Wait()

	// Root context to passed to child goroutines. Context will be cancelled if
	// SIGTERM or SIGINT received. Context will be cancelled if error occurs that
	// cannot be recovered from.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Launch goroutine that listens for SIGTERM and SIGINT. In the event either
	// occurs, cancel the context.
	signalc := make(chan os.Signal, 1)
	signal.Notify(signalc, unix.SIGTERM, unix.SIGINT)

	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case <-ctx.Done():
			return
		case <-signalc:
			cancel()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		err := streamHandler.Launch(ctx)
		if errors.Is(err, context.Canceled) {
			return
		}
		if err != nil {
			logger.Error("[Startup] Failed to process stream handler.", zap.Error(err))
			cancel()
		}
	}()

	if config.DirectorEnabled() {
		director := director.New(logger, redis.New(redisClient), store, ctrl)

		// Launch director.WatchAndDirect in separate goroutine. When goroutine
		// closes decrement WaitGroup. If director.WatchAndDirect returns an
		// unexpected error, log and cancel root context.
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := director.WatchAndDirect(ctx)
			if errors.Is(err, context.Canceled) {
				logger.Info("[Startup] Another process cancelled Controller.WatchAndDirect.")
				return
			}
			if err != nil {
				logger.Error("[Startup] Controller failed to WatchAndDirect.", zap.Error(err))
				cancel()
			}
		}()
	}

	// Wait for root context to be cancelled in a separate goroutine. Until then
	// indicate that service is healthy via healthz package. When separate
	// goroutine cancels context, update health to "sick" and initiate
	// http.Server.Shutdown to gracefully shutdown http API.
	healthz.Healthy()
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer healthz.Sick()
		<-ctx.Done()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("[Startup] Failed to correctly shutdown cronman.", zap.Error(err))
		}
	}()

	logger.Sugar().Infof("cronman API listening at :%d", config.Port())
	err := srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return
	}
	if err != nil {
		logger.Panic("[Startup] Failed to listen and serve cronman API.", zap.Error(err))
	}
}

func newLogger() *zap.Logger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	return logger
}

func newDBConnection(logger *zap.Logger) *gorm.DB {
	dbconn, err := db.Open(config.DSN())
	if err != nil {
		logger.Panic("[Startup] Failed to establish DB connection.", zap.Error(err))
	}
	return dbconn
}

func migrateDB(logger *zap.Logger, dbconn *gorm.DB) {
	if err := db.Migrate(dbconn, config.Migrations()); err != nil {
		logger.Panic("[Startup] Failed to migrate DB.", zap.Error(err))
	}
}

func newRedisClient(ctx context.Context, logger *zap.Logger) *redisv8.Client {
	client := redisv8.NewClient(&redisv8.Options{
		Addr:     config.RedisAddr(),
		Password: config.RedisPassword(),
	})
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Panic("[Startup] Failed to initialize Redis client.", zap.Error(err))
	}
	return client
}

func newStreamClient(ctx context.Context, logger *zap.Logger, redisClient *redisv8.Client) *istream.Client {
	streamClient, err := istream.Init(ctx, logger, redisClient, "cronman")
	if err != nil {
		logger.Panic("[Startup] Failed to initialze stream client.", zap.Error(err))
	}
	return streamClient
}

func newServerDirector(ctx context.Context, logger *zap.Logger) *controller.ServerDirector {
	awscfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Panic("[Startup] Failed to acquire AWS config.")
	}
	usEastEC2 := ec2.NewFromConfig(awscfg, func(opts *ec2.Options) {
		opts.Region = "us-east-1"
	})
	logger.Info("[Startup] Loaded us-east-1 client.")
	usWestEC2 := ec2.NewFromConfig(awscfg, func(opts *ec2.Options) {
		opts.Region = "us-west-1"
	})
	logger.Info("[Startup] Loaded us-west-1 client.")
	euCentralEC2 := ec2.NewFromConfig(awscfg, func(opts *ec2.Options) {
		opts.Region = "eu-central-1"
	})

	return controller.NewServerDirector(
		server.NewManager(logger, usEastEC2),
		server.NewManager(logger, usWestEC2),
		server.NewManager(logger, euCentralEC2),
	)
}
