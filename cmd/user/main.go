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

	"github.com/tjper/rustcron/cmd/user/admin"
	"github.com/tjper/rustcron/cmd/user/config"
	"github.com/tjper/rustcron/cmd/user/controller"
	"github.com/tjper/rustcron/cmd/user/db"
	"github.com/tjper/rustcron/cmd/user/rest"
	"github.com/tjper/rustcron/internal/email"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"
	"golang.org/x/sys/unix"
	"gorm.io/gorm"

	redisv8 "github.com/go-redis/redis/v8"
	"github.com/mailgun/mailgun-go/v4"
	"go.uber.org/zap"
)

func main() {
	logger := newLogger()
	defer func() { _ = logger.Sync() }()

	dbconn := newDBConnection(logger)
	migrateDB(logger, dbconn)

	redisClient := newRedisClient(context.Background(), logger)
	sessionManager := session.NewManager(logger, redisClient, 48*time.Hour)

	mailgunClient := mailgun.NewMailgun(config.MailgunDomain(), config.MailgunAPIKey())
	store := db.NewStore(logger, dbconn)
	emailer := email.NewMailgunEmailer(mailgunClient, config.MailgunHost())
	admins := admin.NewAdminSet(config.Admins())

	ctrl := controller.New(store, emailer, admins)

	api := rest.NewAPI(
		logger,
		ctrl,
		ihttp.CookieOptions{
			Domain:   config.CookieDomain(),
			Secure:   config.CookieSecure(),
			SameSite: config.CookieSameSite(),
		},
		sessionManager,
		ihttp.NewSessionMiddleware(
			logger,
			sessionManager,
		),
	)

	srv := http.Server{
		Handler:      api.Mux,
		Addr:         fmt.Sprintf(":%d", config.Port()),
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

	// Wait for root context to close in separate goroutine. When goroutine
	// closes call http.Server.Shutdown to gracefully shutdown http API.
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("[Startup] Failed to correctly shutdown cronman.", zap.Error(err))
		}
	}()

	logger.Sugar().Infof("user API listening at :%d", config.Port())
	err := srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return
	}
	if err != nil {
		logger.Panic("[Startup] Failed to listen and serve user API.", zap.Error(err))
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
