package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/tjper/rustcron/cmd/payment/config"
	"github.com/tjper/rustcron/cmd/payment/db"
	"github.com/tjper/rustcron/cmd/payment/rest"
	"github.com/tjper/rustcron/cmd/payment/staging"
	"github.com/tjper/rustcron/cmd/payment/stream"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"
	istream "github.com/tjper/rustcron/internal/stream"
	"github.com/tjper/rustcron/internal/stripe"

	redisv8 "github.com/go-redis/redis/v8"
	"github.com/stripe/stripe-go/v72/client"
	"go.uber.org/zap"
)

func main() {
	os.Exit(run())
}

const (
	ecExit = iota
	_
	ecDatabaseConnection
	ecMigration
	ecRedisConnection
	ecStreamClient
)

func run() int {
	ctx := context.Background()
	cfg := config.Load()

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = logger.Sync() }()

	logger.Info("[Startup] Connecting to DB ...")
	dbconn, err := db.Open(cfg.DSN())
	if err != nil {
		logger.Error(
			"[Startup] Failed to initialize database connection.",
			zap.Error(err),
		)
		return ecDatabaseConnection
	}
	logger.Info("[Startup] Connected to DB.")

	logger.Info("[Startup] Migrating DB ...")
	if err := db.Migrate(dbconn, cfg.Migrations()); err != nil {
		logger.Error(
			"[Startup] Failed to migrate database model.",
			zap.Error(err),
		)
		return ecMigration
	}
	logger.Info("[Startup] Migrated DB.")

	logger.Info("[Startup] Connecting to Redis ...")
	rdb := redisv8.NewClient(&redisv8.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.RedisPassword(),
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Error(
			"[Startup] Failed to initialize Redis client.",
			zap.Error(err),
		)
		return ecRedisConnection
	}
	logger.Info("[Startup] Connected to Redis.")

	logger.Info("[Startup] Creating store client ...")
	store := db.NewStore(dbconn)
	logger.Info("[Startup] Created store client.")

	stripeClient := &client.API{}
	stripeClient.Init(cfg.StripeKey(), nil)

	logger.Info("[Startup] Creating session manager ...")
	sessionManager := session.NewManager(logger, rdb, 48*time.Hour)
	logger.Info("[Startup] Created session manager.")

	logger.Info("[Startup] Creating stripe clients ...")
	stripeWrapper := stripe.New(
		cfg.StripeWebhookSecret(),
		stripeClient.BillingPortalSessions,
		stripeClient.CheckoutSessions,
	)
	logger.Info("[Startup] Created stripe clients.")

	logger.Info("[Startup] Creating staging client ...")
	stagingClient := staging.NewClient(rdb)
	logger.Info("[Startup] Created staging client.")

	logger.Info("[Startup] Creating stream client ...")
	streamClient, err := istream.Init(ctx, logger, rdb, "payment")
	if err != nil {
		logger.Error(
			"[Startup] Failed to initialze stream client.",
			zap.Error(err),
		)
		return ecStreamClient
	}
	logger.Info("[Startup] Created stream client.")

	logger.Info("[Startup] Creating stream handler ...")
	streamHandler := stream.NewHandler(
		logger,
		stagingClient,
		store,
		streamClient,
	)
	go func() {
		if err := streamHandler.Launch(ctx); err != nil {
			logger.Error("[Startup] Failed to launch stream handler.", zap.Error(err))
		}
	}()
  logger.Info("[Startup] Created stream handler.")

	logger.Info("[Startup] Creating REST API ...")
	api := rest.NewAPI(
		logger,
		store,
		staging.NewClient(rdb),
		streamClient,
		stripeWrapper,
		ihttp.NewSessionMiddleware(
			logger,
			sessionManager,
		),
	)
	logger.Info("[Startup] Created REST API.")

	logger.Sugar().Infof("[Startup] payment API listening at :%d", cfg.Port())
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port()), api.Mux))
	return ecExit
}
