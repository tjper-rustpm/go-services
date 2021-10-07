package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/tjper/rustcron/cmd/cronman/config"
	"github.com/tjper/rustcron/cmd/cronman/controller"
	"github.com/tjper/rustcron/cmd/cronman/db"
	"github.com/tjper/rustcron/cmd/cronman/graph"
	"github.com/tjper/rustcron/cmd/cronman/graph/generated"
	loggerpkg "github.com/tjper/rustcron/cmd/cronman/logger"
	"github.com/tjper/rustcron/cmd/cronman/rcon"
	"github.com/tjper/rustcron/cmd/cronman/redis"
	"github.com/tjper/rustcron/cmd/cronman/server"
	rpmgraph "github.com/tjper/rustcron/internal/graph"
	rpmhttp "github.com/tjper/rustcron/internal/http"
	"github.com/tjper/rustcron/internal/session"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/go-chi/chi"
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
	sessionManager := session.NewManager(rdb)
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
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Info("[Startup] Controller.WatchAndDirect launched.")
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

	resolver := graph.NewResolver(
		logger,
		ctrl,
	)
	directive := rpmgraph.NewDirective(logger, sessionManager)
	srv := handler.NewDefaultServer(
		generated.NewExecutableSchema(
			generated.Config{
				Resolvers: resolver,
				Directives: generated.DirectiveRoot{
					HasRole:         directive.HasRole,
					IsAuthenticated: directive.IsAuthenticated,
				},
			},
		),
	)
	srv.Use(loggerpkg.NewTracer(logger))

	logger.Info("[Startup] Launching server ...")
	router := chi.NewRouter()
	router.Use(
		loggerpkg.Middleware(),
		rpmhttp.AccessMiddleware(),
	)
	router.Handle("/", playground.Handler("GraphQL playground", "/query"))
	router.Handle("/query", srv)

	logger.Sugar().Infof("[Startup] cronman endpoint listening at :%d", config.Port())
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Port()), router))
	return ecExit
}
