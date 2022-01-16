package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/tjper/rustcron/cmd/cronman/config"
	"github.com/tjper/rustcron/cmd/cronman/controller"
	"github.com/tjper/rustcron/cmd/cronman/db"
	"github.com/tjper/rustcron/cmd/cronman/rcon"
	"github.com/tjper/rustcron/cmd/cronman/redis"
	"github.com/tjper/rustcron/cmd/cronman/rest"
	"github.com/tjper/rustcron/cmd/cronman/server"
	ihttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	redisv8 "github.com/go-redis/redis/v8"
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
	ecAwsConfig
	ecServerAPI
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

	logger.Info("[Startup] Loading AWS configuration ...")
	awscfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error("[Startup] Failed to acquire AWS config.")
		return ecAwsConfig
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
	logger.Info("[Startup] Loaded eu-central-1 client.")
	logger.Info("[Startup] Loaded AWS configuration.")

	logger.Info("[Startup] Creating session manager ...")
	sessionManager := session.NewManager(logger, rdb)
	logger.Info("[Startup] Created session manager.")

	logger.Info("[Startup] Creating controller ...")
	ctrl := controller.New(
		logger,
		redis.New(rdb),
		db.NewStore(logger, dbconn),
		controller.NewServerDirector(
			server.NewManager(logger, usEastEC2),
			server.NewManager(logger, usWestEC2),
			server.NewManager(logger, euCentralEC2),
		),
		controller.NewHub(logger),
		rcon.NewWaiter(logger),
		controller.NewNotifier(logger, rdb),
	)
	logger.Info("[Startup] Created controller.")

	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if config.DirectorEnabled() {
		logger.Info("[Startup] Creating director ...")
		wg.Add(1)

		go func() {
			defer wg.Done()
			logger.Info("[Startup] Created director.")
			err := ctrl.WatchAndDirect(ctx)
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

	logger.Info("[Startup] Launching server ...")
	srv := http.Server{
		Handler:      api.Mux,
		Addr:         fmt.Sprintf(":%d", config.Port()),
		ReadTimeout:  config.HttpReadTimeout(),
		WriteTimeout: config.HttpWriteTimeout(),
	}
	logger.Sugar().Infof("[Startup] cronman API listening at :%d", config.Port())
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("[Startup] Failed to listen and serve server API.", zap.Error(err))
		return ecServerAPI
	}
	return ecExit
}
