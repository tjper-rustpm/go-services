package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/tjper/rustcron/cmd/payment/config"
	"github.com/tjper/rustcron/cmd/payment/controller"
	"github.com/tjper/rustcron/cmd/payment/db"
	"github.com/tjper/rustcron/cmd/payment/rest"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"

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

	stripe := &client.API{}
	stripe.Init(cfg.StripeKey(), nil)

	logger.Info("[Startup] Creating session manager ...")
	sessionManager := session.NewManager(logger, rdb)
	logger.Info("[Startup] Created session manager.")

	logger.Info("[Startup] Creating controller ...")
	ctrl := controller.New(
		logger,
		stripe.CheckoutSessions,
		stripe.BillingPortalSessions,
	)
	logger.Info("[Startup] Created controller.")

	logger.Info("[Startup] Creating REST API ...")
	api := rest.NewAPI(
		logger,
		ctrl,
		ihttp.NewSessionMiddleware(
			logger,
			sessionManager,
			48*time.Hour, // 2 days
		),
	)
	logger.Info("[Startup] Created REST API.")

	logger.Sugar().Infof("[Startup] payment API listening at :%d", cfg.Port())
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port()), api.Mux))
	return ecExit
}
