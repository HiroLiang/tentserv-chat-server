package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/config"
	"github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/persistence/database"
	infraPush "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/push"
	redis2 "github.com/HiroLiang/tentserv-chat-server/internal/infrastructure/redis"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type App struct {
	Server           *http.Server
	Redis            *redis.Client
	DataSources      *database.DataSources
	schedulerCancel  context.CancelFunc
}

func CreateApp() *App {
	return &App{}
}

func (app *App) Start() error {
	start := time.Now()
	var err error

	// Initialize Redis
	redisConfig := &redis2.ClientConfig{
		Addr:     config.App().Redis.Addr,
		Password: config.App().Redis.Password,
		DB:       config.App().Redis.DB,
	}
	app.Redis, err = redis2.InitRedis(redisConfig)
	if err != nil {
		return err
	}

	// Initialize DataSource
	databaseConfig := database.BuildDatabaseConfigs(config.App().Database)
	app.DataSources, err = database.NewDataSources(databaseConfig)
	if err != nil {
		return err
	}

	// Init all dependencies
	dependencies, err := BuildDeps(app.Redis, app.DataSources)
	if err != nil {
		return err
	}

	// Start retry scheduler
	schedulerCtx, schedulerCancel := context.WithCancel(context.Background())
	app.schedulerCancel = schedulerCancel
	go infraPush.NewRetryScheduler(dependencies.DeliveryQueueRepo, dependencies.Hub).Start(schedulerCtx)

	// build use cases
	useCases := BuildUseCases(dependencies)

	// Start api server
	app.Server = NewServer(
		":"+config.Env("SERVER_PORT", "8080"),
		useCases,
		dependencies,
	)

	go func() {
		if err := app.Server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error("server error", zap.Error(err))
		}
	}()

	logger.Log.Info(
		"application boot completed",
		zap.Duration("boot_time", time.Since(start)),
	)
	return nil
}

func (app *App) Stop(ctx context.Context) {

	// 0. Stop scheduler
	if app.schedulerCancel != nil {
		app.schedulerCancel()
	}

	// 1. Stop accepting requests
	if app.Server != nil {
		if err := app.Server.Shutdown(ctx); err != nil {
			logger.Log.Error("server shutdown error", zap.Error(err))
		}
	}

	// 2. Close DB
	if app.DataSources != nil {
		app.DataSources.CloseAllDBs()
	}

	// 3. Close Redis
	if app.Redis != nil {
		_ = app.Redis.Close()
	}

	logger.Log.Info("application stopped")
}
