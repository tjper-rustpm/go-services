package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/tjper/rustcron/cmd/user/admin"
	"github.com/tjper/rustcron/cmd/user/config"
	"github.com/tjper/rustcron/cmd/user/controller"
	"github.com/tjper/rustcron/cmd/user/db"
	"github.com/tjper/rustcron/cmd/user/rest"
	"github.com/tjper/rustcron/internal/email"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"

	redisv8 "github.com/go-redis/redis/v8"
	"github.com/mailgun/mailgun-go/v4"
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

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = logger.Sync() }()

	logger.Info("[Startup] Connecting to DB ...")
	dbconn, err := db.Open(config.DSN())
	if err != nil {
		logger.Error(
			"[Startup] Failed to initialize database connection.",
			zap.Error(err),
		)
		return ecDatabaseConnection
	}
	logger.Info("[Startup] Connected to DB.")

	logger.Info("[Startup] Migrating DB ...")
	if err := db.Migrate(dbconn, config.Migrations()); err != nil {
		logger.Error(
			"[Startup] Failed to migrate database model.",
			zap.Error(err),
		)
		return ecMigration
	}
	logger.Info("[Startup] Migrated DB.")

	logger.Info("[Startup] Connecting to Redis ...")
	rdb := redisv8.NewClient(&redisv8.Options{
		Addr:     config.RedisAddr(),
		Password: config.RedisPassword(),
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Error(
			"[Startup] Failed to initialize Redis client.",
			zap.Error(err),
		)
		return ecRedisConnection
	}
	logger.Info("[Startup] Connected to Redis.")

	logger.Info("[Startup] Creating emailer ...")
	mg := mailgun.NewMailgun(config.MailgunDomain(), config.MailgunAPIKey())
	logger.Info("[Startup] Created emailer.")

	logger.Info("[Startup] Creating session manager ...")
	sessionManager := session.NewManager(rdb)
	logger.Info("[Startup] Created session manager.")

	logger.Info("[Startup] Creating controller ...")
	ctrl := controller.New(
		sessionManager,
		db.NewStore(logger, dbconn),
		email.NewMailgunEmailer(mg, config.MailgunHost()),
		admin.NewAdminSet(config.Admins()),
	)
	logger.Info("[Startup] Created controller.")

	logger.Info("[Startup] Creating REST API ...")
	api := rest.NewAPI(
		logger,
		ctrl,
		ihttp.CookieOptions{
			Domain:   config.CookieDomain(),
			Secure:   config.CookieSecure(),
			SameSite: config.CookieSameSite(),
		},
	)
	logger.Info("[Startup] Created REST API.")

	logger.Info("[Startup] Launching server ...")

	logger.Sugar().Infof("[Startup] user API listening at :%d", config.Port())
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port()), api.Mux))
	return ecExit
}
