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

	"github.com/tjper/rustcron/cmd/payment/config"
	"github.com/tjper/rustcron/cmd/payment/db"
	"github.com/tjper/rustcron/cmd/payment/rest"
	"github.com/tjper/rustcron/cmd/payment/staging"
	"github.com/tjper/rustcron/cmd/payment/stream"
	"github.com/tjper/rustcron/internal/healthz"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"
	istream "github.com/tjper/rustcron/internal/stream"
	"github.com/tjper/rustcron/internal/stripe"

	redisv8 "github.com/go-redis/redis/v8"
	"github.com/stripe/stripe-go/v72/client"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	logger := newLogger()
	defer func() { _ = logger.Sync() }()

	dbconn := newDBConnection(logger, cfg)
	migrateDB(logger, cfg, dbconn)
	store := db.NewStore(dbconn)

	redisClient := newRedisClient(context.Background(), cfg, logger)
	streamClient := newStreamClient(context.Background(), logger, redisClient)
	sessionManager := session.NewManager(logger, redisClient, 48*time.Hour)
	stagingClient := staging.NewClient(redisClient)
	streamHandler := stream.NewHandler(logger, stagingClient, store, streamClient)

	stripeClient := &client.API{}
	stripeClient.Init(cfg.StripeKey(), nil)
	stripeWrapper := stripe.New(
		cfg.StripeWebhookSecret(),
		stripeClient.BillingPortalSessions,
		stripeClient.CheckoutSessions,
	)

	healthz := healthz.NewHTTP()
	api := rest.NewAPI(
		logger,
		store,
		staging.NewClient(redisClient),
		streamClient,
		stripeWrapper,
		ihttp.NewSessionMiddleware(
			logger,
			sessionManager,
		),
		healthz,
	)

	srv := http.Server{
		Handler:      api.Mux,
		Addr:         fmt.Sprintf(":%d", cfg.Port()),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Waitgroup to ensure all supporting goroutines close properly on
	// application close.
	var wg sync.WaitGroup

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

	// Wait for root context to be cancelled in a separate goroutine. Until then
	// indicate that service is healthy via healthz package. When seperate
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

	logger.Sugar().Infof("payment API listening at :%d", cfg.Port())
	err := srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return
	}
	if err != nil {
		logger.Panic("[Startup] Failed to listen and serve payment API.", zap.Error(err))
	}
}

func newLogger() *zap.Logger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	return logger
}

func newDBConnection(logger *zap.Logger, cfg *config.Config) *gorm.DB {
	dbconn, err := db.Open(cfg.DSN())
	if err != nil {
		logger.Panic("[Startup] Failed to establish DB connection.", zap.Error(err))
	}
	return dbconn
}

func migrateDB(logger *zap.Logger, cfg *config.Config, dbconn *gorm.DB) {
	if err := db.Migrate(dbconn, cfg.Migrations()); err != nil {
		logger.Panic("[Startup] Failed to migrate DB.", zap.Error(err))
	}
}

func newRedisClient(ctx context.Context, cfg *config.Config, logger *zap.Logger) *redisv8.Client {
	client := redisv8.NewClient(&redisv8.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.RedisPassword(),
	})
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Panic("[Startup] Failed to initialize Redis client.", zap.Error(err))
	}
	return client
}

func newStreamClient(ctx context.Context, logger *zap.Logger, redisClient *redisv8.Client) *istream.Client {
	streamClient, err := istream.Init(ctx, logger, redisClient, "payment")
	if err != nil {
		logger.Panic("[Startup] Failed to initialze stream client.", zap.Error(err))
	}
	return streamClient
}
