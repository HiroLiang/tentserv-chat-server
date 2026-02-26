// @title Goat-Server
// @version 1.0.0
// @description Server for my all Goat application
// @host dev.hiroliang.com
// @basePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HiroLiang/goat-server/internal/bootstrap"
	"github.com/HiroLiang/goat-server/internal/config"
	"github.com/HiroLiang/goat-server/internal/logger"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	_ "github.com/HiroLiang/goat-server/swag-docs"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("[WARN] No .env file found")
	}
}

func main() {

	// Initialize logger
	logger.Init()
	defer logger.Stop()

	// Load configuration
	if err := config.LoadConfig(config.Env("CONFIG_PATH", "./config")); err != nil {
		logger.Log.Fatal("load config error", zap.Error(err))
	}

	// Create application
	app := bootstrap.CreateApp()

	// Start application
	if err := app.Start(); err != nil {
		logger.Log.Fatal("start app failed", zap.Error(err))
	}

	// Wait for the stop signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Log.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Stop application
	app.Stop(ctx)
}
