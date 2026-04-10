// @title Tentserv Chat Server
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

	"github.com/HiroLiang/tentserv-chat-server/internal/bootstrap"
	"github.com/HiroLiang/tentserv-chat-server/internal/config"
	"github.com/HiroLiang/tentserv-chat-server/internal/logger"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	_ "github.com/HiroLiang/tentserv-chat-server/swag-docs"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("[WARN] No .env file found")
	}
}

// [EN] main: server entry point. Initializes logger and config, creates and starts the app,
//
//	then blocks on SIGTERM/SIGINT for graceful shutdown (5s timeout).
//
// [中] main：伺服器入口。初始化 logger 與設定，建立並啟動 App，
//
//	監聽 SIGTERM/SIGINT 訊號後執行 5 秒超時優雅關機。
//
// [日] main：サーバーエントリ。logger と設定を初期化し、App を作成・起動した後、
//
//	SIGTERM/SIGINT を待機して 5 秒タイムアウトのグレースフルシャットダウンを実行する。
func main() {

	// Initialize logger
	logger.Init()
	defer logger.Stop()

	// Load configuration
	if err := config.LoadConfig(config.Env("CONFIG_PATH", "./config")); err != nil {
		logger.Log.Fatal("load config error", zap.Error(err))
	}
	logger.Log.Info("config loaded")

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
